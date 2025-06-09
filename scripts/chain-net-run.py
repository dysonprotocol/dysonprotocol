#!/usr/bin/env python3
import json
import click
import os
import signal # For graceful shutdown
import shutil
import subprocess
import tomlkit
import tomlkit.items # For Table type hint
from concurrent.futures import ProcessPoolExecutor, as_completed
from typing import Dict, Any, List, Tuple, Union, Optional, TextIO
import sys
import time
import shlex
import threading
import requests # For RPC queries
import re
import errno

# Global constants
DEFAULT_KEYRING_BACKEND = "test"

# Track running dysond processes if started by this script
running_processes: List[subprocess.Popen] = []

class CommandInterruptedError(Exception):
    """Custom exception for when a command is interrupted by an exit event."""
    pass

def stop_all_started_nodes():
    """Attempts to stop all dysond processes started by this script."""
    if not running_processes:
        return
    click.echo("\nStopping all started processes (dysond nodes, Hermes, etc.)...")
    for process in running_processes:
        if process.poll() is None:  # If process is still running
            try:
                # Send SIGTERM to the process group
                if hasattr(os, 'killpg') and process.pid is not None:
                    os.killpg(os.getpgid(process.pid), signal.SIGTERM)
                else:
                    process.terminate()
                click.echo(f"Sent SIGTERM to process {process.pid}")
            except ProcessLookupError:
                click.echo(f"Process {process.pid} not found.")
            except Exception as e:
                click.echo(f"Failed to send SIGTERM to process {process.pid}: {e}")
    
    # Wait for processes to terminate
    for process in running_processes:
        if process.pid is None: continue
        try:
            process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            click.echo(f"Process {process.pid} did not terminate gracefully, sending SIGKILL.")
            if hasattr(os, 'killpg') and process.pid is not None:
                try:
                    os.killpg(os.getpgid(process.pid), signal.SIGKILL)
                except ProcessLookupError: # process might have terminated between poll and killpg
                    pass 
            else:
                process.kill()
        except Exception as e:
            click.echo(f"Error waiting for process {process.pid} to terminate: {e}")
    click.echo("Node stopping sequence complete.")

def signal_handler(sig, frame):
    """Handle Ctrl+C to gracefully stop all running nodes."""
    stop_all_started_nodes()
    # It might be good to exit with a specific code or let Python's default ^C behavior work after cleanup.
    # For now, just ensure cleanup and then let it exit.
    click.echo("Exiting due to signal.")
    sys.exit(sig) # Exit with the signal number

# Register signal handler early
signal.signal(signal.SIGINT, signal_handler)
signal.signal(signal.SIGTERM, signal_handler)

def run_command(
    cmd_list: List[str],
    cwd: Optional[str] = None,
    env: Optional[Dict[str, str]] = None,
    check: bool = True,
    exit_event: Optional[threading.Event] = None,
    poll_interval: float = 0.1
) -> Tuple[str, str, int]:
    """
    Run a command and return its stdout, stderr, and return code.
    Can be interrupted by an exit_event.
    """
    effective_cwd = cwd or os.getcwd()
    print(f"Running: {' '.join(cmd_list)} (cwd: {effective_cwd}){f' (with custom env: {env})' if env else ''}")
    
    process_env = os.environ.copy()
    if env:
        process_env.update(env)
    
    process = subprocess.Popen(
        cmd_list,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        cwd=effective_cwd,
        env=process_env,
        preexec_fn=os.setsid if hasattr(os, 'setsid') else None # For process group killing
    )

    while process.poll() is None:
        if exit_event and exit_event.is_set():
            click.echo(f"Exit event detected, interrupting command: {' '.join(cmd_list)}")
            try:
                if hasattr(os, 'killpg') and process.pid is not None:
                    os.killpg(os.getpgid(process.pid), signal.SIGTERM)
                    time.sleep(poll_interval) # Give it a moment to terminate
                    if process.poll() is None: # Still running
                        os.killpg(os.getpgid(process.pid), signal.SIGKILL)
                else:
                    process.terminate()
                    time.sleep(poll_interval)
                    if process.poll() is None:
                        process.kill()
            except ProcessLookupError:
                pass # Process already terminated
            except OSError as e:
                # Handle ESRCH (No such process), which can happen if process dies quickly
                if e.errno != errno.ESRCH:
                    click.echo(f"Error terminating process group for {' '.join(cmd_list)}: {e}", err=True)
            except Exception as e: # Catch any other error during termination
                click.echo(f"Exception during process termination for {' '.join(cmd_list)}: {e}", err=True)
            
            # After attempting termination, wait briefly for streams to close.
            # stdout, stderr might be incomplete if process was killed forcefully.
            stdout_val, stderr_val = process.communicate(timeout=1) # Short timeout
            raise CommandInterruptedError(f"Command interrupted: {' '.join(cmd_list)}. Stdout: {stdout_val.strip()}, Stderr: {stderr_val.strip()}")

        time.sleep(poll_interval)

    stdout_val, stderr_val = process.communicate()
    return_code = process.returncode

    if check and return_code != 0:
        # This behavior is similar to subprocess.run(check=True)
        # Create a CalledProcessError-like message
        error_message = (
            f"Command failed: {' '.join(cmd_list)}\n"
            f"Return code: {return_code}\n"
        )
        if stdout_val:
            error_message += f"STDOUT: {stdout_val.strip()}\n"
        if stderr_val:
            error_message += f"STDERR: {stderr_val.strip()}"
        
        # Mimic CalledProcessError by raising a generic exception with details
        # Or, you could define a custom exception similar to CalledProcessError
        raise subprocess.CalledProcessError(return_code, cmd_list, output=stdout_val, stderr=stderr_val)

    return stdout_val.strip(), stderr_val.strip(), return_code

