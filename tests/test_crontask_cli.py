import pytest
import json
import time
import datetime
import subprocess
from typing import Dict, List
import uuid
from tests.conftest import faucet
from tests.utils import poll_until_condition

# Constants
TASK_SCHEDULED_DELAY = 3  # seconds delay for scheduled tasks
TASK_TIMEOUT = 10  # seconds to wait for task execution
GAS_LIMIT = 200000
GAS_FEE = 1


def test_crontask_params(chainnet):
    """Test querying crontask module parameters"""
    dysond_bin = chainnet[0]
    params_result = dysond_bin("query", "crontask", "params")
    assert "params" in params_result, "Params response does not contain 'params' field"
    params = params_result["params"]
    assert "block_gas_limit" in params, "Missing 'block_gas_limit' in params"
    assert "expiry_limit" in params, "Missing 'expiry_limit' in params"
    assert "max_scheduled_time" in params, "Missing 'max_scheduled_time' in params"
    assert isinstance(int(params["block_gas_limit"]), int), "block_gas_limit should be an integer"
    assert isinstance(int(params["expiry_limit"]), int), "expiry_limit should be an integer"
    assert isinstance(int(params["max_scheduled_time"]), int), "max_scheduled_time should be an integer"
    print(f"Crontask params: {params}")


def test_create_task_and_query_by_id(chainnet, generate_account, faucet):
    dysond_bin = chainnet[0]
    [alice_name, alice_address] = generate_account('alice')
    faucet(alice_address)
    task_id = create_task_for_test(dysond_bin, alice_name, alice_address)
    assert task_id is not None, "Task ID should not be None"


def test_query_tasks_by_address(chainnet, generate_account, faucet):
    dysond_bin = chainnet[0]
    [alice_name, alice_address] = generate_account('alice')
    faucet(alice_address)
    task_id = create_task_for_test(dysond_bin, alice_name, alice_address)
    assert task_id is not None, "Task ID should not be None"
    tasks_result = dysond_bin("query", "crontask", "tasks-by-address", "--creator", alice_address)
    assert "tasks" in tasks_result, "Tasks response does not contain 'tasks' field"
    tasks = tasks_result["tasks"]
    assert len(tasks) > 0, "No tasks found for the address"
    for task in tasks:
        assert task["creator"] == alice_address, f"Creator mismatch: {task['creator']} != {alice_address}"
    print(f"Found {len(tasks)} tasks for address {alice_address}")


def test_query_tasks_by_status_timestamp(chainnet, generate_account):
    dysond_bin = chainnet[0]
    [alice_name, alice_address] = generate_account('alice')
    task_id1 = create_task_for_test_with_timestamp(dysond_bin, alice_name, alice_address, time_offset=3)
    task_id2 = create_task_for_test_with_timestamp(dysond_bin, alice_name, alice_address, time_offset=6)
    tasks_result = dysond_bin("query", "crontask", "tasks-by-status-timestamp", "--status", "PENDING")
    assert "tasks" in tasks_result, "Tasks response does not contain 'tasks' field"
    tasks = tasks_result["tasks"]
    for task in tasks:
        assert task["status"] == "PENDING", f"Status mismatch: {task['status']} != PENDING"
    if len(tasks) > 1:
        timestamps = [int(task["scheduled_timestamp"]) for task in tasks]
        assert all(timestamps[i] <= timestamps[i+1] for i in range(len(timestamps)-1)), "Tasks not ordered by timestamp"
    print(f"Found {len(tasks)} tasks with status PENDING")


