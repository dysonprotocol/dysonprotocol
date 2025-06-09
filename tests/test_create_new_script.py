#!/usr/bin/env python3
import json
import pytest
import tempfile
from typing import Dict, Any
from utils import poll_until_condition

def test_create_new_script(chainnet, generate_account, faucet):
    """
    Test the create-new-script command to create a new script with a deterministic address.
    This tests:
    1. Creating a new script with deterministic address
    2. Verifying the script exists with correct content
    3. Checking that authz grants were created for the creator
    """
    dysond_bin = chainnet[0]
    [alice_name, alice_address] = generate_account('alice')
    faucet(alice_address, denom="dys", amount="10")
    script_code = "def hello(): return 'Hello from new script!'"
    create_result = dysond_bin(
        "tx", "script", "create-new-script",
        "--code", script_code,
        "--from", alice_name, "--keyring-backend", "test", "--yes"
    )
    script_address = None
    for event in create_result["events"]:
        if event["type"] == "dysonprotocol.script.v1.EventCreateNewScript":
            for attr in event["attributes"]:
                if attr["key"] == "script_address":
                    script_address = json.loads(attr["value"])
                    break
    assert script_address, "Script address not found in transaction events"
    print(f"Created new script with address: {script_address}")
    script_info = dysond_bin("query", "script", "script-info", "--address", script_address)
    assert "script" in script_info, f"Script info not found: {script_info}"
    assert script_info["script"]["address"] == script_address, "Script address mismatch"
    assert script_info["script"]["code"] == script_code, "Script code mismatch"
    assert script_info["script"]["version"] == "1", "Script version should be 1"
    authz_info = dysond_bin("query", "authz", "grants", script_address, alice_address)
    assert "grants" in authz_info, f"No authz grants found: {authz_info}"
    assert len(authz_info["grants"]) > 0, "No authz grants found"
    found_update_grant = False
    for grant in authz_info["grants"]:
        if grant["authorization"]["type"] == "/cosmos.authz.v1beta1.GenericAuthorization":
            if grant["authorization"]["value"]["msg"] == "/dysonprotocol.script.v1.MsgUpdateScript":
                found_update_grant = True
                break
    assert found_update_grant, "No authz grant found for MsgUpdateScript"
    print("Successfully created new script with authz permissions")


def test_script_update_with_authz(chainnet, generate_account, faucet):
    """
    Test that a creator can update their scripts using the authz permission.
    Tests the full workflow with the authz exec command.
    """
    dysond_bin = chainnet[0]
    [alice_name, alice_address] = generate_account('alice')
    faucet(alice_address, denom="dys", amount="10")
    script_code = "def update_test(): return 'Initial version'"
    create_result = dysond_bin(
        "tx", "script", "create-new-script",
        "--code", script_code,
        "--from", alice_name, "--keyring-backend", "test", "--yes"
    )
    script_address = None
    for event in create_result["events"]:
        if event["type"] == "dysonprotocol.script.v1.EventCreateNewScript":
            for attr in event["attributes"]:
                if attr["key"] == "script_address":
                    script_address = json.loads(attr["value"])
                    break
    assert script_address, "Script address not found in transaction events"
    print(f"Created new script with address: {script_address}")
    authz_info = dysond_bin("query", "authz", "grants", script_address, alice_address)
    print(f"Authz grants from {script_address} to {alice_address}:")
    print(json.dumps(authz_info, indent=2))
    assert authz_info["grants"][0]["authorization"]["value"]["msg"] == "/dysonprotocol.script.v1.MsgUpdateScript", f"Authz info is not valid: {authz_info}"
    updated_code = "def update_test(): return 'Updated version'"
    update_script_msg = {
        "@type": "/dysonprotocol.script.v1.MsgUpdateScript",
        "address": script_address,
        "code": updated_code
    }
    tx = {
        "body": {
            "messages": [
                update_script_msg
            ],
        }
    }
    
    with tempfile.NamedTemporaryFile(mode='w', suffix='.json') as f:
        json.dump(tx, f, indent=2)
        f.flush()  # Ensure data is written to disk
        
        authz_exec_result = dysond_bin(
            "tx", "authz", "exec", f.name,
            "--from", alice_name, "--keyring-backend", "test", "--yes"
        )
    
    assert authz_exec_result["code"] == 0, f"Script update transaction failed: {authz_exec_result}"

    script_info = dysond_bin("query", "script", "script-info", "--address", script_address)
    assert script_info["script"]["code"] == updated_code, "Script code not updated"
    assert script_info["script"]["version"] == "2", "Script version should be incremented to 2"

