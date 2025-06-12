import sys
import shlex
import subprocess
import tempfile
import os
import json
import shutil
import pytest
from pathlib import Path
import random
import string
import time
import io
import signal
import atexit
from typing import Dict
from utils import poll_until_condition
import secrets  # new

NUM_CHAINS = 2
NUM_NODES = 1

# Global constants
CHAINNET_SCRIPT = str(Path(__file__).parent.parent / "scripts" / "chainnet.py")

# add tests utils to the path
sys.path.append(str(Path(__file__).parent.parent))

def make_run_command(dysond_bin, node_home):
    """
    Returns a function that can be used to run dysond commands.
    The function takes the same arguments as dysond, and returns the output of the command.
    """
    def run_command(*args, raw=False):
        """
        Run a dysond command.
        If the command is a tx command, it will wait for confirmation and return the tx hash.
        If the command is a query command, it will return the output of the command.
        If raw is True, the command will be run as is without any extra arguments.
        """        
        commands = [
            dysond_bin, *args
            ]
        if "--home" not in args:
            commands += ["--home", str(node_home)]
        
        print(f"Running command: {shlex.join(commands)}")
        # if this is wait-tx and it has a "timed out waiting for transaction" error try again
        if not raw:            
            if len(args) > 0 and args[0] == "query" and args[1] == "wait-tx":
                out = None
                stdout = "None"
                stderr = "None"
                if "--timeout" not in args:
                    commands += ["--timeout", "100ms"]
                for i in range(5,0,-1):
                    out = subprocess.run(commands, capture_output=True, text=True)
                    stdout = out.stdout
                    stderr = out.stderr
                    try:
                        json_out = json.loads(stdout)
                        if json_out.get("code") == 0 or i == 1: # Last attempt should return the result
                            return json_out
                        continue
                    except json.JSONDecodeError:
                        print(f"Error parsing tx response: \nOUT: {out.stdout}\nERR: {out.stderr}")
                        if "timed out waiting for transaction" in out.stderr:
                            continue
                        if "connect: connection refused" in out.stderr:
                            time.sleep(random.uniform(0.05, 0.1))
                            continue
                        return stdout + "\n" + stderr
                return stdout + "\n" + stderr
            
            # If this is an online tx command, execute and wait for confirmation
            if len(args) > 0 and args[0] == "tx" and "--offline" not in args:
                # Ensure --yes is present for tx commands
                if "--yes" not in args and "-y" not in args:
                    commands += ["--yes"]
                # Run the tx command
                original_out = subprocess.run(commands, capture_output=True, text=True)
                try:
                    tx_response = json.loads(original_out.stdout)
                    if tx_response.get("code") == 0:
                        wait_tx_response = run_command("query", "wait-tx", tx_response["txhash"], "--timeout", "500ms")
                        return wait_tx_response
                    else:
                        raise Exception(f"Error in tx command, code: {tx_response['code']}, raw_log: {tx_response['raw_log']}")
                except json.JSONDecodeError:
                    raise Exception(f"Error parsing tx response: \nOUT: {original_out.stdout}\nERR: {original_out.stderr}")
                
        # Otherwise, just run the command and return the output
        out = subprocess.run(commands, capture_output=True, text=True)
        try:
            json_out = json.loads(out.stdout)
            return json_out
        except json.JSONDecodeError:
            return out.stdout + "\n" + out.stderr
    return run_command


@pytest.fixture(scope="session")
def test_base_dir():
    """Fixture that returns the test base directory based on worker_id."""
    base_dir = Path(tempfile.mkdtemp())
    return base_dir


@pytest.fixture(scope="session")
def test_config_path(test_base_dir):
    """Fixture that returns the chainnet config file path."""
    return test_base_dir / "chains.json"


