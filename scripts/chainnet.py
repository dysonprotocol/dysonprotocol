#!/usr/bin/env python3
"""
Unified CLI for Cosmos SDK chains and IBC with Hermes.

Commands:
  - generate: write network config JSON (and optional Hermes TOML)
  - setup: init nodes, add accounts, configure peers, gentx, collect-gentxs, distribute genesis
  - start: launch dysond nodes and Hermes relayer
  - ibc: create IBC connections between chains

Examples:
  # Generate network config with 2 chains, 1 node each, and Hermes TOML
  ./scripts/chainnet.py generate --chains 2 --nodes 1 --hermes-config

  # Setup the network (init, gentx, genesis) from default config
  ./scripts/chainnet.py setup --config-file /tmp/dysonchains/chains.json --force

  # Start nodes with extra flags (e.g., pruning, timeout)
  ./scripts/chainnet.py start --config-file /tmp/dysonchains/chains.json -- --pruning everything

  # Create IBC channels between chains
  ./scripts/chainnet.py ibc --config-file /tmp/dysonchains/chains.json
"""
import json
import os
import signal
import shutil
import subprocess
import sys
import atexit
from pathlib import Path
import time
from typing import cast

import click
import requests
import tomlkit


# --- Defaults & Constants ---
DEFAULT_DENOM = "dys"
DEFAULT_BASE_DIR = Path("/tmp/dysonchains")
DEFAULT_CONFIG_PATH = DEFAULT_BASE_DIR / "chains.json"
DEFAULT_GENTX_AMOUNT = f"1000000{DEFAULT_DENOM}"
DEFAULT_INITIAL_BALANCE = f"10000000000{DEFAULT_DENOM}"
USER_KEYS = {
    "alice": {"address": "dys1tvhkv3gqr90jpycaky02xa5ukhaxllu38wawhz", "mnemonic": "public feature teach face federal matrix throw legend bridge brass diary beach typical doll evoke weapon among crane regret trust enact swarm brother outside"},
    "bob":   {"address": "dys1fhhxp9xveswc4yhxekr32eqe80rkwpurya0jh0", "mnemonic": "aerobic creek copper rice disagree become brass elegant century elegant apology position infant saddle metal brain gain loud alpha add boy balance truth cherry"},
    "charlie": {"address": "dys1cvqzw2968lq5wzldcglds02gnxg3d49f523ksf", "mnemonic": "blind people aim sheriff awkward once wish above agree journey unknown uncover swap damage bamboo volume clay error weekend fiber acquire diamond vintage lake"}
}
MAX_CHAINNET_OFFSET = 10
MAX_NODES_PER_CHAIN = 10

# --- Helpers for generation ---
def generate_ports(port_offset: int, chainnet_offset: int) -> dict:
    """Generate port mappings for a node with given offsets.
    
    Args:
        port_offset: Offset for this node within its chain (0-based)
        chainnet_offset: Offset for this chainnet instance (0-based)
        
    Returns:
        Dict mapping service names to port numbers
    """

    assert port_offset < MAX_NODES_PER_CHAIN, f"port_offset must be less than {MAX_NODES_PER_CHAIN}, do you really need more than {MAX_NODES_PER_CHAIN} nodes per chain?"
    assert chainnet_offset < MAX_CHAINNET_OFFSET, f"chainnet_offset must be less than {MAX_CHAINNET_OFFSET}, do you really need more than {MAX_CHAINNET_OFFSET} chains?"

    base_ports = {
        "p2p": 26656,
        "rpc": 26657, 
        "abci": 26658,
        "grpc": 9090,
        "api": 1317,
        "telemetry": 8001,
        "pprof": 6060
    }
    
    return {
        service: base + (port_offset * 100) + (chainnet_offset * 1000)
        for service, base in base_ports.items()
    }


