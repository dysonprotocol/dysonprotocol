import json
import pytest
import tempfile
import os
from tests.utils import poll_until_condition
import time


def test_update_and_query_script(chainnet, generate_account):
    dysond_bin = chainnet[0]
    [alice_name, alice_address] = generate_account('alice')
    script_code = """
def wsgi(environ, start_response):
    status = '200 OK'
    headers = [('Content-type', 'text/html')]
    start_response(status, headers)
    return [b'<html><body><h1>Hello from Script Test!</h1></body></html>']
"""
    update_result = dysond_bin("tx", "script", "update", "--code", script_code, "--from", alice_name, "--keyring-backend", "test", "--yes")
    print(f"Update script result: {update_result}")
    assert update_result.get("code", 1) == 0, "Failed to update script"
    script_info = dysond_bin("query", "script", "script-info", "--address", alice_address)
    print(f"Script info: {script_info}")
    assert script_info.get("script", {}).get("address") == alice_address, "Script address doesn't match"
    assert script_info.get("script", {}).get("code") == script_code, "Script code doesn't match"


def test_encode_json(chainnet):
    dysond_bin = chainnet[0]
    test_data = {
        "@type": "/cosmos.bank.v1beta1.MsgSend",
        "from_address": "dys1example",
        "to_address": "dys1example",
        "amount": [{"denom": "dys", "amount": "100"}]
    }
    test_json = json.dumps(test_data)
    encode_result = dysond_bin("query", "script", "encode-json", "--json", test_json)
    print(f"Encode JSON result: {encode_result}")
    assert "bytes" in encode_result, "Encode result missing 'bytes' field"


def test_decode_bytes(chainnet):
    dysond_bin = chainnet[0]
    test_data = {
        "@type": "/cosmos.bank.v1beta1.MsgSend",
        "from_address": "dys1example",
        "to_address": "dys1example",
        "amount": [{"denom": "dys", "amount": "100"}]
    }
    test_json = json.dumps(test_data)
    encode_result = dysond_bin("query", "script", "encode-json", "--json", test_json)
    encoded_bytes = encode_result.get("bytes")
    assert encoded_bytes, "Failed to get encoded bytes"
    decode_result = dysond_bin("query", "script", "decode-bytes", "--bytes", encoded_bytes, "--type-url", "/cosmos.bank.v1beta1.MsgSend")
    print(f"Decode bytes result: {decode_result}")
    assert "json" in decode_result, "Decode result missing 'json' field"
    assert "dys1example" in decode_result["json"], "Decoded data doesn't contain original content"


def test_exec_script(chainnet, generate_account):
    dysond_bin = chainnet[0]
    [alice_name, alice_address] = generate_account('alice')
    function_code = """
def add(a, b):
    return a + b
"""
    update_result = dysond_bin("tx", "script", "update", "--code", function_code, "--from", alice_name, "--keyring-backend", "test", "--yes")
    assert update_result.get("code", 1) == 0, "Failed to update script"
    args = [5, 7]
    args_json = json.dumps(args)
    exec_result = dysond_bin(
        "tx", "script", "exec",
        "--script-address", alice_address,
        "--function-name", "add",
        "--args", args_json,
        "--from", alice_name,
        #"--gas", "auto",
        #"--gas-adjustment", "5"
    )
    assert exec_result.get("code", 1) == 0, "Failed to execute script"
    result_value = None
    for event in exec_result.get("events", []):
        if event.get("type") == "dysonprotocol.script.v1.EventExecScript":
            for attr in event.get("attributes", []):
                if attr.get("key") == "response":
                    response_json = attr.get("value")
                    response_data = json.loads(response_json)
                    result_data = json.loads(response_data.get("result", "{}"))
                    result_value = result_data.get("result")
                    break
            if result_value is not None:
                break
    assert result_value == 12, f"Expected result 12, got '{result_value}'"