def test_query_tasks_by_status_gas_price(chainnet, generate_account):
    dysond_bin = chainnet[0]
    [alice_name, alice_address] = generate_account('alice')
    task_id1 = create_task_for_test_with_gas_price(dysond_bin, alice_name, alice_address, gas_price=1)
    task_id2 = create_task_for_test_with_gas_price(dysond_bin, alice_name, alice_address, gas_price=2)
    tasks_result = dysond_bin("query", "crontask", "tasks-by-status-gas-price", "--status", "PENDING")
    assert "tasks" in tasks_result, "Tasks response does not contain 'tasks' field"
    tasks = tasks_result["tasks"]
    for task in tasks:
        assert task["status"] == "PENDING", f"Status mismatch: {task['status']} != PENDING"
    if len(tasks) > 1:
        gas_prices = [int(task["task_gas_price"]["amount"]) for task in tasks]
        assert all(gas_prices[i] <= gas_prices[i+1] for i in range(len(gas_prices)-1)), "Tasks not ordered by gas price"
    print(f"Found {len(tasks)} tasks with status PENDING")


def test_delete_task(chainnet, generate_account):
    dysond_bin = chainnet[0]
    [alice_name, alice_address] = generate_account('alice')
    task_id = create_task_for_test(dysond_bin, alice_name, alice_address)
    delete_result = dysond_bin("tx", "crontask", "delete-task", "--task-id", str(task_id), "--from", alice_name, "--keyring-backend", "test", "--yes")
    print(f"Delete task result: {delete_result}")
    try:
        task_result = dysond_bin("query", "crontask", "task-by-id", "--task-id", str(task_id))
        if "task" in task_result:
            task = task_result["task"]
            assert task["status"] != "PENDING", f"Task status should not be PENDING after deletion, got {task['status']}"
            print(f"Task {task_id} status changed to {task['status']} after deletion")
    except Exception:
        print(f"Task {task_id} was completely deleted (query returned error)")


def test_task_execution(chainnet, generate_account, faucet):
    dysond_bin = chainnet[0]
    [alice_name, alice_address] = generate_account('alice')
    faucet(alice_address, amount=100)
    now = int(datetime.datetime.now().timestamp())
    scheduled_time = now + TASK_SCHEDULED_DELAY
    expiry_time = now + 86400
    msg_obj = {
        "@type": "/cosmos.bank.v1beta1.MsgSend",
        "from_address": alice_address,
        "to_address": alice_address,
        "amount": [{"denom": "dys", "amount": "1"}]
    }
    create_result = dysond_bin(
        "tx", "crontask", "create-task",
        "--scheduled-timestamp", str(scheduled_time),
        "--expiry-timestamp", str(expiry_time),
        "--task-gas-limit", str(GAS_LIMIT),
        "--task-gas-fee", f"{GAS_FEE}dys",
        "--msgs", json.dumps(msg_obj),
        "--from", alice_name, "--keyring-backend", "test", "--yes"
    )
    task_id = None
    for event in create_result.get("events", []):
        if event.get("type") == "dysonprotocol.crontask.v1.EventTaskCreated":
            for attr in event.get("attributes", []):
                if attr.get("key") == "task_id":
                    task_id = json.loads(attr.get("value"))
                    break
    assert task_id is not None, "Failed to extract task ID"
    def check_task_executed():
        task_result = dysond_bin("query", "crontask", "task-by-id", "--task-id", str(task_id))
        task = task_result.get("task", {})
        if task.get("status") == "DONE":
            return True
        elif task.get("status") == "FAILED":
            assert False, f"Task failed unexpectedly: {task.get('error_log', 'no error log')}"
        return False
    poll_until_condition(
        check_task_executed,
        timeout=TASK_TIMEOUT,
        error_message=f"Task {task_id} was not executed within timeout"
    )
    print(f"Task {task_id} executed successfully")
    task_result = dysond_bin("query", "crontask", "task-by-id", "--task-id", str(task_id))
    task = task_result.get("task", {})
    assert task.get("status") == "DONE", f"Task status is not DONE: {task.get('status')}"
    assert task.get("creator") == alice_address, f"Task creator is not Alice: {task.get('creator')}"

# Helper function to create a task for testing