def generate_chain_structure(chainnet_offset: int, idx: int, num_chains: int, nodes_per_chain: int, base_dir: Path) -> dict:
    chain_id = f"chain-{chr(ord('a')+idx)}"
    genesis = {
        "governance_params": {
            "voting_period": "1s",
            "expedited_voting_period": "500ms",
            "expedited_threshold": "0.667",
            "min_deposit": 1,
            "quorum": "0.01",
            "threshold": "0.1"
        },
        "nameservice_params": {
            "reject_bid_valuation_fee_percent": "0.03",
            "bid_timeout": "2s"
        }
    }
    nodes = []
    for j in range(nodes_per_chain):
        nid = idx * nodes_per_chain + j + 1
        moniker = f"node-{nid}"
        home = base_dir / f"{chain_id}-{moniker}"
        ports = generate_ports(port_offset=nid, chainnet_offset=chainnet_offset)
        nodes.append({
            "moniker": moniker,
            "home": str(home),
            "ports": ports,
            "validator": {"gentx_amount": DEFAULT_GENTX_AMOUNT, "initial_balance": DEFAULT_INITIAL_BALANCE}
        })
    return {"chain_id": chain_id, "genesis": genesis, "nodes": nodes}


def generate_accounts() -> list:
    return [{"name": n, "address": d["address"], "mnemonic": d["mnemonic"], "initial_balance": DEFAULT_INITIAL_BALANCE} for n, d in USER_KEYS.items()]


def generate_hermes_config(chains: list, base_dir: Path, denom: str) -> str:
    """Generate a full Hermes relayer TOML config with chain entries as [[chains]] array of tables."""
    return f"""
[global]
log_level = 'info'

[mode]

[mode.clients]
enabled = true
refresh = true
misbehaviour = true

[mode.connections]
enabled = true

[mode.channels]
enabled = true

[mode.packets]
enabled = true
clear_interval = 1
clear_on_start = true
tx_confirmation = true

[telemetry]
enabled = true
host = '127.0.0.1'
port = 3001

{"".join(f'''
[[chains]]
id = "{chain["chain_id"]}"
type = "CosmosSdk"
rpc_addr = "http://localhost:{chain["nodes"][0]["ports"]["rpc"]}"
grpc_addr = "http://localhost:{chain["nodes"][0]["ports"]["grpc"]}"
event_source = {{ mode = "pull", interval = '100ms' }}
rpc_timeout = "15s"
trusted_node = true
account_prefix = "dys"
key_name = "charlie"
store_prefix = "ibc"
gas_price = {{ price = 0.001, denom = "{denom}" }}
gas_multiplier = 1.2
default_gas = 1000000
max_gas = 10000000
max_msg_num = 30
max_tx_size = 2097152
clock_drift = "5s"
max_block_time = "500ms"
trusting_period = "14days"
trust_threshold = {{ numerator = "2", denominator = "3" }}

[chains.packet_filter]
policy = "allow"
list = [
  ["*", "*"],
]
''' for chain in chains)}

"""

# --- CLI ---
@click.group()
def chainnet():
    """Manage chain generation, setup, start, and IBC."""
    pass

@chainnet.command()
@click.option('--base-dir', default=DEFAULT_BASE_DIR, type=click.Path(), show_default=True)
@click.option('--chains', 'num_chains', default=2, type=int, show_default=True)
@click.option('--chainnet-offset', default=0, type=int, show_default=True)
@click.option('--nodes', 'nodes_per_chain', default=1, type=int, show_default=True)
@click.option('--denom', default=DEFAULT_DENOM, show_default=True)
@click.option('--dysond-bin', default='dysond')
@click.option('--hermes-config', is_flag=True)
@click.option('--output', type=click.Path(), help='JSON output path')
def generate(base_dir, num_chains, chainnet_offset, nodes_per_chain, denom, dysond_bin, hermes_config, output):
    """Generate and persist network config JSON (and optional Hermes TOML)."""
    base = Path(base_dir); base.mkdir(parents=True, exist_ok=True)
    cfg = {"dysond_bin": dysond_bin, "default_denom": denom, "base_dir": str(base), "chains": [], "accounts": generate_accounts()}
    for i in range(num_chains):
        cfg["chains"].append(generate_chain_structure(chainnet_offset=chainnet_offset, idx=i, num_chains=num_chains, nodes_per_chain=nodes_per_chain, base_dir=base))
    path = Path(output) if output else DEFAULT_CONFIG_PATH
    path.write_text(json.dumps(cfg, indent=2))
    # print contents of path
    click.echo(f"Wrote config JSON to {path}")
    if hermes_config:
        hermes_dir = base / 'hermes'; hermes_dir.mkdir(exist_ok=True)
        toml = generate_hermes_config(cfg['chains'], base, denom)
        (hermes_dir / 'config.toml').write_text(toml)
        click.echo(f"Wrote Hermes TOML to {str(hermes_dir / 'config.toml')}")