def init_node_and_get_peer_info(node_spec: Dict[str, Any], chain_id_str: str, global_config: Dict[str, Any], force: bool) -> Dict[str, Any]:
    """ 
    Initializes a node using 'dysond init', captures its node_id, and constructs the peer string.
    Handles directory creation/forcing. Halts on any error.
    """
    dysond_bin = global_config["dysond_bin"]
    node_home = node_spec["home"]
    moniker = node_spec["moniker"]
    default_denom = global_config["default_denom"]
    p2p_port = node_spec["ports"]["p2p"] # Expect p2p port to be in node_spec["ports"]

    print(f"[{moniker}] Initializing and getting node ID...")

    if os.path.exists(node_home):
        if force:
            print(f"[{moniker}] Force removing existing directory: {node_home}")
            shutil.rmtree(node_home)
        else:
            # Halt if directory exists and --force is not used.
            raise FileExistsError(f"Node directory {node_home} already exists. Use --force to remove and re-initialize.")
    
    os.makedirs(node_home, exist_ok=True)
    print(f"[{moniker}] Created directory: {node_home}")

    init_cmd = [dysond_bin, "init", moniker, "--chain-id", chain_id_str, "--default-denom", default_denom, "-o", "--home", node_home]
    # dysond init -o outputs JSON to stdout (or stderr based on recent observation)
    # Let's capture both and try to parse JSON from stdout first, then stderr if stdout is empty or fails.
    init_stdout, init_stderr, _ = run_command(init_cmd)
    
    init_output_json_str = ""
    if init_stdout.strip():
        init_output_json_str = init_stdout
    elif init_stderr.strip(): # Check stderr if stdout is empty
        print(f"[{moniker}] 'dysond init -o' output JSON to stderr, not stdout. Using stderr content.")
        init_output_json_str = init_stderr
    
    if not init_output_json_str:
        raise Exception(f"[{moniker}] 'dysond init -o' produced no JSON output on stdout or stderr. Stdout: '{init_stdout}', Stderr: '{init_stderr}'")

    try:
        init_output_json = json.loads(init_output_json_str)
        node_id = init_output_json["node_id"]
    except (json.JSONDecodeError, KeyError) as e:
        raise Exception(f"[{moniker}] Failed to parse node_id from 'dysond init' output. Error: {e}. Output was: {init_output_json_str}")

    peer_string = f"{node_id}@127.0.0.1:{p2p_port}" # Assuming peers are on 127.0.0.1
    actual_config_dir = os.path.join(node_home, "config")

    print(f"[{moniker}] Initialized. Node ID: {node_id}, Peer String: {peer_string}")
    
    return {
        "node_num": node_spec["node_num"],
        "moniker": moniker,
        "node_home": node_home,
        "peer_string": peer_string,
        "actual_config_dir": actual_config_dir,
        # Pass through original spec for later merging/use
        "original_spec": node_spec 
    }


def complete_node_setup(full_node_config: Dict[str, Any], chain_id_str: str, global_config: Dict[str, Any]) -> Dict[str, Any]:
    """
    Completes node setup: client config, keys, TOML files (with persistent peers).
    Assumes 'dysond init' has already been run. Halts on any error.
    """
    dysond_bin = global_config["dysond_bin"]
    node_home = full_node_config["home"]
    moniker = full_node_config["moniker"]
    # actual_config_dir is determined directly from node_home
    actual_config_dir = os.path.join(node_home, "config")

    print(f"[{moniker}] Completing setup (keys, final configs)...")
    # Directory and init are already done. No --force handling here for dir removal.

    client_config_cmds = [
        [dysond_bin, "config", "set", "client", "chain-id", chain_id_str, "--home", node_home],
        [dysond_bin, "config", "set", "client", "keyring-backend", DEFAULT_KEYRING_BACKEND, "--home", node_home]
    ]
    for cmd in client_config_cmds:
        run_command(cmd)
    print(f"[{moniker}] Configured client.")

    for acc in global_config["accounts"]:
        # Use recover with mnemonic instead of import-hex
        recover_cmd = [
            dysond_bin, "keys", "add", acc["name"],
            "--recover",
            "--keyring-backend", DEFAULT_KEYRING_BACKEND,
            "--home", node_home
        ]
        
        # Using Popen to handle interactive input for the mnemonic
        process = subprocess.Popen(
            recover_cmd,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )
        
        # Send the mnemonic to the process
        stdout, stderr = process.communicate(input=acc["mnemonic"] + "\n")
        
        if process.returncode != 0 and "already exists" not in stderr.lower():
            # If key already exists, we can ignore that error
            if "already exists" not in stderr.lower():
                raise Exception(f"[{moniker}] Failed to import key {acc['name']}: {stderr}")
        
        print(f"[{moniker}] Imported/verified key for {acc['name']}.")
            
    validator_info = full_node_config["validator"]
    val_name = validator_info["name"]
    add_val_key_cmd = [
        dysond_bin, "keys", "add", val_name,
        "--keyring-backend", DEFAULT_KEYRING_BACKEND,
        "--home", node_home
    ]
    _, stderr_val, retcode_val = run_command(add_val_key_cmd, check=False)
    if retcode_val != 0 and "already exists" not in stderr_val.lower():
        raise Exception(f"[{moniker}] Failed to add validator key {val_name}: {stderr_val}")
    print(f"[{moniker}] Added/verified validator key {val_name}.")

    show_val_addr_cmd = [dysond_bin, "keys", "show", val_name, "--address", "--keyring-backend", DEFAULT_KEYRING_BACKEND, "--home", node_home]
    validator_address, _, _ = run_command(show_val_addr_cmd)
    print(f"[{moniker}] Validator {val_name} address: {validator_address}")

    client_toml_path = os.path.join(actual_config_dir, "client.toml")
    with open(client_toml_path, 'r+') as f:
        client_toml_data = tomlkit.parse(f.read())
        for key, value in full_node_config["config"]["client"].items():
            client_toml_data[key] = value
        f.seek(0)
        f.truncate()
        f.write(tomlkit.dumps(client_toml_data))
    print(f"[{moniker}] Updated {client_toml_path}")

    def update_toml_file(file_path: str, updates: Dict[str, Any]):
        with open(file_path, 'r+') as f:
            toml_data = tomlkit.parse(f.read())
            for full_key, value in updates.items():
                parts = full_key.split('.')
                current_section: Union[tomlkit.TOMLDocument, tomlkit.items.Table] = toml_data
                for i, part in enumerate(parts[:-1]):
                    if part not in current_section:
                        current_section[part] = tomlkit.table()
                    current_section = current_section[part] # type: ignore 
                current_section[parts[-1]] = value # type: ignore
            f.seek(0)
            f.truncate()
            f.write(tomlkit.dumps(toml_data))
        print(f"[{moniker}] Updated {file_path}")

    # Apply block speed if specified in global config
    config_updates = dict(full_node_config["config"]["config"])
    if "block_speed" in global_config and global_config["block_speed"]:
        # Add consensus.timeout_commit setting
        block_speed = global_config["block_speed"]
        config_updates["consensus.timeout_commit"] = f"{block_speed}s"
        print(f"[{moniker}] Setting block time to {block_speed} seconds")

    # full_node_config now contains the "p2p.persistent_peers" under ["config"]["config"]
    update_toml_file(os.path.join(actual_config_dir, "config.toml"), config_updates)
    update_toml_file(os.path.join(actual_config_dir, "app.toml"), full_node_config["config"]["app"])
    
    print(f"[{moniker}] Node final configuration complete.")
    return {
        "node_num": full_node_config["node_num"],
        "moniker": moniker,
        "home_dir": node_home,
        "config_dir": actual_config_dir,
        "validator_name": val_name,
        "validator_address": validator_address,
        "gentx_amount": validator_info["gentx_amount"], 
        "initial_balance": validator_info["initial_balance"]
    }