def test_verify_arbitrary_data_signature(chainnet, generate_account, faucet):
    """Test signing and verifying arbitrary data using MsgArbitraryData"""
    dysond_bin = chainnet[0]
    [alice_name, alice_address] = generate_account('alice')
    faucet(alice_address, denom="dys", amount="10")

    import tempfile
    import os
    # Create temporary files for the transaction
    with tempfile.NamedTemporaryFile(suffix='.json', delete=False) as tx_file, \
         tempfile.NamedTemporaryFile(suffix='.json', delete=False) as signed_tx_file:
        tx_json_path = tx_file.name
        signed_tx_json_path = signed_tx_file.name
        try:
            # Construct MsgArbitraryData transaction
            tx_data = {
                "body": {
                    "messages": [
                        {
                            "@type": "/dysonprotocol.script.v1.MsgArbitraryData",
                            "signer": alice_address,
                            "data": "this is test data",
                            "app_domain": "fooApp/123"
                        }
                    ],
                    "memo": ""
                },
                "auth_info": {
                    "signer_infos": [],
                    "fee": {
                        "amount": [],
                        "gas_limit": "0"
                    }
                },
                "signatures": []
            }
            # Write the transaction data to the file
            with open(tx_json_path, 'w') as f:
                json.dump(tx_data, f)
            # Sign the transaction (offline, ADR-036 parameters)
            sign_result = dysond_bin(
                "tx", "sign", tx_json_path,
                "--from", alice_name,
                "--chain-id", "",
                "--account-number", "0",
                "--sequence", "0",
                "--offline",
                "--output-document", signed_tx_json_path,
                "--keyring-backend", "test"
            )
            # Read the signed transaction
            with open(signed_tx_json_path, 'r') as f:
                signed_tx_json = f.read()
            # Verify the signed transaction
            verify_result = dysond_bin(
                "query", "script", "verify-tx",
                "--tx-json", signed_tx_json,
                "-o", "json"
            )
            print(f"Verify transaction result: {verify_result}")
            assert isinstance(verify_result, dict), "Verification failed"
        finally:
            for path in [tx_json_path, signed_tx_json_path]:
                try:
                    os.remove(path)
                except OSError:
                    pass