@chainnet.command()
@click.option('--config-file', default=DEFAULT_CONFIG_PATH, type=click.Path(exists=True))
@click.option('--force', is_flag=True)
def setup(config_file, force):
    """Initialize nodes, keys, genesis, gentx, collect-gentxs, and distribute genesis.json"""
    cfg = json.loads(Path(config_file).read_text())
    bin_path = cfg['dysond_bin']
    denom = cfg.get('default_denom', DEFAULT_DENOM)

    for chain_idx, chain in enumerate(cfg['chains']):
        cid = chain['chain_id']
        click.echo(f"Setting up {cid}")
        peer_map = {}

        # Step 1: Initialize nodes, configure ports/peers, add keys, get validator addresses,
        # and prepare individual genesis files with initial balances and params.
        for node_idx, node_config_data in enumerate(chain['nodes']):
            home = Path(node_config_data['home'])
            moniker = node_config_data['moniker']
            node_ports = node_config_data['ports']

            if home.exists():
                if force: shutil.rmtree(home)
                else: raise RuntimeError(f"{home} exists; use --force")
            home.mkdir(parents=True)

            # Init node
            init_proc = subprocess.run([
                bin_path, 'init', moniker, '--chain-id', cid, '--default-denom', denom,
                '-o', '--home', str(home)
            ], stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, check=True)
            init_output = init_proc.stdout.strip() or init_proc.stderr.strip()
            node_info = json.loads(init_output)
            peer_map[moniker] = f"{node_info['node_id']}@127.0.0.1:{node_ports['p2p']}"
            node_config_data['node_id'] = node_info['node_id'] # Store node_id for later use if needed

            # Configure config.toml for this node
            cfg_dir = home / 'config'
            toml_conf_path = cfg_dir / 'config.toml'
            toml_conf = tomlkit.parse(toml_conf_path.read_text())
            toml_conf['proxy_app'] = f"tcp://127.0.0.1:{node_ports['abci']}"
            toml_conf['pprof_laddr'] = f"localhost:{node_ports['pprof']}"
            cast(dict, toml_conf.setdefault('rpc', tomlkit.table()))['laddr'] = f"tcp://127.0.0.1:{node_ports['rpc']}"
            p2p_table = cast(dict, toml_conf.setdefault('p2p', tomlkit.table()))
            # Persistent peers will be set later after all nodes in this chain are processed
            p2p_table['laddr'] = f"tcp://0.0.0.0:{node_ports['p2p']}"
            p2p_table['allow_duplicate_ip'] = True
            inst_table = cast(dict, toml_conf.setdefault('instrumentation', tomlkit.table()))
            inst_table['prometheus_listen_addr'] = f":{node_ports['telemetry']}"
            inst_table['prometheus'] = True
            toml_conf_path.write_text(tomlkit.dumps(toml_conf))

            # Configure client.toml for this node
            client_toml_path = cfg_dir / 'client.toml'
            client_toml_obj = tomlkit.parse(client_toml_path.read_text())
            client_toml_obj['chain-id'] = cid
            client_toml_obj['keyring-backend'] = 'test'
            client_toml_obj['node'] = f"tcp://localhost:{node_ports['rpc']}"
            client_toml_path.write_text(tomlkit.dumps(client_toml_obj))

            # Configure app.toml for this node
            app_toml_path = cfg_dir / 'app.toml'
            if app_toml_path.exists():
                app_toml = tomlkit.parse(app_toml_path.read_text())
                cast(dict, app_toml.setdefault('api', tomlkit.table()))['address'] = f"tcp://localhost:{node_ports['api']}"
                cast(dict, app_toml.setdefault('api', tomlkit.table()))['enable'] = True
                cast(dict, app_toml.setdefault('grpc', tomlkit.table()))['address'] = f"localhost:{node_ports['grpc']}"
                cast(dict, app_toml.setdefault('grpc', tomlkit.table()))['enable'] = True
                cast(dict, app_toml.setdefault('grpc-web', tomlkit.table()))['enable'] = True
                app_toml_path.write_text(tomlkit.dumps(app_toml))
            else:
                click.echo(f"Warning: app.toml not found at {app_toml_path}, skipping its port configuration.")

            # Add user keys to this node's keyring
            for acc in cfg['accounts']:
                subprocess.run([
                    bin_path, 'keys', 'add', acc['name'], '--recover', '--keyring-backend', 'test', '--home', str(home)
                ], input=acc['mnemonic'] + "\n", text=True, check=True, capture_output=True)
            
            # Add validator key for this node to its own keyring and get address
            subprocess.run([
                bin_path, 'keys', 'add', "validator", '--keyring-backend', 'test', '--home', str(home)
            ], check=True, capture_output=True, text=True)
            show_addr_proc = subprocess.run([
                bin_path, 'keys', 'show', "validator", '-a', '--keyring-backend', 'test', '--home', str(home)
            ], capture_output=True, text=True, check=True)
            validator_address = show_addr_proc.stdout.strip()
            node_config_data['validator_address'] = validator_address

            # Modify this node's genesis.json
            current_genesis_path = cfg_dir / 'genesis.json'
            gdata = json.loads(current_genesis_path.read_text())
            app_state = gdata.setdefault('app_state', {})
            # Gov params
            gov_params = chain['genesis']['governance_params']
            app_state_gov = app_state.setdefault('gov', {})
            app_state_gov_params = app_state_gov.setdefault('params', {})
            for key, value in gov_params.items():
                if key == 'min_deposit' and isinstance(value, (int, float)):
                    app_state_gov_params[key] = [{'denom': denom, 'amount': str(int(value))}]
                else:
                    app_state_gov_params[key] = str(value)
            # Nameservice params
            ns_params = chain['genesis']['nameservice_params']
            app_state_ns = app_state.setdefault('nameservice', {})
            app_state_ns_params = app_state_ns.setdefault('params', {})
            for key, value in ns_params.items():
                app_state_ns_params[key] = str(value)
            current_genesis_path.write_text(json.dumps(gdata, indent=2))

            # Add user accounts to this node's genesis
            for acc in cfg['accounts']:
                subprocess.run([
                    bin_path, 'genesis', 'add-genesis-account', acc['address'], acc['initial_balance'], '--home', str(home)
                ], check=True, capture_output=True, text=True)
            # Add this validator's initial_balance to its own genesis
            subprocess.run([
                bin_path, 'genesis', 'add-genesis-account', validator_address, 
                node_config_data['validator']['initial_balance'], '--home', str(home)
            ], check=True, capture_output=True, text=True)

        # Now that all nodes in this chain are initialized, set persistent_peers for each
        for node_config_data_for_peers in chain['nodes']:
            home = Path(node_config_data_for_peers['home'])
            moniker = node_config_data_for_peers['moniker']
            peers_for_this_node = [p_str for m, p_str in peer_map.items() if m != moniker]
            
            toml_conf_path = home / 'config' / 'config.toml'
            toml_conf = tomlkit.parse(toml_conf_path.read_text())
            cast(dict, toml_conf.setdefault('p2p', tomlkit.table()))['persistent_peers'] = ",".join(peers_for_this_node)
            toml_conf_path.write_text(tomlkit.dumps(toml_conf))

        # Step 2: Generate Gentx for each node using its own prepared genesis
        for node_config_data in chain['nodes']:
            subprocess.run([
                bin_path, 'genesis', 'gentx', "validator", 
                node_config_data['validator']['gentx_amount'], 
                '--chain-id', cid, '--home', str(node_config_data['home'])
            ], check=True)

        # Step 3: Collect Gentxs into the primary node's genesis and distribute
        primary_node_home_str = str(Path(chain['nodes'][0]['home']))
        subprocess.run([
            bin_path, 'genesis', 'collect-gentxs', '--home', primary_node_home_str
        ], check=True, capture_output=True, text=True)

        final_genesis_content = (Path(primary_node_home_str) / 'config' / 'genesis.json').read_text()
        for i in range(1, len(chain['nodes'])): # Distribute to other nodes
            other_node_home = Path(chain['nodes'][i]['home'])
            (other_node_home / 'config' / 'genesis.json').write_text(final_genesis_content)
            
    click.echo("Setup complete")