def generate_gentx_for_node(node_details: Dict[str, Any], chain_id_str: str, dysond_bin: str):
    """Generates gentx for a single node. Halts on error."""
    moniker = node_details["moniker"]
    print(f"[{moniker}] Generating gentx...")
    node_home_for_gentx = node_details["home_dir"]
    gentx_cmd = [
        dysond_bin, "genesis", "gentx", node_details["validator_name"], node_details["gentx_amount"],
        "--chain-id", chain_id_str,
        "--keyring-backend", DEFAULT_KEYRING_BACKEND,
        "--home", node_home_for_gentx  # Keep --home flag
    ]
    # Pass HOME in environment, similar to chain-net.py practice
    stdout, stderr, _ = run_command(gentx_cmd, env={"HOME": node_home_for_gentx})
    
    # Verify the gentx file was created
    gentx_dir = os.path.join(node_home_for_gentx, "config", "gentx")
    gentx_files = os.listdir(gentx_dir) if os.path.exists(gentx_dir) else []
    if gentx_files:
        print(f"[{moniker}] Gentx file(s) found in {gentx_dir}: {gentx_files}")
        for gentx_file in gentx_files:
            gentx_path = os.path.join(gentx_dir, gentx_file)
            gentx_size = os.path.getsize(gentx_path)
            print(f"[{moniker}] Gentx file {gentx_file} size: {gentx_size} bytes")
    else:
        print(f"[{moniker}] WARNING: No gentx files found in {gentx_dir} after generation!")
        
    print(f"[{moniker}] Generated gentx successfully.")
    return node_details

def start_single_node(node_home_dir: str, node_moniker: str, dysond_bin: str, extra_flags: List[str]):
    """Starts a single dysond node as a background process."""
    click.echo(f"[{node_moniker}] Starting node...")
    
    # The node_home_dir from node_details is the data directory (e.g., /tmp/dysonchains/chain-a-node-1/data)
    # The dysond start command often uses --home pointing to this data directory.
    # Logs should ideally go into the parent of the data directory, e.g., /tmp/dysonchains/chain-a-node-1/node.log
    log_file_path = os.path.join(node_home_dir, f"node.log")

    start_cmd_list = [dysond_bin, "start", "--home", node_home_dir] + extra_flags
    
    click.echo(f"[{node_moniker}] Start command: {' '.join(start_cmd_list)}")
    click.echo(f"[{node_moniker}] Log file: {log_file_path}")

    try:
        with open(log_file_path, 'w') as log_file:
            process = subprocess.Popen(
                start_cmd_list,
                stdout=log_file,
                stderr=log_file,
                # cwd=node_base_dir, # Not strictly necessary if --home is used correctly
                preexec_fn=os.setsid if hasattr(os, 'setsid') else None # Detach from parent tty
            )
        running_processes.append(process)
        click.echo(f"[{node_moniker}] Started with PID {process.pid}.")
    except Exception as e:
        click.echo(f"[{node_moniker}] Failed to start node: {e}", err=True)
        # Decide if this should halt everything. For now, it just reports.
        # If one node fails to start, others might still be desired.

def get_node_rpc_url(node_home_dir: str) -> str:
    """Get the RPC URL for a node from its home directory."""
    config_dir = os.path.join(node_home_dir, "config")
    config_file = os.path.join(config_dir, "config.toml")
    
    with open(config_file, 'r') as f:
        config = tomlkit.parse(f.read())
    
    # Extract RPC address from config
    rpc_section = config.get("rpc", {})
    rpc_addr = str(rpc_section.get("laddr", "tcp://127.0.0.1:26657"))
    
    # Convert format like tcp://127.0.0.1:26657 to http://127.0.0.1:26657
    if rpc_addr.startswith("tcp://"):
        rpc_addr = "http://" + rpc_addr[6:]
    
    return rpc_addr

def get_block_height(rpc_url: str) -> Optional[int]:
    """Get the current block height from a node's RPC endpoint."""
    try:
        response = requests.get(f"{rpc_url}/status", timeout=5)
        if response.status_code == 200:
            data = response.json()
            return int(data["result"]["sync_info"]["latest_block_height"])
        else:
            print(f"Error getting block height from {rpc_url}: {response.status_code}")
            return None
    except Exception as e:
        print(f"Exception when querying {rpc_url}: {str(e)}")
        return None

def monitor_block_heights(global_config: Dict[str, Any], timeout: float, exit_event: threading.Event):
    """Monitor block heights and signal exit if no new blocks within timeout period."""
    print(f"\nStarting block height monitoring with {timeout} seconds timeout...")
    # Get all RPC URLs by iterating through global_config
    node_urls = []
    for chain_spec in global_config.get("chains", []):
        chain_id = chain_spec["chain_id"]
        for node_spec in chain_spec.get("nodes", []):
            node_home_dir = node_spec["home"]
            moniker = node_spec["moniker"]
            node_num = node_spec["node_num"] # Assuming node_num is present for unique key
            url = get_node_rpc_url(node_home_dir)
            # Using a more unique key: chain_id-moniker-node_num
            node_urls.append({"chain_id": chain_id, "node_num": node_num, "moniker": moniker, "url": url, "key": f"{chain_id}-{moniker}-{node_num}"})
    
    if not node_urls:
        print("No nodes found in global_config for block height monitoring.")
        return True # Nothing to monitor

    prev_heights = {}
    last_change_time = time.time()
    
    while not exit_event.is_set():
        time.sleep(timeout / 2)  # Check at half the timeout interval
        current_time = time.time()
        
        progress_detected = False
        all_nodes_info = []
        
        # Check block heights for all nodes
        for node_info in node_urls:
            url = node_info["url"]
            moniker = node_info["moniker"]
            node_key = node_info["key"] # Use the pre-constructed unique key
            
            height = get_block_height(url)
            all_nodes_info.append(f"{moniker} ({node_info['chain_id']}): {height if height is not None else 'Error'}")
            
            if height is not None:
                if node_key in prev_heights:
                    if height > prev_heights[node_key]:
                        progress_detected = True
                        last_change_time = current_time
                        
                prev_heights[node_key] = height
        
        # Check if timeout has been reached with no block progress
        if current_time - last_change_time > timeout:
            print(f"\nTIMEOUT REACHED: No new blocks created in {timeout} seconds!")
            print("Current block heights:")
            for info in all_nodes_info:
                print(f"  {info}")
            # Set exit event to signal main thread to exit
            exit_event.set()
            return False
        
    return True

