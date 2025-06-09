import pytest
import json
import time
import datetime
from test_crontask_cli import TASK_SCHEDULED_DELAY, TASK_TIMEOUT
from typing import Dict, Any, List
from tests.utils import poll_until_condition

# Function to get blockchain time
def get_blockchain_time(dysond_bin):
    """Get the current blockchain time from the node status."""
    status = dysond_bin("status")
    latest_block_time = status.get("SyncInfo", {}).get("latest_block_time", "")
    if latest_block_time:
        # Parse the time (format: 2023-10-01T12:34:56.789Z)
        dt = datetime.datetime.strptime(latest_block_time, "%Y-%m-%dT%H:%M:%S.%fZ")
        return int(dt.timestamp())
    return int(time.time())  # Fallback if parsing fails

# Test for successful fee deduction
def test_fee_deduction_success(chainnet, generate_account):
    dysond_bin = chainnet[0]
    [alice_name, alice_address] = generate_account('alice')
    [bob_name, bob_address] = generate_account('bob')
    balance_before = dysond_bin("query", "bank", "balances", alice_address)
    dys_balance_before = next((coin["amount"] for coin in balance_before["balances"] if coin["denom"] == "dys"), "0")
    
    print(f"Initial balance: {dys_balance_before} dys")
    
    # Set up test parameters using blockchain time
    current_time = get_blockchain_time(dysond_bin)
    scheduled_time = current_time + 3  # 3 seconds from now
    expiry_time = scheduled_time + 86400  # 1 day later
    gas_limit = 200000
    gas_fee = 1
    
    # Create a simple send message to send 1 dys from Alice to Bob
    msg_obj = {
        "@type": "/cosmos.bank.v1beta1.MsgSend",
        "from_address": alice_address,
        "to_address": bob_address, 
        "amount": [{"denom": "dys", "amount": "1"}]
    }
    
    # Create the task using execute_tx_and_wait directly
    create_result = dysond_bin(
        "tx", "crontask", "create-task",
        "--scheduled-timestamp", str(scheduled_time),
        "--expiry-timestamp", str(expiry_time),
        "--task-gas-limit", str(gas_limit),
        "--task-gas-fee", f"{gas_fee}dys",
        "--msgs", json.dumps(msg_obj),
        "--from", alice_name, 
    )
    
    # Extract the task ID
    task_id = None
    for event in create_result.get("events", []):
        if event.get("type") == "dysonprotocol.crontask.v1.EventTaskCreated":
            for attr in event.get("attributes", []):
                if attr.get("key") == "task_id":
                    task_id = json.loads(attr.get("value"))
                    break
    
    assert task_id is not None, "Failed to extract task ID"
    
    # Define a check function for polling task completion
    def check_task_executed():
        task_result = dysond_bin("query", "crontask", "task-by-id", "--task-id", str(task_id))
        task = task_result.get("task", {})
        
        if task.get("status") == "DONE":
            return True
        elif task.get("status") == "FAILED":
            assert False, f"Task failed unexpectedly: {task.get('error_log', 'no error log')}"
        return False
    
    # Poll until task is executed
    poll_until_condition(
        check_task_executed,
        timeout=TASK_TIMEOUT,
        error_message="Task was not executed within timeout"
    )
    print(f"Task {task_id} executed successfully")
        
    # Define a check function for polling balance updates
    def check_balance_updated():
        nonlocal balance_after_int_result
        balance_after = dysond_bin("query", "bank", "balances", alice_address)
        dys_balance_after = next((coin["amount"] for coin in balance_after["balances"] if coin["denom"] == "dys"), "0")
        
        balance_before_int = int(dys_balance_before)
        balance_after_int = int(dys_balance_after)
        
        # If balance has been reduced (fee deducted), return True
        if balance_before_int > balance_after_int:
            # Store the updated balance for later comparison
            balance_after_int_result = balance_after_int
            return True
        return False
    
    # Use a nonlocal variable to store the result
    balance_after_int_result = 0
    
    # Poll until balance is updated
    poll_until_condition(
        check_balance_updated,
        timeout=10,  # 10 seconds should be enough
        error_message="Balance was not updated within timeout period"
    )
    print(f"Balance was successfully updated")
    
    # Calculate the balance difference
    balance_before_int = int(dys_balance_before)
    balance_diff = balance_before_int - balance_after_int_result
    
    # The difference should be at least the gas fee plus the sent amount
    expected_diff = gas_fee + 1  # gas fee + 1 dys sent to Bob
    assert balance_diff >= expected_diff, f"Expected fee deduction of at least {expected_diff}, but got {balance_diff}"
    
    print(f"Successfully verified fee deduction: {balance_diff} dys")


# Test for insufficient funds during task execution
def test_fee_deduction_insufficient_funds(chainnet, generate_account):
    dysond_bin = chainnet[0]
    [alice_name, alice_address] = generate_account('alice')
    [poor_name, poor_address] = generate_account('poor_account', faucet_amount=1)
    # Fund poor_account with 1 dys
    
    # Calculate timestamps using blockchain time
    now = get_blockchain_time(dysond_bin)
    scheduled_time = now + TASK_SCHEDULED_DELAY
    expiry_time = now + 86400
    
    # Task parameters
    gas_limit = 200000
    gas_fee = max(2, int(gas_limit * 0.0000001))  # Set fee higher than balance (at least 2 dys)
    
    # Create a task that should fail during execution due to insufficient funds
    msg_obj = {
        "@type": "/cosmos.bank.v1beta1.MsgSend",
        "from_address": poor_address,
        "to_address": poor_address,
        "amount": [{"denom": "dys", "amount": "10"}]
    }
    
    # Create the task using execute_tx_and_wait directly
    create_result = dysond_bin(
        "tx", "crontask", "create-task",
        "--scheduled-timestamp", str(scheduled_time),
        "--expiry-timestamp", str(expiry_time),
        "--task-gas-limit", str(gas_limit),
        "--task-gas-fee", f"{gas_fee}dys",
        "--msgs", json.dumps(msg_obj),
        "--from", poor_name, 
    )
    print(f"Task creation succeeded with result: {json.dumps(create_result, indent=2)}")
    assert create_result["code"] == 0, f"Task creation failed unexpectedly: {create_result['raw_log']}"
    
    # Extract the task ID from the events
    task_id = None
    for event in create_result.get("events", []):
        if event.get("type") == "dysonprotocol.crontask.v1.EventTaskCreated":
            for attr in event.get("attributes", []):
                if attr.get("key") == "task_id":
                    task_id = json.loads(attr.get("value"))
                    break
    assert task_id is not None, "Failed to extract task ID from creation result"
    print(f"Created task with ID: {task_id}")
    
    # Define a check function for polling task status
    def check_task_failed():
        task_result = dysond_bin("query", "crontask", "task-by-id", "--task-id", str(task_id))
        task = task_result.get("task", {})
        if task.get("status") == "FAILED":
            error_log = task.get("error_log", "")
            assert "failed to deduct gas fee" in error_log or "insufficient funds" in error_log.lower(), \
                f"Task failed but not due to insufficient funds: {error_log}"
            print(f"Task {task_id} failed as expected due to insufficient funds: {error_log}")
            return True
        elif task.get("status") == "DONE":
            assert False, f"Task {task_id} executed successfully when it should have failed due to insufficient funds"
        return False
    
    # Poll until task status is updated to FAILED
    poll_until_condition(
        check_task_failed,
        timeout=TASK_TIMEOUT,
        error_message="Task did not fail as expected within timeout"
    )
    print("Successfully verified task execution fails with insufficient funds") 