@chainnet.command()
@click.option('--config-file', default=DEFAULT_CONFIG_PATH, type=click.Path(exists=True))
@click.option('--block-speed', default=None, help='Override block production speed (timeout_commit) for all nodes, e.g. "500ms" or "1s".')
@click.option('--no-blocks-timeout', default=None, type=float, help='Timeout in seconds to stop if no new blocks are produced by any node.')
@click.argument('extra_args', nargs=-1)
def start(config_file, block_speed, extra_args, no_blocks_timeout):
    """Start all dysond nodes and Hermes relayer."""
    import threading, time, requests
    cfg = json.loads(Path(config_file).read_text())
    bin_path = cfg['dysond_bin']
    procs = []
    hermes_started = False
    stop_event = threading.Event()

    def cleanup_processes():
        """Simple cleanup function."""
        for p in procs:
            if p.poll() is None:
                try:
                    os.killpg(os.getpgid(p.pid), signal.SIGTERM)
                    p.wait(timeout=3)
                except (ProcessLookupError, OSError, subprocess.TimeoutExpired):
                    try:
                        os.killpg(os.getpgid(p.pid), signal.SIGKILL)
                    except (ProcessLookupError, OSError):
                        pass

    atexit.register(cleanup_processes)

    # If block_speed is provided, patch all config.toml files (let errors bubble up)
    if block_speed:
        for chain in cfg['chains']:
            for node in chain['nodes']:
                config_toml_path = Path(node['home']) / 'config' / 'config.toml'
                doc = tomlkit.parse(config_toml_path.read_text())
                consensus = doc.setdefault('consensus', tomlkit.table())
                consensus['timeout_commit'] = str(block_speed)
                config_toml_path.write_text(tomlkit.dumps(doc))

    for chain in cfg['chains']:
        for node in chain['nodes']:
            log_path = os.path.join(node['home'], 'node.log')
            log_file = open(log_path, 'w')
            p = subprocess.Popen([bin_path, 'start', '--home', node['home'], *extra_args], preexec_fn=os.setsid, stdout=log_file, stderr=log_file)
            procs.append(p)


    hcfg = Path(cfg['base_dir']) / 'hermes' / 'config.toml'
    if hcfg.exists() and shutil.which('hermes'):
        time.sleep(1)
        click.echo(f"Starting Hermes relayer with config: {hcfg}")
        hermes_log_path = Path(cfg['base_dir']) / 'hermes' / 'hermes.log'
        hermes_log_file = open(hermes_log_path, 'w')
        hermes_proc = subprocess.Popen(['hermes', '--config', str(hcfg), 'start'], preexec_fn=os.setsid, stdout=hermes_log_file, stderr=hermes_log_file)
        procs.append(hermes_proc)
        hermes_started = True
    elif hcfg.exists():
        click.echo("Warning: Hermes binary not found in PATH, skipping Hermes start.")
    
    click.echo(f"Nodes {'and Hermes ' if hermes_started else ''}started. Ctrl+C to stop.")

    def get_rpc_url(node):
        config_path = Path(node['home']) / 'config' / 'config.toml'
        doc = tomlkit.parse(config_path.read_text())
        rpc_addr = str(doc.get('rpc', {}).get('laddr', 'tcp://127.0.0.1:26657'))
        if rpc_addr.startswith('tcp://'):
            rpc_addr = 'http://' + rpc_addr[6:]
        return rpc_addr

    def get_block_height(rpc_url):
        try:
            resp = requests.get(f"{rpc_url}/status", timeout=3)
            if resp.status_code == 200:
                return int(resp.json()['result']['sync_info']['latest_block_height'])
        except Exception:
            return None
        return None

    def monitor_blocks(timeout):
        node_infos = []
        for chain in cfg['chains']:
            for node in chain['nodes']:
                node_infos.append({'moniker': node['moniker'], 'chain_id': chain['chain_id'], 'rpc': get_rpc_url(node)})
        prev_heights = {}
        last_change = time.time()
        try:
            while not stop_event.is_set():
                time.sleep(timeout / 2)
                changed = False
                for info in node_infos:
                    h = get_block_height(info['rpc'])
                    key = f"{info['chain_id']}-{info['moniker']}"
                    if h is not None:
                        if key in prev_heights and h > prev_heights[key]:
                            changed = True
                            last_change = time.time()
                        prev_heights[key] = h
                if time.time() - last_change > timeout:
                    click.echo(f"No new blocks produced by any node in {timeout} seconds. Stopping all nodes.")
                    stop_event.set()
                    return
        except Exception as e:
            click.echo(f"Exception in block monitor: {e}")
            stop_event.set()
            return

    if no_blocks_timeout:
        t = threading.Thread(target=monitor_blocks, args=(no_blocks_timeout,), daemon=True)
        t.start()

    def handle_signal(sig, frame):
        cleanup_processes()
        sys.exit(0)

    signal.signal(signal.SIGINT, handle_signal)
    signal.signal(signal.SIGTERM, handle_signal)

    try:
        for p in procs:
            if p.args[0] == bin_path:
                while True:
                    if stop_event.is_set():
                        cleanup_processes()
                        sys.exit(1)
                    ret = p.poll()
                    if ret is not None:
                        break
                    time.sleep(0.5)
    except Exception as e:
        cleanup_processes()
        raise