def process_single_chain(chain_spec: Dict[str, Any], global_config: Dict[str, Any], force: bool) -> Dict[int, Dict[str, Any]]:
    """
    Process a single chain through all stages: init, peer setup, genesis, and gentx.
    Returns a map of node_num -> node_details for all fully configured nodes in this chain.
    Uses non-interruptible run_command for these core setup steps.
    """
    chain_id_str = chain_spec["chain_id"]
    nodes_in_chain_specs = chain_spec["nodes"]
    chain_node_details_map = {}
    
    click.echo(f"\nProcessing Chain '{chain_id_str}'...")

    if not nodes_in_chain_specs:
        click.echo(f"Chain '{chain_id_str}' has no nodes. Skipping.")
        return chain_node_details_map

    # Stage 1.1: Initializing nodes and gathering peer info
    click.echo(f"[{chain_id_str}] Stage 1.1: Initializing nodes and gathering peer info...")
    init_tasks = [(node_spec, chain_id_str, global_config, force) for node_spec in nodes_in_chain_specs]
    node_init_results_for_chain: List[Dict[str, Any]] = []
    
    with ProcessPoolExecutor() as executor:
        future_to_moniker = {executor.submit(init_node_and_get_peer_info, *task_args): task_args[0]["moniker"] for task_args in init_tasks}
        for future in as_completed(future_to_moniker):
            moniker = future_to_moniker[future]
            result = future.result() # Will re-raise exceptions from worker
            node_init_results_for_chain.append(result)
            click.echo(f"[{moniker}] Initial init and peer info gathered.")
    
    if len(node_init_results_for_chain) != len(nodes_in_chain_specs):
        raise Exception(f"Chain '{chain_id_str}': Not all nodes initialized successfully for peer info gathering.")

    # Stage 1.2: Prepare full node configurations with persistent peers for this chain
    click.echo(f"[{chain_id_str}] Stage 1.2: Preparing full configurations with persistent peers...")
    peer_string_map_for_chain = {info["node_num"]: info["peer_string"] for info in node_init_results_for_chain}
    tasks_for_completion_stage: List[Tuple[Dict[str,Any], str, Dict[str,Any]]] = []

    for init_result in node_init_results_for_chain:
        current_node_num = init_result["node_num"]
        # Start with the original spec from the input config file
        full_node_config_for_completion = dict(init_result["original_spec"])
        
        current_persistent_peers_list = []
        for other_node_num, peer_str in peer_string_map_for_chain.items():
            if other_node_num != current_node_num:
                current_persistent_peers_list.append(peer_str)
        
        # Convert the list of peer strings to a single comma-separated string
        persistent_peers_str = ",".join(current_persistent_peers_list)

        # Inject persistent_peers. Ensure path exists.
        if "config" not in full_node_config_for_completion: full_node_config_for_completion["config"] = {}
        if "config" not in full_node_config_for_completion["config"]: full_node_config_for_completion["config"]["config"] = {}
        full_node_config_for_completion["config"]["config"]["p2p.persistent_peers"] = persistent_peers_str
        
        tasks_for_completion_stage.append((full_node_config_for_completion, chain_id_str, global_config))
        click.echo(f"[{init_result['moniker']}] Configuration prepared with persistent_peers: '{persistent_peers_str}'")

    # Stage 1.3: Completing Node Setups (keys, final configs)
    click.echo(f"[{chain_id_str}] Stage 1.3: Completing node setups...")
    with ProcessPoolExecutor() as executor:
        future_to_moniker_complete = {executor.submit(complete_node_setup, *task_args): task_args[0]["moniker"] for task_args in tasks_for_completion_stage}
        for future in as_completed(future_to_moniker_complete):
            moniker = future_to_moniker_complete[future]
            result = future.result() # Will re-raise exceptions
            chain_node_details_map[result["node_num"]] = result
            click.echo(f"[{moniker}] Successfully completed final setup stage.")

    if len(chain_node_details_map) != len(tasks_for_completion_stage):
        raise Exception(f"[{chain_id_str}] Node final setup failed for one or more nodes after peer configuration.")

    # Stage 2: Genesis and Gentx Processing
    click.echo(f"[{chain_id_str}] Stage 2: Processing Genesis and Gentx...")
    current_chain_node_details_list: List[Dict[str, Any]] = list(chain_node_details_map.values())
    
    if not current_chain_node_details_list:
        click.echo(f"No fully configured node details found for chain '{chain_id_str}'. Skipping genesis.")
        return chain_node_details_map

    primary_node_details = current_chain_node_details_list[0]
    primary_node_config_dir = primary_node_details["config_dir"]
    primary_node_home = primary_node_details["home_dir"]
    primary_genesis_file_path = os.path.join(primary_node_config_dir, "genesis.json")
    dysond_bin = global_config["dysond_bin"]

    click.echo(f"[{chain_id_str}] Primary node for genesis: {primary_node_details['moniker']}")

    # Update genesis file with chain parameters
    with open(primary_genesis_file_path, 'r+') as f_genesis:
        genesis_data = json.load(f_genesis)
        genesis_data['chain_id'] = chain_id_str
        app_state = genesis_data['app_state']

        gov_params_config = chain_spec["genesis"]["governance_params"]
        if 'gov' not in app_state: app_state['gov'] = {}
        if 'params' not in app_state['gov']: app_state['gov']['params'] = {}
        gov_params_state = app_state['gov']['params']
        for key, value in gov_params_config.items():
            if key == "min_deposit" and isinstance(value, (int, float)):
                gov_params_state[key] = [{"denom": global_config["default_denom"], "amount": str(int(value))}]
            else:
                gov_params_state[key] = str(value) if not isinstance(value, list) else value
        click.echo(f"[{chain_id_str}] Applied governance params to genesis.")

        ns_params_config = chain_spec["genesis"]["nameservice_params"]
        if 'nameservice' not in app_state: app_state['nameservice'] = {}
        if 'params' not in app_state['nameservice']: app_state['nameservice']['params'] = {}
        ns_params_state = app_state['nameservice']['params']
        for key, value in ns_params_config.items():
                ns_params_state[key] = str(value) if not isinstance(value, list) else value
        click.echo(f"[{chain_id_str}] Applied nameservice params to genesis.")
        
        f_genesis.seek(0)
        f_genesis.truncate()
        json.dump(genesis_data, f_genesis, indent=2)

    # Add all validators to genesis accounts
    for node_d_for_acc in current_chain_node_details_list:
        add_acc_cmd = [
            dysond_bin, "genesis", "add-genesis-account", node_d_for_acc["validator_address"], node_d_for_acc["initial_balance"],
            "--home", primary_node_home
        ]
        run_command(add_acc_cmd)
        click.echo(f"[{chain_id_str}] Added validator {node_d_for_acc['validator_name']} to genesis accounts via CLI.")

    # Add all user accounts to genesis accounts
    for acc_spec in global_config["accounts"]:
        # We now have the address directly in the account spec
        user_address = acc_spec["address"]
        if user_address:
            add_user_acc_cmd = [
                dysond_bin, "genesis", "add-genesis-account", user_address, acc_spec["initial_balance"],
                "--home", primary_node_home
            ]
            run_command(add_user_acc_cmd)
            click.echo(f"[{chain_id_str}] Added user {acc_spec['name']} to genesis accounts via CLI.")
        else:
            click.echo(f"[{chain_id_str}] Warning: No address found for user {acc_spec['name']}. Skipping genesis account.", err=True)
    
    click.echo(f"[{chain_id_str}] Prepared initial genesis on {primary_node_details['moniker']}.")

    # Distribute the initial genesis file to all other nodes in this chain
    for node_d_distribute in current_chain_node_details_list:
        if node_d_distribute["home_dir"] != primary_node_home:
            dest_genesis_path = os.path.join(node_d_distribute["config_dir"], "genesis.json")
            shutil.copy(primary_genesis_file_path, dest_genesis_path)
            click.echo(f"[{chain_id_str}] Copied initial genesis to {node_d_distribute['moniker']}.")

    # Generate gentx files for all nodes in parallel
    click.echo(f"[{chain_id_str}] Starting Gentx generation for nodes in this chain...")
    gentx_tasks_args = [(node_d_gentx, chain_id_str, dysond_bin) for node_d_gentx in current_chain_node_details_list]
    
    with ProcessPoolExecutor() as gentx_executor:
        gentx_futures = {gentx_executor.submit(generate_gentx_for_node, *task_args): task_args[0]["moniker"] for task_args in gentx_tasks_args}
        for future in as_completed(gentx_futures):
            node_moniker_gentx = gentx_futures[future]
            future.result() 
            click.echo(f"[{node_moniker_gentx}] Successfully generated gentx.")

    # Verify primary node has gentx files and collect gentxs
    primary_gentx_dir = os.path.join(primary_node_home, "config", "gentx")
    primary_gentx_files = os.listdir(primary_gentx_dir) if os.path.exists(primary_gentx_dir) else []
    if not primary_gentx_files:
        click.echo(f"[{chain_id_str}] ERROR: No gentx files found in {primary_node_details['moniker']}'s gentx directory: {primary_gentx_dir}")
        click.echo(f"[{chain_id_str}] Will attempt to recreate gentx directory and regenerate gentx for primary node...")
        
        # Ensure the gentx directory exists
        os.makedirs(primary_gentx_dir, exist_ok=True)
        
        # Regenerate the primary node's gentx
        generate_gentx_for_node(primary_node_details, chain_id_str, dysond_bin)
        
        # Check again
        primary_gentx_files = os.listdir(primary_gentx_dir) if os.path.exists(primary_gentx_dir) else []
        if not primary_gentx_files:
            raise RuntimeError(f"Failed to generate gentx files for {primary_node_details['moniker']} before collect-gentxs")
    else:
        click.echo(f"[{chain_id_str}] Found {len(primary_gentx_files)} gentx file(s) in {primary_node_details['moniker']}'s gentx directory: {primary_gentx_files}")

    # Collect gentxs
    collect_cmd = [dysond_bin, "genesis", "collect-gentxs", "--home", primary_node_home]
    run_command(collect_cmd)
    click.echo(f"[{chain_id_str}] Ran collect-gentxs on {primary_node_details['moniker']}.")

    # Distribute final genesis to all nodes
    click.echo(f"[{chain_id_str}] Distributing final genesis file from {primary_node_details['moniker']}...")
    for node_d_final_dist in current_chain_node_details_list:
        if node_d_final_dist["home_dir"] != primary_node_home:
            dest_genesis_path_final = os.path.join(node_d_final_dist["config_dir"], "genesis.json")
            shutil.copy(primary_genesis_file_path, dest_genesis_path_final)
            click.echo(f"[{chain_id_str}] Copied final genesis to {node_d_final_dist['moniker']}.")
    
    click.echo(f"[{chain_id_str}] Chain setup complete.")
    return chain_node_details_map