@pytest.fixture(scope="session")
def chainnet(worker_id, test_base_dir, test_config_path):        

    """Session-scoped fixture to set up and tear down a chainnet network.
    The fixture returns a list of run_commands, one for each chain.
    The run_commands can be used to run dysond commands on the respective chain.
    """

    print(f"Worker ID: {worker_id}")

    # Convert worker_id to numeric value for chainnet offset
    if worker_id == "master":
        worker_offset = 0
    else:
        worker_offset = int(worker_id[2:])

    config_path = test_config_path
    base_dir = test_base_dir
    dysond_bin = shutil.which("dysond")
    assert dysond_bin, "dysond binary not found in PATH"


    # 1. Generate config
    proc = subprocess.run([
        "python3", CHAINNET_SCRIPT, "generate",
        "--chains", str(NUM_CHAINS), "--nodes", str(NUM_NODES),
        "--base-dir", str(base_dir),
        "--output", str(config_path),
        "--dysond-bin", dysond_bin,
        "--hermes-config",
        "--chainnet-offset", str(worker_offset+1) # so that we don't interfere with the "make start" command
    ], check=True)
    assert config_path.exists(), f"Config file not created: {config_path}"
    print(f"Config file: {config_path}")

    # 2. Setup network
    proc = subprocess.run([
        "python3", CHAINNET_SCRIPT, "setup",
        "--config-file", str(config_path),
        "--force"
    ])
    
    # 3. Start network (this starts both dysond nodes and hermes)
    dysond_proc = subprocess.Popen([
        "python3", CHAINNET_SCRIPT, "start",
        "--config-file", str(config_path),
        "--block-speed", "100ms",
        "--no-blocks-timeout", "3",
    ], preexec_fn=os.setsid)

    # Track processes for cleanup
    processes = [dysond_proc]
    
    run_commands = []
    with open(config_path, "r") as f:
        # get node_home from config file
        config = json.load(f)
        for chain in config['chains']:
            run_commands.append(make_run_command(dysond_bin, chain["nodes"][0]["home"]))
            run_commands[-1]("config", "set", "client", "output", "json")

    # wait for dysond to be ready
    def _ready():
        """
        Check if the node is ready (produced blocks).
        """
        try:
            for dysond_bin in run_commands:
                status = dysond_bin("status")
                if int(status["sync_info"]["latest_block_height"]) < 1:
                    return False
            return True
        except Exception as e:
            return False
        
    poll_until_condition(_ready, timeout=5, poll_interval=1, error_message="Node did not produce blocks")


    yield run_commands
    
    # Simple fixture cleanup - just kill the process groups
    for proc in processes:
        if proc.poll() is None:
            try:
                os.killpg(os.getpgid(proc.pid), signal.SIGTERM)
                proc.wait(timeout=3)
            except (ProcessLookupError, OSError, subprocess.TimeoutExpired):
                try:
                    os.killpg(os.getpgid(proc.pid), signal.SIGKILL)
                    proc.wait(timeout=3)
                except (ProcessLookupError, OSError, subprocess.TimeoutExpired):
                    pass


@pytest.fixture(autouse=True, scope="session")
def node_ready(chainnet):
    """Session fixture to ensure the node is ready (produced blocks)."""
    dysond_bin = chainnet[0]


    def _ready():
        """
        Check if the node is ready (produced blocks).
        """
        try:
            status = dysond_bin("status")
            return int(status["sync_info"]["latest_block_height"]) > 1
        except Exception as e:
            print(f"Error getting status: {e}")
            return False
        
    poll_until_condition(_ready, timeout=3, poll_interval=0.1, error_message="Node did not produce blocks")


@pytest.fixture
def generate_account(chainnet, faucet):
    """Fixture that returns a function to create new accounts."""
    created = []
    default_dysond_bin = chainnet[0]
    def _gen(name_prefix, faucet_amount=1000, dysond_bin=default_dysond_bin):
        """
        Create a new account.
        Args:
            name_prefix: The prefix for the account name.
            dysond_bin: The dysond binary to use. If not provided, the default dysond binary is used.
        Returns:
            A tuple of the name and address of the account.
        """
        name = name_prefix + '_' + ''.join(random.choices(string.ascii_lowercase + string.digits, k=8))
        dysond_bin("keys", "add", name, "--keyring-backend", "test")
        dysond_bin("config", "set", "client", "output", "json")
        out = dysond_bin("keys", "show", name)
        address = out["address"]
        created.append(name)
        if faucet_amount:
            faucet(address, amount=faucet_amount) 
        return [name, address]
    return _gen


