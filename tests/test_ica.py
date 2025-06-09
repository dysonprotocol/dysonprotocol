import json
import os
import pytest
from tests.utils import poll_until_condition


def test_ica_complete_e2e_workflow(ibc_setup, generate_account, faucet):
    """
    Complete end-to-end test for ICA functionality covering the entire workflow:
    1. Deploy ica_e2e.py script
    2. Register ICA account
    3. Wait for ICA address establishment
    4. Fund the ICA account via IBC transfer
    5. Query balance of ICA account
    6. Wait for balance query callback
    7. Withdraw funds back to controller
    8. Wait for withdrawal callback
    9. Verify all operations and callbacks completed successfully
    """
    dysond_bin = ibc_setup[0]
    [alice_name, alice_address] = generate_account('alice', faucet_amount=20000)
    
    print(f"🔧 Starting ICA E2E test with alice: {alice_name} ({alice_address})")
    
    # 1. Deploy the ICA e2e script
    ica_script_path = "examples/ica_e2e.py"
    abs_ica_script_path = os.path.abspath(ica_script_path)
    print(f"📝 Deploying script from: {abs_ica_script_path}")
    
    update_result = dysond_bin("tx", "script", "update", "--code-path", abs_ica_script_path, "--from", alice_name, "--gas", "1000000")
    print(f"📝 Script deployment result: {update_result}")
    assert update_result.get("code", 1) == 0, f"Failed to deploy ICA e2e script: {update_result}"
    
    # 2. Register ICA account
    print(f"🔗 Registering ICA account...")
    register_result = dysond_bin("tx", "script", "exec", "--script-address", alice_address, "--function-name", "register", "--args", "[]", "--from", alice_name, "--gas", "1000000")
    print(f"🔗 ICA registration result: {register_result}")
    assert register_result.get("code", 1) == 0, f"Failed to register ICA account: {register_result}"
    
    # 3. Wait for ICA address to be established via get_ica_address
    print(f"⏳ Waiting for ICA address establishment...")
    def _ica_address_established():
        """Check if ICA address is available using get_ica_address function"""
        result = dysond_bin("tx", "script", "exec", "--script-address", alice_address, "--function-name", "get_ica_address", "--args", "[]", "--from", alice_name, "--gas", "500000")
        print(f"🔍 get_ica_address check result: {result}")
        assert result.get("code", 1) == 0, f"get_ica_address execution failed: {result}"
        
        for event in result.get("events", []):
            if event.get("type") == "dysonprotocol.script.v1.EventExecScript":
                for attr in event.get("attributes", []):
                    if attr.get("key") == "response":
                        response_json = attr.get("value")
                        response_data = json.loads(response_json)
                        result_data = json.loads(response_data.get("result", "{}"))
                        ica_result = result_data.get("result", {})
                        print(f"🔍 ICA address check response: {ica_result}")
                        
                        if isinstance(ica_result, dict):
                            if ica_result.get("status") == "success":
                                registered_address = ica_result.get("registered_address", "")
                                print(f"✅ Found ICA address: {registered_address}")
                                return registered_address and registered_address.startswith("dys1")
                            else:
                                print(f"❌ ICA address not ready: {ica_result}")
        return False
    
    poll_until_condition(_ica_address_established, timeout=60, poll_interval=3, error_message="ICA address not established after registration")
    
    # 4. Get the established ICA address
    print(f"📍 Getting final ICA address...")
    ica_address_result = dysond_bin("tx", "script", "exec", "--script-address", alice_address, "--function-name", "get_ica_address", "--args", "[]", "--from", alice_name, "--gas", "500000")
    print(f"📍 Final ICA address result: {ica_address_result}")
    assert ica_address_result.get("code", 1) == 0, "Failed to get ICA address"
    
    ica_address = None
    for event in ica_address_result.get("events", []):
        if event.get("type") == "dysonprotocol.script.v1.EventExecScript":
            for attr in event.get("attributes", []):
                if attr.get("key") == "response":
                    response_json = attr.get("value")
                    response_data = json.loads(response_json)
                    result_data = json.loads(response_data.get("result", "{}"))
                    ica_result = result_data.get("result", {})
                    
                    if isinstance(ica_result, dict) and ica_result.get("status") == "success":
                        ica_address = ica_result.get("registered_address", "")
                    break
        if ica_address:
            break
    
    assert ica_address, "ICA address should be returned"
    assert ica_address.startswith("dys1"), "ICA address should be valid bech32 format"
    print(f"✅ ICA Address established: {ica_address}")
    
    # 5. Fund the ICA account via IBC transfer
    fund_args = ["dys", "5000"]
    print(f"💰 Funding ICA account with {fund_args}...")
    fund_result = dysond_bin("tx", "script", "exec", "--script-address", alice_address, "--function-name", "fund", "--args", json.dumps(fund_args), "--from", alice_name, "--gas", "1000000")
    print(f"💰 Fund result: {fund_result}")
    assert fund_result.get("code", 1) == 0, f"Failed to fund ICA account: {fund_result}"
    
    # 6. Request balance query
    print(f"📊 Requesting balance query...")
    query_result = dysond_bin("tx", "script", "exec", "--script-address", alice_address, "--function-name", "request_query_balance", "--args", "[]", "--from", alice_name, "--gas", "1000000")
    print(f"📊 Query request result: {query_result}")
    assert query_result.get("code", 1) == 0, f"Failed to request balance query: {query_result}"
    
    # Extract sequence number from query result
    query_sequence = None
    for event in query_result.get("events", []):
        if event.get("type") == "dysonprotocol.script.v1.EventExecScript":
            for attr in event.get("attributes", []):
                if attr.get("key") == "response":
                    response_json = attr.get("value")
                    response_data = json.loads(response_json)
                    result_data = json.loads(response_data.get("result", "{}"))
                    query_response = result_data.get("result", {})
                    print(f"📊 Query response data: {query_response}")
                    
                    if isinstance(query_response, dict) and query_response.get("status") == "success":
                        query_sequence = query_response.get("sequence")
                        print(f"📊 Extracted sequence: {query_sequence}")
                    break
        if query_sequence is not None:
            break
    
    assert query_sequence is not None, "Balance query sequence should be returned"
    print(f"✅ Balance query sent with sequence: {query_sequence}")
    
    # 7. Wait for balance query callback to be stored
    print(f"⏳ Waiting for balance query callback (sequence: {query_sequence})...")
    def _balance_callback_received():
        """Check if balance query callback has been received"""
        callback_args = ["balance_query", query_sequence]
        print(f"🔍 Checking for callback with args: {callback_args}")
        result = dysond_bin("tx", "script", "exec", "--script-address", alice_address, "--function-name", "get_callback", "--args", json.dumps(callback_args), "--from", alice_name, "--gas", "500000")
        print(f"🔍 Callback check result: {result}")
        assert result.get("code", 1) == 0, f"get_callback execution failed: {result}"
        
        for event in result.get("events", []):
            if event.get("type") == "dysonprotocol.script.v1.EventExecScript":
                for attr in event.get("attributes", []):
                    if attr.get("key") == "response":
                        response_json = attr.get("value")
                        response_data = json.loads(response_json)
                        result_data = json.loads(response_data.get("result", "{}"))
                        callback_result = result_data.get("result", {})
                        print(f"🔍 Callback check response: {callback_result}")
                        
                        if isinstance(callback_result, dict):
                            status = callback_result.get("status")
                            print(f"🔍 Callback status: {status}")
                            if status == "success":
                                print(f"✅ Callback found!")
                                return True
                            elif status == "not_found":
                                print(f"❌ Callback not found yet")
                                return False
                            else:
                                print(f"❓ Unexpected callback status: {status}")
                                return False
        print(f"❌ No callback response found in events")
        return False
    
    # Let's also check what callbacks exist at all
    print(f"🔍 Checking what callbacks exist...")
    list_callbacks_result = dysond_bin("tx", "script", "exec", "--script-address", alice_address, "--function-name", "get_callback", "--args", '["balance_query"]', "--from", alice_name, "--gas", "500000")
    print(f"🔍 List callbacks result: {list_callbacks_result}")
    
    poll_until_condition(_balance_callback_received, timeout=60, poll_interval=3, error_message="Balance query callback not received")
    
    # 8. Verify balance query callback data
    callback_args = ["balance_query", query_sequence]
    print(f"✅ Getting callback data for verification...")
    callback_result = dysond_bin("tx", "script", "exec", "--script-address", alice_address, "--function-name", "get_callback", "--args", json.dumps(callback_args), "--from", alice_name, "--gas", "500000")
    print(f"✅ Callback verification result: {callback_result}")
    assert callback_result.get("code", 1) == 0, "Failed to get balance callback"
    
    # 9. Withdraw funds back to controller
    withdraw_args = ["ibc/3B2294AF63D402DF9B10DA43CEC03677D9041297A1031AB1AFC789C492280D79", "1000"]
    print(f"💸 Withdrawing funds with args: {withdraw_args}")
    withdraw_result = dysond_bin("tx", "script", "exec", "--script-address", alice_address, "--function-name", "withdraw", "--args", json.dumps(withdraw_args), "--from", alice_name, "--gas", "1000000")
    print(f"💸 Withdraw result: {withdraw_result}")
    assert withdraw_result.get("code", 1) == 0, f"Failed to withdraw from ICA account: {withdraw_result}"
    
    # Extract withdrawal sequence number
    withdrawal_sequence = None
    for event in withdraw_result.get("events", []):
        if event.get("type") == "dysonprotocol.script.v1.EventExecScript":
            for attr in event.get("attributes", []):
                if attr.get("key") == "response":
                    response_json = attr.get("value")
                    response_data = json.loads(response_json)
                    result_data = json.loads(response_data.get("result", "{}"))
                    withdraw_response = result_data.get("result", {})
                    print(f"💸 Withdraw response data: {withdraw_response}")
                    
                    if isinstance(withdraw_response, dict) and withdraw_response.get("status") == "success":
                        # For withdrawal, sequence comes from tx_result or we can derive from block
                        # Let's get latest callback for withdrawal topic
                        break
    
    # 10. Wait for withdrawal callback
    print(f"⏳ Waiting for withdrawal callback...")
    def _withdrawal_callback_received():
        """Check if withdrawal callback has been received"""
        # Get latest callback for withdrawal topic
        callback_args = ["withdrawal"]
        print(f"🔍 Checking for withdrawal callback...")
        result = dysond_bin("tx", "script", "exec", "--script-address", alice_address, "--function-name", "get_callback", "--args", json.dumps(callback_args), "--from", alice_name, "--gas", "1500000")
        print(f"🔍 Withdrawal callback check result: {result}")
        assert result.get("code", 1) == 0, f"get_callback execution failed: {result}"
        
        for event in result.get("events", []):
            if event.get("type") == "dysonprotocol.script.v1.EventExecScript":
                for attr in event.get("attributes", []):
                    if attr.get("key") == "response":
                        response_json = attr.get("value")
                        response_data = json.loads(response_json)
                        result_data = json.loads(response_data.get("result", "{}"))
                        callback_result = result_data.get("result", {})
                        print(f"🔍 Withdrawal callback response: {callback_result}")
                        
                        if isinstance(callback_result, dict):
                            status = callback_result.get("status")
                            print(f"🔍 Withdrawal callback status: {status}")
                            return status == "success"
        return False
    
    poll_until_condition(_withdrawal_callback_received, timeout=60, poll_interval=3, error_message="Withdrawal callback not received")
    
    # 11. Verify withdrawal callback data
    withdrawal_callback_args = ["withdrawal"]
    print(f"✅ Getting withdrawal callback data for verification...")
    withdrawal_callback_result = dysond_bin("tx", "script", "exec", "--script-address", alice_address, "--function-name", "get_callback", "--args", json.dumps(withdrawal_callback_args), "--from", alice_name, "--gas", "1500000")
    print(f"✅ Withdrawal callback verification result: {withdrawal_callback_result}")
    assert withdrawal_callback_result.get("code", 1) == 0, "Failed to get withdrawal callback"
    
    print(f"🎉 Complete ICA E2E workflow successful!")
    print(f"   - ICA Address: {ica_address}")
    print(f"   - Registration, funding, balance query, and withdrawal completed")
    print(f"   - All callbacks received and stored properly")
    print(f"   - Query sequence: {query_sequence}")
    print(f"   - Script deployment, IBC operations, and callback handling working")