def configure_hermes(global_config: Dict[str, Any], exit_event: threading.Event):
    """Sets up Hermes keys using the charlie mnemonic from the global configuration."""
    click.echo("\n--- Setting up Hermes keys ---")
    
    # First, find the alice account and get the mnemonic
    acc_mnemonic = None
    acc_name = "charlie"
    for acc in global_config["accounts"]:
        if acc["name"] == acc_name:
            acc_mnemonic = acc["mnemonic"]
            break
    
    if not acc_mnemonic:
        click.echo("Warning: Could not find alice account for Hermes keys setup", err=True)
        return
    
    hermes_config_path = global_config.get("hermes_config_path")
    if not hermes_config_path or not os.path.exists(hermes_config_path):
        hermes_config_path = os.path.join(global_config["base_dir"], "hermes", "config.toml")
        if not os.path.exists(hermes_config_path):
            click.echo(f"Warning: Hermes config not found at {hermes_config_path}", err=True)
            return
    
    # Create a temporary file for the alice mnemonic
    acc_mnemonic_file = os.path.join(global_config["base_dir"], "hermes", "acc_mnemonic.txt")
    os.makedirs(os.path.dirname(acc_mnemonic_file), exist_ok=True)
    
    try:
        with open(acc_mnemonic_file, 'w') as f:
            f.write(acc_mnemonic)
        
        # For each chain defined in global_config, add the key
        for chain_spec in global_config.get("chains", []):
            chain_id = chain_spec.get("chain_id")
            if not chain_id:
                click.echo(f"Warning: Skipping Hermes key setup for a chain missing 'chain_id' in global_config: {chain_spec}", err=True)
                continue

            click.echo(f"Adding {acc_name} key to Hermes for chain {chain_id}...")
            
            hermes_add_key_cmd = [
                "hermes", 
                "--config", hermes_config_path,
                "keys", "add",
                "--chain", chain_id,
                "--mnemonic-file", acc_mnemonic_file,
                "--key-name", acc_name,
                "--overwrite"
            ]
            
            try:
                stdout, stderr, retcode = run_command(hermes_add_key_cmd, check=False, exit_event=exit_event)
                if retcode == 0:
                    click.echo(f"Successfully added alice key to Hermes for chain {chain_id}")
                else:
                    click.echo(f"Warning: Failed to add alice key to Hermes for chain {chain_id}: {stderr}", err=True)
            except Exception as e:
                click.echo(f"Error adding alice key to Hermes for chain {chain_id}: {str(e)}", err=True)
    finally:
        # Clean up - remove the mnemonic file
        if os.path.exists(acc_mnemonic_file):
            os.remove(acc_mnemonic_file)
            click.echo("Cleaned up temporary mnemonic file")
    
    return hermes_config_path