@chainnet.command()
@click.option('--config-file', default=DEFAULT_CONFIG_PATH, type=click.Path(exists=True))
def ibc(config_file):
    """Create IBC channels between consecutive chains."""
    cfg = json.loads(Path(config_file).read_text())
    if not cfg['chains'] or len(cfg['chains']) < 2:
        click.echo("IBC setup requires at least two chains. Skipping.")
        return

    chains_ids = [c['chain_id'] for c in cfg['chains']]
    hcfg = Path(cfg['base_dir']) / 'hermes' / 'config.toml'
    if not hcfg.exists():
        click.echo("Hermes config missing, cannot create IBC channels.")
        raise RuntimeError("Hermes config missing")

    if not shutil.which('hermes'):
        click.echo("Hermes binary not found in PATH, skipping IBC setup.")
        return
    
    # verify both chains are making blocks within the timeout
    some_node_not_ready = True
    while some_node_not_ready:
        some_node_not_ready = False
        for chain in cfg['chains']:
            for node in chain['nodes']:
                rpc_url = f"http://localhost:{node['ports']['rpc']}"
                resp = requests.get(f"{rpc_url}/status")
                if resp.status_code != 200:
                    click.echo(f"Chain {chain['chain_id']} is not running. waiting...")
                    time.sleep(1)
                    some_node_not_ready = True
    

    for i in range(len(chains_ids) - 1):
        chain_a_id = chains_ids[i]
        chain_b_id = chains_ids[i+1]
        click.echo(f"Creating IBC channel between {chain_a_id} and {chain_b_id}")
        try:
            subprocess.run([
                'hermes', '--config', str(hcfg), 'create', 'channel',
                '--a-chain', chain_a_id, '--b-chain', chain_b_id,
                '--a-port', 'transfer', '--b-port', 'transfer',
                '--new-client-connection', '--yes'
            ], check=True, capture_output=True, text=True)
            click.echo(f"Successfully created channel between {chain_a_id} and {chain_b_id}")
        except subprocess.CalledProcessError as e:
            click.echo(f"Error creating IBC channel between {chain_a_id} and {chain_b_id}:")
            click.echo(f"Stdout: {e.stdout}")
            click.echo(f"Stderr: {e.stderr}")
            # Optionally, re-raise or handle more gracefully
            # raise # Re-raise the exception to halt execution if needed
    click.echo("IBC connection attempts complete.")

if __name__ == '__main__':
    chainnet()