def test_script_governance_param_update_and_storage_history(chainnet, generate_account, faucet):
    """
    Test script parameter updates via governance proposal and storage history functionality.
    
    This test:
    1. Sets new script params with a gov proposal
    2. Creates a function to store storage data {"block_height": <height>} in storage at index "test_history" 
    3. Calls the storage function 3 separate times
    4. Creates a function query(heights: List) that passes heights and returns "test_history" at each height
    5. Verifies that the return data is correct
    """
    dysond_bin = chainnet[0]
    [alice_name, alice_address] = generate_account('alice')
    faucet(alice_address, denom="dys", amount="1000000")
    
    # Delegate tokens to get voting power
    validators_result = dysond_bin("query", "staking", "validators")
    if isinstance(validators_result, str):
        assert False, f"Failed to query validators: {validators_result}"
    validator_operator = validators_result["validators"][0]["operator_address"]
    
    delegate_result = dysond_bin("tx", "staking", "delegate", validator_operator, "20000dys", "--from", alice_name)
    if isinstance(delegate_result, str):
        assert False, f"Failed to delegate: {delegate_result}"
    assert delegate_result["code"] == 0, f"Failed to delegate: {delegate_result}"
    print("Delegated tokens to get voting power")
    
    # Step 1: Create and submit governance proposal to update script params
    print("Creating governance proposal to update script parameters...")
    
    # Get the governance module address
    gov_module_result = dysond_bin("query", "auth", "module-account", "gov")
    if isinstance(gov_module_result, str):
        assert False, f"Failed to query gov module account: {gov_module_result}"
    gov_address = gov_module_result.get("account", {}).get("value", {}).get("address", "")
    assert gov_address, "Could not get governance module address"
    print(f"Using governance module address: {gov_address}")
    
    # Create proposal JSON file
    proposal_data = {
        "messages": [
            {
                "@type": "/dysonprotocol.script.v1.MsgUpdateParams",
                "authority": gov_address,
                "params": {
                    "maxRelativeHistoricalBlocks": "1000",  # Update from default to 1000
                    "absoluteHistoricalBlockCutoff": "1"    # Keep default cutoff
                }
            }
        ],
        "metadata": "ipfs://CID",
        "deposit": "100000dys",
        "title": "Update Script Module Parameters",
        "summary": "Update maxRelativeHistoricalBlocks parameter to 1000"
    }
    
    # Submit the proposal using temporary file
    with tempfile.NamedTemporaryFile(mode='w', suffix='.json', delete=True) as proposal_file:
        json.dump(proposal_data, proposal_file, indent=2)
        proposal_file.flush()
        
        result = dysond_bin("tx", "gov", "submit-proposal", proposal_file.name, "--from", alice_name)
    if isinstance(result, str):
        assert False, f"Failed to submit proposal: {result}"
    assert result["code"] == 0, f"Failed to submit proposal: {result}"
    
    # Get proposal ID from the result
    proposal_id = None
    for event in result.get("events", []):
        if event.get("type") == "submit_proposal":
            for attr in event.get("attributes", []):
                if attr.get("key") == "proposal_id":
                    proposal_id = attr.get("value")
                    break
    
    assert proposal_id is not None, "Could not find proposal ID in output"
    print(f"Submitted proposal ID: {proposal_id}")
    
    # Vote on the proposal
    vote_result = dysond_bin("tx", "gov", "vote", proposal_id, "yes", "--from", alice_name)
    if isinstance(vote_result, str):
        assert False, f"Failed to vote on proposal: {vote_result}"
    assert vote_result["code"] == 0, f"Failed to vote on proposal: {vote_result}"
    
    # Wait for proposal to pass
    def check_proposal_status():
        result = dysond_bin("query", "gov", "proposal", proposal_id)
        if isinstance(result, str):
            return False
        return result.get("proposal", {}).get("status") == "PROPOSAL_STATUS_PASSED"
    
    poll_until_condition(check_proposal_status, timeout=60, poll_interval=2)
    print("Governance proposal passed!")
    
    # Step 2: Create a script that stores block height data in storage
    print("Creating script to store block height data...")
    
    storage_script_code = '''
def store_block_height():
    """Store current block height in storage at index 'test_history'"""
    import json
    from dys import get_block_info, get_script_address, _msg
    
    block_info = get_block_info()
    current_height = block_info["height"]
    data = {"block_height": current_height}
    
    # Store data using storage module
    result = _msg({
        "@type": "/dysonprotocol.storage.v1.MsgStorageSet",
        "owner": get_script_address(),
        "index": "test_history",
        "data": json.dumps(data)
    })
    
    return f"Stored block height {current_height}"

def query_heights(heights):
    """Query storage data for given heights"""
    import json
    from dys import get_script_address, _query
    
    results = []
    for height in heights:
        try:
            # Query storage at specific height
            result = _query({
                "@type": "/dysonprotocol.storage.v1.QueryStorageGetRequest",
                "owner": get_script_address(),
                "index": "test_history"
            }, query_height=height)
            
            results.append(result)

        except Exception as e:
            results.append({"error": str(e)})
    
    return results
'''
    
    # Create the storage script
    result = dysond_bin("tx", "script", "create-new-script", "storage_test_script", "--code", storage_script_code, "--from", alice_name)
    if isinstance(result, str):
        assert False, f"Failed to create storage script: {result}"
    assert result["code"] == 0, f"Failed to create storage script: {result}"
    
    # Get script address from the create transaction result
    script_address = get_script_address_from_create_result(result)
    print(f"Created storage script at address: {script_address}")
    
    # Step 3: Call the storage function 3 separate times to store data at different heights
    print("Storing block height data 3 times...")
    
    stored_heights = []
    for i in range(3):
        print(f"Storing data iteration {i+1}...")
        
        # Execute the store_block_height function
        result = dysond_bin("tx", "script", "exec", "--script-address", script_address, "--function-name", "store_block_height", "--args", "[]", "--from", alice_name)
        if isinstance(result, str):
            assert False, f"Failed to execute store_block_height iteration {i+1}: {result}"
        assert result["code"] == 0, f"Failed to execute store_block_height iteration {i+1}: {result}"
        
        # Get current block height to track what was stored
        block_result = dysond_bin("query", "block")
        if isinstance(block_result, str):
            # Try to parse the JSON from the string response
            # The response might have a prefix, so extract just the JSON part
            try:
                # Look for the JSON part (starts with '{')
                json_start = block_result.find('{')
                if json_start != -1:
                    json_part = block_result[json_start:].strip()
                    block_result = json.loads(json_part)
                else:
                    assert False, f"No JSON found in block query response: {block_result}"
            except json.JSONDecodeError:
                assert False, f"Failed to parse block query response: {block_result}"
                    # Debug: print the structure to understand the response format
            print(f"Block result structure: {list(block_result.keys())}")
            if "header" in block_result:
                current_height = int(block_result["header"]["height"])
            elif "block" in block_result:
                current_height = int(block_result["block"]["header"]["height"])
            else:
                assert False, f"Unexpected block result structure: {block_result}"
        stored_heights.append(current_height)
        
        # Wait a bit to ensure different block heights
        time.sleep(2)
    
        print(f"Stored data at heights: {stored_heights}")
        
        # Step 4: First check if data was stored by querying current state
        print("Checking if data was stored in current state...")
        
        # Query current storage state directly
        current_storage_result = dysond_bin("query", "storage", "get", script_address, "--index", "test_history")
        if isinstance(current_storage_result, str):
            print(f"Current storage query returned string: {current_storage_result}")
        else:
            print(f"Current storage state: {current_storage_result}")
        
        # Step 5: Query the stored data using the query function
        print("Querying stored data...")

        # Execute the query_heights function with the stored heights
        heights_json = json.dumps(stored_heights)
        result = dysond_bin("tx", "script", "exec", "--script-address", script_address, "--function-name", "query_heights", "--args", f'[{heights_json}]', "--from", alice_name)
    if isinstance(result, str):
        assert False, f"Failed to execute query_heights: {result}"
    assert result["code"] == 0, f"Failed to execute query_heights: {result}"
    
    # Extract the script execution result from transaction events
    script_result = None
    for event in result.get("events", []):
        if event.get("type") == "dysonprotocol.script.v1.EventExecScript":
            for attr in event.get("attributes", []):
                if attr.get("key") == "response":
                    response_json = attr.get("value")
                    response_data = json.loads(response_json)
                    result_data = json.loads(response_data.get("result", "{}"))
                    script_result = result_data.get("result")
                    break
    
    assert script_result is not None, "Could not find script execution result in transaction events"
    
    # Step 5: Verify that the return data is correct
    print("Verifying query results...")
    
    # Parse the result
    try:
        query_results = script_result
        print(f"Query results: {query_results}")
        
        # Verify each result
        assert len(query_results) == 3, f"Expected 3 results, got {len(query_results)}"
        
        for i, (stored_height, result_data) in enumerate(zip(stored_heights, query_results)):
            assert result_data is not None, f"Result {i+1} should not be None"
            assert "@type" in result_data, f"Result {i+1} should contain '@type'"
            assert result_data["@type"] == "/dysonprotocol.storage.v1.QueryStorageGetResponse", f"Result {i+1} should be a QueryStorageGetResponse"
            assert "entry" in result_data, f"Result {i+1} should contain 'entry'"
            assert "data" in result_data["entry"], f"Result {i+1} entry should contain 'data'"
            
            # Parse the JSON data from the entry
            data_str = result_data["entry"]["data"]
            data = json.loads(data_str)
            assert "block_height" in data, f"Result {i+1} data should contain 'block_height'"
            
            # The stored height should match what we expect (within a reasonable range due to timing)
            result_height = data["block_height"]
            assert abs(result_height - stored_height) <= 2, \
                f"Result {i+1}: expected height ~{stored_height}, got {result_height}"
            
            print(f"✓ Verified result {i+1}: stored at height {result_height}")
        
        print("✓ All storage history data verified successfully!")
        
    except (json.JSONDecodeError, TypeError) as e:
        assert False, f"Failed to parse script result: {e}"
    
    print("Test completed successfully!")


def get_script_address_from_create_result(create_result):
    """Helper function to extract script address from create script transaction result"""
    for event in create_result.get("events", []):
        if event.get("type") == "dysonprotocol.script.v1.EventCreateNewScript":
            for attr in event.get("attributes", []):
                if attr.get("key") == "script_address":
                    # The value might be JSON-encoded, so try to parse it
                    script_address = attr.get("value")
                    try:
                        import json
                        script_address = json.loads(script_address)
                    except (json.JSONDecodeError, TypeError):
                        # If it's not JSON, use the value as-is
                        pass
                    return script_address
    
    assert False, f"Script address not found in create transaction events: {create_result}"