def _extract_hermes_result_json(output_text: str) -> Dict[str, Any]:
    """Parse output containing multiple JSON objects and extract the result JSON."""
    json_objects = []
    for line in output_text.strip().split('\n'):
        try:
            json_obj = json.loads(line)
            json_objects.append(json_obj)
        except json.JSONDecodeError:
            continue
    for json_obj in reversed(json_objects):
        if isinstance(json_obj, dict) and "result" in json_obj:
            return json_obj
    raise Exception(f"No JSON object with 'result' field found in output: {output_text}")

def verify_hermes_channels(chains_list: List[str], hermes_config_path: str, exit_event: threading.Event):
    """Queries and logs existing IBC channels for the given chains using Hermes."""
    if not chains_list:
        click.echo("No chains provided for channel verification.")
        return

    click.echo("\n--- Verifying/Listing IBC channels ---")
    for chain in chains_list:
        try:
            query_cmd = [
                "hermes",
                "--config", hermes_config_path,
                "--json",
                "query", "channels",
                "--show-counterparty",
                "--chain", chain
            ]
            stdout, stderr, _ = run_command(query_cmd, exit_event=exit_event)
            output_text = stdout if stdout else stderr
            click.echo(f"\nChannels for {chain}:")
            
            try:
                result_json = _extract_hermes_result_json(output_text)
                
                if "result" in result_json and result_json["result"]: # Check if result is not empty
                    for channel in result_json["result"]:
                        click.echo(f"  {channel}")
                else:
                    click.echo(f"  No channels found or unexpected format for {chain}.")
            except Exception as e:
                click.echo(f"  Failed to parse channel query output for {chain}: {str(e)}")
                click.echo(f"  Raw output for {chain}: {output_text}") # Log raw output for debugging
        except CommandInterruptedError:
            click.echo(f"Channel verification for {chain} interrupted.", err=True)
            raise # Re-raise to stop further processing if interrupted
        except subprocess.CalledProcessError as e:
            click.echo(f"Error querying channels for {chain} (command failed): {str(e)}", err=True)
            if e.stderr: click.echo(f"STDERR: {e.stderr.strip()}", err=True)
        except Exception as e:
            click.echo(f"Unexpected error querying channels for {chain}: {str(e)}", err=True)

def create_hermes_connections(chains_list: List[str], hermes_config_path: str, exit_event: threading.Event):
    """Creates IBC connections between chains in a line topology (chain-a -> chain-b -> chain-c ...)
    using 'hermes create channel --new-client-connection'.
    """
    if len(chains_list) < 2:
        click.echo("Need at least 2 chains to create connections. Skipping IBC connection creation.")
        return

    click.echo("\n--- Creating IBC connections in a line topology (using --new-client-connection) ---")
    chains_list.sort() # Ensure consistent ordering for line topology
    
    for i in range(len(chains_list) - 1):
        chain_a = chains_list[i]
        chain_b = chains_list[i + 1]
        
        click.echo(f"\nChecking for existing channel between {chain_a} and {chain_b} on transfer ports...")

        channel_exists = False
        try:
            query_channels_cmd = [
                "hermes",
                "--config", hermes_config_path,
                "--json",
                "query", "channels",
                "--show-counterparty",
                "--chain", chain_a
            ]
            # Run with check=False as no channels found is not an error for this specific check.
            stdout_query, stderr_query, retcode_query = run_command(query_channels_cmd, exit_event=exit_event, check=False)
            output_text_query = stdout_query if stdout_query else stderr_query

            if retcode_query == 0 and output_text_query: # Command succeeded and there's output
                try:
                    result_json_query = _extract_hermes_result_json(output_text_query)
                    if "result" in result_json_query and result_json_query["result"]:
                        for channel in result_json_query["result"]:
                            counterparty = channel.get("counterparty", {})
                            # Check if this channel connects chain_a's 'transfer' port to chain_b's 'transfer' port
                            if (channel.get("port_id") == "transfer" and
                                counterparty.get("chain_id") == chain_b and
                                counterparty.get("port_id") == "transfer"):
                                click.echo(f"  Found existing channel: {channel.get('channel_id')} on {chain_a} (port {channel.get('port_id')}) connected to {chain_b} (port {counterparty.get('port_id')})")
                                channel_exists = True
                                break 
                    elif "result" in result_json_query and not result_json_query["result"]:
                        click.echo(f"  No channels found listed for {chain_a}.")
                    else: # No "result" field or unexpected structure
                         click.echo(f"  Channel query for {chain_a} returned unexpected JSON structure. Raw: {output_text_query[:200]}...")


                except Exception as e:
                    click.echo(f"  Could not parse channel query output for {chain_a}: {str(e)}. Proceeding with creation attempt.", err=True)
                    # click.echo(f"  Raw output for {chain_a} during parsing error: {output_text_query[:200]}...", err=True)
            elif retcode_query != 0 :
                 click.echo(f"  Channel query command for {chain_a} failed with code {retcode_query}. Stdout: {stdout_query[:100]}, Stderr: {stderr_query[:100]}. Proceeding with creation attempt.", err=True)
            else: # retcode_query == 0 but no output_text_query
                click.echo(f"  No channel information output by Hermes for {chain_a}. Proceeding with creation attempt.")

        except CommandInterruptedError:
            click.echo(f"Channel query for {chain_a} interrupted.", err=True)
            raise # Re-raise to stop further processing if interrupted
        except subprocess.CalledProcessError as e: # Should be caught by check=False, but as a safeguard
            click.echo(f"Error during channel query for {chain_a} (command failed unexpectedly): {str(e)}. Proceeding with creation attempt.", err=True)
            if e.stderr: click.echo(f"STDERR: {e.stderr.strip()}", err=True)
        except Exception as e: # Catch any other unexpected error during query phase
            click.echo(f"Unexpected error querying channels for {chain_a}: {str(e)}. Proceeding with creation attempt.", err=True)

        if channel_exists:
            click.echo(f"Relevant channel already exists between {chain_a} and {chain_b}. Skipping creation.")
            continue # Move to the next pair of chains
        
        click.echo(f"No existing relevant channel found. Attempting to create channel with new client & connection: {chain_a} <-> {chain_b}")
        
        create_channel_full_cmd = [
            "hermes", 
            "--config", hermes_config_path,
            "--json",
            "create", "channel", 
            "--a-chain", chain_a, 
            "--b-chain", chain_b,
            "--a-port", "transfer", # Assuming standard transfer port
            "--b-port", "transfer", # Assuming standard transfer port
            "--new-client-connection",
            "--yes" # Skip confirmation
        ]
        
        try:
            stdout, stderr, _ = run_command(create_channel_full_cmd, exit_event=exit_event)
            output_text = stdout if stdout else stderr
            
            result_json = _extract_hermes_result_json(output_text)
            
            channel_id_a_side = "N/A"
            if "result" in result_json and isinstance(result_json["result"], dict) and "a_side" in result_json["result"]:
                a_side_details = result_json["result"]["a_side"]
                if isinstance(a_side_details, dict) and "channel_id" in a_side_details:
                    channel_id_a_side = a_side_details["channel_id"]
            
            click.echo(f"Successfully initiated channel creation between {chain_a} and {chain_b}. Channel ID (a_side): {channel_id_a_side}")

        except CommandInterruptedError:
            click.echo(f"Channel creation between {chain_a} and {chain_b} interrupted.", err=True)
            raise # Re-raise
        except subprocess.CalledProcessError as e:
            click.echo(f"Error creating channel (command failed) with new client/connection between {chain_a} and {chain_b}: {str(e)}", err=True)
            if e.stderr: click.echo(f"STDERR: {e.stderr.strip()}", err=True)
            raise # Halt if one connection fails.
        except Exception as e: # Catch other errors like JSON parsing
            click.echo(f"Unexpected error creating channel with new client/connection between {chain_a} and {chain_b}: {str(e)}", err=True)
            click.echo(f"Raw output: {output_text if 'output_text' in locals() else 'N/A'}", err=True)
            raise
        
        click.echo(f"Connection setup initiated for {chain_a} -> {chain_b}")
    
    click.echo("\nFinished attempting to create IBC connections in line topology.")

    # Moved channel verification to after all Hermes setup attempts (create connections, start relayer)