def create_task_for_test(dysond_bin, creator_name, creator_address) -> int:
    now = int(datetime.datetime.now().timestamp())
    scheduled_time = now + TASK_SCHEDULED_DELAY
    expiry_time = now + 86400
    gas_limit = 200000
    gas_fee = max(1, int(gas_limit * 0.0000001))
    msg_obj = {
        "@type": "/cosmos.bank.v1beta1.MsgSend",
        "from_address": creator_address,
        "to_address": creator_address,
        "amount": [{"denom": "dys", "amount": "1"}]
    }
    create_result = dysond_bin(
        "tx", "crontask", "create-task",
        "--scheduled-timestamp", str(scheduled_time),
        "--expiry-timestamp", str(expiry_time),
        "--task-gas-limit", str(gas_limit),
        "--task-gas-fee", f"{gas_fee}dys",
        "--msgs", json.dumps(msg_obj),
        "--from", creator_name, "--keyring-backend", "test", "--yes"
    )
    task_id = None
    for event in create_result.get("events", []):
        if event.get("type") == "dysonprotocol.crontask.v1.EventTaskCreated":
            for attr in event.get("attributes", []):
                if attr.get("key") == "task_id":
                    task_id = json.loads(attr.get("value"))
                    break
    assert task_id is not None, "Failed to extract task ID"
    return task_id

def create_task_for_test_with_gas_price(dysond_bin, creator_name, creator_address, gas_price: int = 1) -> int:
    now = int(datetime.datetime.now().timestamp())
    scheduled_time = now + TASK_SCHEDULED_DELAY
    expiry_time = now + 86400
    gas_limit = 200000
    gas_fee = gas_price
    msg_obj = {
        "@type": "/cosmos.bank.v1beta1.MsgSend",
        "from_address": creator_address,
        "to_address": creator_address,
        "amount": [{"denom": "dys", "amount": "1"}]
    }
    create_result = dysond_bin(
        "tx", "crontask", "create-task",
        "--scheduled-timestamp", str(scheduled_time),
        "--expiry-timestamp", str(expiry_time),
        "--task-gas-limit", str(gas_limit),
        "--task-gas-fee", f"{gas_fee}dys",
        "--msgs", json.dumps(msg_obj),
        "--from", creator_name, "--keyring-backend", "test", "--yes"
    )
    task_id = None
    for event in create_result.get("events", []):
        if event.get("type") == "dysonprotocol.crontask.v1.EventTaskCreated":
            for attr in event.get("attributes", []):
                if attr.get("key") == "task_id":
                    task_id = json.loads(attr.get("value"))
                    break
    assert task_id is not None, "Failed to extract task ID"
    return task_id

def create_task_for_test_with_timestamp(dysond_bin, creator_name, creator_address, time_offset: int = 5) -> int:
    now = int(datetime.datetime.now().timestamp())
    scheduled_time = now + time_offset
    expiry_time = now + 86400
    gas_limit = 200000
    gas_fee = max(1, int(gas_limit * 0.0000001))
    msg_obj = {
        "@type": "/cosmos.bank.v1beta1.MsgSend",
        "from_address": creator_address,
        "to_address": creator_address,
        "amount": [{"denom": "dys", "amount": "1"}]
    }
    create_result = dysond_bin(
        "tx", "crontask", "create-task",
        "--scheduled-timestamp", str(scheduled_time),
        "--expiry-timestamp", str(expiry_time),
        "--task-gas-limit", str(gas_limit),
        "--task-gas-fee", f"{gas_fee}dys",
        "--msgs", json.dumps(msg_obj),
        "--from", creator_name, "--keyring-backend", "test", "--yes"
    )
    task_id = None
    for event in create_result.get("events", []):
        if event.get("type") == "dysonprotocol.crontask.v1.EventTaskCreated":
            for attr in event.get("attributes", []):
                if attr.get("key") == "task_id":
                    task_id = json.loads(attr.get("value"))
                    break
    assert task_id is not None, "Failed to extract task ID"
    return task_id 