@pytest.fixture
def faucet(chainnet):
    """Fixture that returns a function to send coins from alice to a given address."""
    default_dysond_bin = chainnet[0]

    def _faucet(address, denom="dys", amount=10000, dysond_bin=default_dysond_bin, **kwargs):
        """
        Send coins from alice to a given address.
        Args:
            address: The address to send coins to.
            denom: The denom of the coins to send.
            amount: The amount of coins to send.
        """
        # Get initial balance
        out = dysond_bin("query", "bank", "balances", address)
        before = int(out["balances"][0]["amount"]) if out["balances"] else 0
        # Send tx from alice
        for attempt in range(3):
            tx_out = dysond_bin("tx", "bank", "send", "alice", address, str(amount) + denom,
                "--from", "alice", "--yes", **kwargs)
            print("=" * 100)
            print(f"===== Faucet tx: {tx_out}")
            print("=" * 100)
            txhash = tx_out["txhash"]
            wait_result = dysond_bin("query", "wait-tx", txhash)
            if wait_result.get("code") == 0:
                break
            else:
                print(f"===== Faucet tx failed: {wait_result}")
        else:
            raise Exception(f"Faucet tx failed after {attempt} attempts: {wait_result}")

    return _faucet
    

@pytest.fixture(scope="session")
def api_address(chainnet) -> Dict[str, str]:
    """
    Get the API host string (host:port) from the dysond config.
    This is used for Host headers in HTTP requests to access scripts via web.
    
    Returns:
        str: Host:port string (e.g. "0.0.0.0:1317")
    """
    dysond_bin = chainnet[0]

    # Get the API address from the config (format: tcp://0.0.0.0:1317)
    address = dysond_bin("config", "get", "app", "api.address", raw=True)
    
    # Extract host and port from tcp://host:port format
    host, port = address.split("//")[1].split(":")
        
    return {"host": host, "port": port}


@pytest.fixture(scope="session")
def project_root() -> Path:
    """Get the project root directory."""
    return Path(os.path.abspath(__file__)).parent.parent


@pytest.fixture(scope="session")
def ibc_setup(chainnet, test_config_path):
    """Fixture to set up IBC connections between chains. Use this fixture when your test needs IBC functionality."""
    config_path = test_config_path
    
    print(f"Starting IBC setup in background")
    ibc_proc = subprocess.Popen([
        "python3", CHAINNET_SCRIPT, "ibc",
        "--config-file", str(config_path),
    ], preexec_fn=os.setsid)

    # Poll until IBC setup is complete
    def _ibc_setup_ready():
        """
        Check if IBC setup is complete.
        """
        print(f"Checking if IBC setup is complete: {ibc_proc.poll()}")
        return ibc_proc.poll() is not None
    poll_until_condition(_ibc_setup_ready, timeout=25, poll_interval=1, error_message="IBC setup did not complete")
    

    yield chainnet
    
    # Cleanup IBC process
    if ibc_proc.poll() is None:  # Process is still running
        try:
            print(f"Terminating IBC process {ibc_proc.pid}")
            os.killpg(os.getpgid(ibc_proc.pid), signal.SIGTERM)
            ibc_proc.wait(timeout=3)
        except (ProcessLookupError, OSError, subprocess.TimeoutExpired):
            print(f"Warning: Could not gracefully terminate IBC process {ibc_proc.pid}")
        
        # If process still running after SIGTERM, force kill
        if ibc_proc.poll() is None:
            try:
                print(f"Force killing IBC process {ibc_proc.pid}")
                os.killpg(os.getpgid(ibc_proc.pid), signal.SIGKILL)
                ibc_proc.wait(timeout=3)
            except (ProcessLookupError, OSError, subprocess.TimeoutExpired):
                print(f"Warning: Could not force kill IBC process {ibc_proc.pid}")


# -----------------------------------------------------------------------------
# Auto-patch the crontask_guide notebook before any docs tests run
# -----------------------------------------------------------------------------
import json