def start_hermes(hermes_config_path: str):
    """Starts the Hermes relayer as a background process using the provided config path."""
    click.echo("\n--- Starting Hermes Relayer ---")
    
    if not os.path.exists(hermes_config_path):
        click.echo(f"Error: Hermes config not found at {hermes_config_path}", err=True)
        return None
    
    # Create a log file for Hermes
    hermes_dir = os.path.dirname(hermes_config_path)
    log_file_path = os.path.join(hermes_dir, "hermes.log")
    
    click.echo(f"Starting Hermes with config: {hermes_config_path}")
    click.echo(f"Hermes log file: {log_file_path}")
    
    start_cmd = ["hermes", "--config", hermes_config_path, "start"]
    print(shlex.join(start_cmd))
    try:
        with open(log_file_path, 'w') as log_file:
            process = subprocess.Popen(
                start_cmd,
                stdout=log_file,
                stderr=log_file,
                preexec_fn=os.setsid if hasattr(os, 'setsid') else None  # Detach from parent tty
            )
        running_processes.append(process)
        click.echo(f"Hermes started with PID {process.pid}")
        return process
    except Exception as e:
        click.echo(f"Failed to start Hermes: {e}", err=True)
        return None

@click.command(context_settings=dict(ignore_unknown_options=True, allow_extra_args=True))
@click.option('--config-file', type=click.File('r'), required=True, help='Path to the JSON network configuration file, or - for stdin.')
@click.option('--force', is_flag=True, help='Force re-initialization if node directories exist.')
@click.option('--setup', is_flag=True, help='Initialize and configure nodes (genesis, gentx, etc.).')
@click.option('--start-nodes', is_flag=True, help='Start all nodes after setup. Extra args are passed to dysond start.')
@click.option('--no-blocks-timeout', type=float, help='Timeout in seconds to stop all nodes if no new blocks are created.')
@click.option('--block-speed', type=float, help='Set the block production speed in seconds (timeout_commit value).')
@click.option('--hermes', is_flag=True, help='Set up Hermes keys and start the Hermes relayer')
@click.option('--create-ibc-connections', is_flag=True, help='Create IBC connections between chains using Hermes')
@click.pass_context
def main(ctx: click.Context, config_file: TextIO, force: bool, setup: bool, start_nodes: bool, no_blocks_timeout: Optional[float], block_speed: Optional[float], hermes: bool, create_ibc_connections: bool):
    """
    Sets up a Dyson Protocol network based on a JSON configuration file.
    Handles node initialization, persistent peer configuration, key setup,
    and genesis/gentx processes. Halts on any error.
    
    If --no-blocks-timeout is set, all nodes will be stopped if no new blocks
    are produced within the specified timeout period (in seconds).
    
    If --block-speed is set, blocks will be produced at the specified interval in seconds.
    
    If --hermes is set, the Hermes relayer will be set up and started after the nodes.
    
    If --create-ibc-connections is set, IBC connections will be created between chains using Hermes.
    """
    # When type=click.File('r'), config_file will be an open file handle
    # So we read from it directly.
    global_config = json.load(config_file)

    click.echo(f"Loaded configuration.") # No longer printing filename directly as it could be stdin
    dysond_bin = global_config["dysond_bin"]
    click.echo(f"Dyson binary: {dysond_bin}")

    if setup:
        click.echo(f"Number of chains to set up: {len(global_config['chains'])}")

        # Add block_speed to global_config if provided
        if block_speed is not None:
            global_config["block_speed"] = block_speed
            click.echo(f"Setting block production speed to {block_speed} seconds")

        if not global_config['chains']:
            click.echo("No chains defined in configuration. Exiting.")
            return

        # Process all chains in parallel using ProcessPoolExecutor
        all_chains_node_details_map: Dict[int, Dict[str, Any]] = {}
        chain_processing_tasks = [(chain_spec, global_config, force) for chain_spec in global_config["chains"]]
        
        click.echo("\n--- Processing all chains in parallel ---")
        try:
            with ProcessPoolExecutor() as executor:
                future_to_chain = {executor.submit(process_single_chain, *task_args): task_args[0]["chain_id"] for task_args in chain_processing_tasks}
                for future in as_completed(future_to_chain):
                    chain_id = future_to_chain[future]
                    chain_node_details = future.result()  # Will re-raise exceptions
                    all_chains_node_details_map.update(chain_node_details)
                    click.echo(f"Chain '{chain_id}' processing complete.")
        except Exception as e:
            click.echo(f"FATAL ERROR during chain processing: {e}", err=True)
            raise

        click.echo("\nAll chains processed. Network setup script finished.")

    hermes_process = None
    
    if start_nodes:
        click.echo("\n--- Stage 3: Starting Nodes ---")

        extra_start_flags = ctx.args
        if extra_start_flags:
            click.echo(f"Passing extra flags to 'dysond start': {' '.join(extra_start_flags)}")
        for chain in global_config["chains"]:
            for node in chain["nodes"]:
                start_single_node(
                    node_home_dir=node["home"],
                    node_moniker=node["moniker"],
                    dysond_bin=dysond_bin,
                    extra_flags=extra_start_flags
                )
        
        if running_processes:
            click.echo("\nAll specified nodes have been launched.")
            
            # Display a grid of chain, node, and port information
            click.echo("\nNetwork Overview:")
            click.echo("=" * 120)
            header = f"{'Chain ID':<12} {'Node':<12} {'RPC':<8} {'gRPC':<8} {'API':<8}  {'Node Info URL':<60}"
            click.echo(header)
            click.echo("-" * 120)
            
            # Find address for node_info_url
            acc_address_for_url = "charlie" # Default account to use for constructing URL
            for acc in global_config["accounts"]:
                if acc["name"] == acc_address_for_url:
                    acc_address_for_url = acc.get("address", "")
                    break
            
            for chain_spec in global_config["chains"]:
                chain_id = chain_spec["chain_id"]
                for node_spec in chain_spec["nodes"]:
                    moniker = node_spec["moniker"]
                    ports = node_spec.get("ports", {"rpc": "n/a", "grpc": "n/a", "api": "n/a"}) # Use .get for safety
                    
                    # Create the node info URL
                    api_port = ports.get("api", "n/a")
                    if api_port != "n/a" and acc_address_for_url:
                        node_info_url = f"{acc_address_for_url}@localhost:{api_port}/cosmos/base/tendermint/v1beta1/node_info"
                    else:
                        node_info_url = "n/a"
                    
                    # Format the row
                    row = f"{chain_id:<12} {moniker:<12} {ports.get('rpc', 'n/a'):<8} {ports.get('grpc', 'n/a'):<8} {api_port:<8}  {node_info_url:<60}"
                    click.echo(row)

            exit_event = threading.Event()

            time.sleep(1) # Wait for nodes to start up

            # Start block height monitoring if timeout is specified
            if no_blocks_timeout:
                monitor_thread = threading.Thread(
                    target=monitor_block_heights,
                    args=(global_config, no_blocks_timeout, exit_event)
                )
                monitor_thread.daemon = True
                monitor_thread.start()
                click.echo(f"Block height monitoring active: will stop all nodes if no blocks for {no_blocks_timeout} seconds.")

            hermes_config_path = None
            try:
                # Start Hermes if requested and nodes are running
                if hermes or create_ibc_connections and not exit_event.is_set():
                    hermes_config_path = configure_hermes(global_config, exit_event)
                    if not hermes_config_path:
                        click.echo("Hermes config not found, cannot start Hermes relayer.", err=True)
                        exit_event.set()

                if hermes and hermes_config_path and not exit_event.is_set():
                    # Start the Hermes relayer
                    click.echo("\nStarting Hermes relayer...")
                    hermes_process = start_hermes(hermes_config_path)
                    
                # Create IBC connections if requested
                if create_ibc_connections and hermes_config_path and not exit_event.is_set():
                    chains_for_hermes_connections = [chain_spec["chain_id"] for chain_spec in global_config.get("chains", []) if "chain_id" in chain_spec]
                    if chains_for_hermes_connections:
                        create_hermes_connections(chains_for_hermes_connections, hermes_config_path, exit_event)
                    else:
                        click.echo("No chains found in global_config to create IBC connections for.", err=True)
                
                # Moved channel verification to after all Hermes setup attempts (create connections, start relayer)

            except CommandInterruptedError as e:
                click.echo(f"Hermes operation interrupted: {e}", err=True)
            except subprocess.CalledProcessError as e:
                click.echo(f"Error during Hermes setup/connection: {e}", err=True)
                if e.stderr: click.echo(f"STDERR: {e.stderr.strip()}", err=True)
                exit_event.set()
            except Exception as e:
                click.echo(f"Unexpected error during Hermes operations: {e}", err=True)
                exit_event.set()
            
            # Always try to verify/log channels if Hermes was configured, regardless of connection creation or relayer start success/failure, but before main loop
            if hermes_config_path and not exit_event.is_set():
                chains_for_verification = [chain_spec["chain_id"] for chain_spec in global_config.get("chains", []) if "chain_id" in chain_spec]
                if chains_for_verification:
                    verify_hermes_channels(chains_for_verification, hermes_config_path, exit_event)
                else:
                    click.echo("No chains found in global_config for Hermes channel verification during final check.", err=True)
       
            click.echo("=" * 120)
            
            click.echo("Press Ctrl+C to attempt graceful shutdown of started nodes.")
            
            try:
                while True:                    
                    # Check if exit event was set by the monitoring thread or Hermes errors
                    if exit_event.is_set():
                        click.echo("Exit event triggered - stopping all nodes...")
                        break
                    
                    # Check if any process is still alive. If all are dead, can exit.
                    any_alive = any(proc.poll() is None for proc in running_processes)
                    if not any_alive and running_processes:
                        click.echo("All started nodes seem to have terminated.")
                        break
                    
                    time.sleep(1) # Keep main thread alive
            except KeyboardInterrupt:
                # signal_handler should have been called. Re-call just in case or rely on it.
                click.echo("Keyboard interrupt received in main loop after starting nodes.")
            finally:
                # Ensure cleanup is attempted if loop breaks for other reasons or on normal exit after ^C
                stop_all_started_nodes()
                if exit_event.is_set():
                    click.echo("Script terminated due to exit event (e.g., timeout or error).")
                    sys.exit(1)
                else:
                    click.echo("Script finished.")
        else:
            click.echo("No nodes were started (or failed to start).")

if __name__ == '__main__':
    main() 