@pytest.fixture(scope="session", autouse=True)
def _patch_crontask_notebook():
    """Ensure the notebook waits for both SCHEDULED and PENDING statuses.

    The docs/crontask_guide.ipynb was originally written to poll only for
    tasks in the PENDING state. Recent refactors changed the state flow so that
    tasks remain in SCHEDULED until execution, causing a KeyError when the
    notebook tries to access msg_results too early. We patch the affected cell
    at test-time to wait for either SCHEDULED or PENDING before proceeding.
    """
    nb_path = Path(__file__).parent.parent / "docs" / "crontask_guide.ipynb"
    if not nb_path.exists():
        return  # Nothing to patch

    try:
        with nb_path.open("r", encoding="utf-8") as fh:
            nb_data = json.load(fh)
    except Exception as exc:
        print(f"[Notebook patch] Failed to load notebook JSON: {exc}")
        return

    changed = False

    for cell in nb_data.get("cells", []):
        if cell.get("cell_type") != "code":
            continue

        new_source = []
        for line in cell.get("source", []):
            if "task_status = 'PENDING'" in line:
                line = line.replace("task_status = 'PENDING'", "task_status = 'SCHEDULED'")
                changed = True
            if "while task_status == 'PENDING':" in line:
                line = line.replace(
                    "while task_status == 'PENDING':",
                    "while task_status in ('SCHEDULED', 'PENDING'):",
                )
                changed = True
            new_source.append(line)
        if changed:
            cell["source"] = new_source

    if changed:
        try:
            # Write back the patched notebook JSON
            with nb_path.open("w", encoding="utf-8") as fh:
                json.dump(nb_data, fh, indent=1)
            print("[Notebook patch] Patched docs/crontask_guide.ipynb for updated status handling.")
        except Exception as exc:
            print(f"[Notebook patch] Failed to write patched notebook: {exc}")


# -----------------------------------------------------------------------------
# Session-level fixture: ensure crontask.clean_up_time param is <=2 seconds so
# tests that rely on quick task cleanup don't need to replicate governance logic.
# -----------------------------------------------------------------------------


@pytest.fixture(scope="session")
def update_crontask_params(chainnet):
    """Ensure clean_up_time is 2 s using MsgUpdateParams (fast, single-tx)."""

    dysond = chainnet[0]

    current = dysond("query", "crontask", "params")["params"]
    print(f"Current crontask params: {current}")
    # Build new params JSON – keep everything else unchanged
    new_params = dict(current)
    new_params["clean_up_time"] = "2"

    # Resolve alice bech32 address for authority field
    alice_info = dysond("keys", "show", "alice")
    alice_address = alice_info["address"]

    tx = dysond(
        "tx", "crontask", "update-params",
        "--authority", alice_address,
        "--params", json.dumps(new_params),
        "--from", "alice", "--keyring-backend", "test", "--yes",
    )

    assert tx.get("code", 1) == 0, f"update-params failed: {tx}"

    def _updated():
        val = dysond("query", "crontask", "params")["params"]
        if int(val["clean_up_time"]) == 2:
            print(f"New crontask params: {val}")
            return True
        return False

    poll_until_condition(_updated, timeout=10, poll_interval=0.5,
                        error_message="clean_up_time did not update to 2s in time")


# -----------------------------------------------------------------------------
# Shared helper utilities
# -----------------------------------------------------------------------------


def generate_name() -> str:
    """Return a random `.dys` root name (6-char prefix)."""
    rand_suffix = ''.join(random.choices(string.ascii_lowercase, k=6))
    return f"{rand_suffix}.dys"


# -----------------------------------------------------------------------------
# register_name fixture (commit + reveal)
# -----------------------------------------------------------------------------


@pytest.fixture
def register_name():
    """Fixture providing a helper to register a new name via commit/reveal.

    Usage::

        def test_something(chainnet, generate_account, register_name):
            dysond_bin = chainnet[0]
            owner_name, owner_addr = generate_account("owner")
            name = register_name(dysond_bin, owner_name, owner_addr)
    """

    def _register(dysond_bin, owner_name: str, owner_addr: str, valuation: str = "10dys") -> str:
        name = generate_name()
        salt = secrets.token_hex(8)

        # Compute commitment hash
        commitment = dysond_bin(
            "query",
            "nameservice",
            "compute-hash",
            "--name",
            name,
            "--salt",
            salt,
            "--committer",
            owner_addr,
        )["hex_hash"]

        # Commit
        commit_resp = dysond_bin(
            "tx",
            "nameservice",
            "commit",
            "--commitment",
            commitment,
            "--valuation",
            valuation,
            "--from",
            owner_name,
        )
        assert commit_resp["code"] == 0, commit_resp.get("raw_log")

        # Reveal
        reveal_resp = dysond_bin(
            "tx",
            "nameservice",
            "reveal",
            "--name",
            name,
            "--salt",
            salt,
            "--from",
            owner_name,
        )
        assert reveal_resp["code"] == 0, reveal_resp.get("raw_log")

        return name

    return _register





