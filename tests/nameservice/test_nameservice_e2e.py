"""
End-to-end test for the Nameservice module based on the README.

This test demonstrates a complete workflow of the Nameservice module:
1. Name Registration (commit-reveal)
2. Destination Setting
3. Name Valuation Update
4. Creating NFT Classes
5. Minting NFTs
6. NFT Metadata Management
7. Custom Coin Operations
8. Bidding System (place-bid, accept-bid, reject-bid, claim-bid)
"""
import pytest
import json
import os
import random
import string
import secrets
import time
import tempfile
from tests.utils import poll_until_condition


def test_nameservice_e2e(chainnet, generate_account, faucet):
    """
    Complete end-to-end test of the Nameservice module.
    Tests all main functionalities described in the Nameservice README.
    """
    dysond_bin = chainnet[0]
    [alice_name, alice_address] = generate_account('alice')
    [bob_name, bob_address] = generate_account('bob')
    [charlie_name, charlie_address] = generate_account('charlie')
    faucet(alice_address, denom="dys", amount="25000")
    faucet(bob_address, denom="dys", amount="1000")
    faucet(charlie_address, denom="dys", amount="1000")
    print(f"Alice address: {alice_address}")
    print(f"Bob address: {bob_address}")
    print(f"Charlie address: {charlie_address}")
    
    # Generate random test values
    timestamp = int(time.time())
    random_suffix = ''.join(random.choices(string.ascii_lowercase, k=8))
    test_id = f"{random_suffix}-{timestamp}"
    
    # Get initial balances to track changes
    initial_alice_balances = dysond_bin("query", "bank", "balances", alice_address)
    initial_bob_balances = dysond_bin("query", "bank", "balances", bob_address)
    
    # Convert balances to dict for easier access
    alice_balances_dict = {b["denom"]: int(b["amount"]) for b in initial_alice_balances.get("balances", [])}
    alice_dys_balance = alice_balances_dict.get("dys", 0)
    print(f"Initial Alice DYS balance: {alice_dys_balance}")
    
    # Step 1: Check module parameters
    params = dysond_bin("query", "nameservice", "params")
    assert "params" in params, "Parameters missing from query result"
    assert "bid_timeout" in params["params"], "Parameters missing bid_timeout"
    assert "allowed_denoms" in params["params"], "Parameters missing allowed_denoms"
    assert "dys" in params["params"]["allowed_denoms"], "DYS not in allowed denoms"
    print(f"Module parameters: {json.dumps(params, indent=2)}")
    
    bid_timeout = params["params"]["bid_timeout"]
    print(f"Bid timeout: {bid_timeout}")
    
    # Check if the timeout is short enough for our test
    wait_for_timeout = False
    if bid_timeout == "5s":
        wait_for_timeout = True
        print("Bid timeout is set to 5 seconds, will wait for timeout in the claim-bid test")
    
    # Step 2: Name Registration
    # Generate a random name and salt
    name = f"test-{test_id}.dys"
    salt = secrets.token_hex(8)
    print(f"Generated salt: {salt} for name: {name}")
    
    # Compute the hash for commitment
    hash_result = dysond_bin("query", "nameservice", "compute-hash", "--name", name, "--salt", salt, "--committer", alice_address)
    commitment_hash = hash_result.get("hex_hash")
    assert commitment_hash, "Could not compute commitment hash"
    print(f"Commitment hash: {commitment_hash}")
    
    # Use random valuation between 10 and 100 DYS
    initial_valuation = random.randint(10, 100)
    
    # Commit to registering the name
    commit_result = dysond_bin("tx", "nameservice", "commit", "--commitment", commitment_hash, "--valuation", f"{initial_valuation}dys", "--from", alice_name)
    assert commit_result["code"] == 0, "Commit transaction failed"
    print(f"Name commitment successful with valuation {initial_valuation}dys")
    
    # Reveal the name to complete registration
    reveal_result = dysond_bin("tx", "nameservice", "reveal", "--name", name, "--salt", salt, "--from", alice_name)
    assert reveal_result["code"] == 0, "Reveal transaction failed"
    print("Name registration successful")
    
    # Verify the registration
    nft_info = dysond_bin("query", "nft", "nft", "nameservice.dys", name)
    assert "nft" in nft_info, "NFT not found after registration"
    assert nft_info["nft"]["id"] == name, "NFT ID doesn't match registered name"
    assert nft_info["nft"]["class_id"] == "nameservice.dys", "NFT class ID incorrect"
    
    # Extract NFT data to verify valuation
    nft_data = nft_info["nft"].get("data", {}).get("value", {})
    assert nft_data.get("valuation", {}).get("amount") == str(initial_valuation), "Incorrect initial valuation"
    print(f"Initial NFT valuation: {nft_data.get('valuation', {}).get('amount')} dys")
    
    # Step 3: Destination Setting
    destination_result = dysond_bin("tx", "nameservice", "set-destination", "--name", name, "--destination", alice_address, "--from", alice_name)
    assert destination_result["code"] == 0, "Set destination transaction failed"
    print("Destination set successfully")
    
    # Verify the destination was set
    updated_nft_info = dysond_bin("query", "nft", "nft", "nameservice.dys", name)
    assert updated_nft_info["nft"]["uri"] == alice_address, "Destination not set correctly"
    print(f"Verified destination is set to: {updated_nft_info['nft']['uri']}")
    
    # Step 4: Name Valuation Update
    # Set new valuation higher than the current one
    new_valuation = initial_valuation + random.randint(50, 150)
    valuation_result = dysond_bin("tx", "nameservice", "set-valuation", "--class-id", "nameservice.dys", "--nft-id", name, "--valuation", f"{new_valuation}dys", "--from", alice_name)
    assert valuation_result["code"] == 0, "Set valuation transaction failed"
    print(f"Valuation updated successfully from {initial_valuation} to {new_valuation}")
    
    # Verify the updated valuation
    revalued_nft_info = dysond_bin("query", "nft", "nft", "nameservice.dys", name)
    updated_valuation = revalued_nft_info["nft"].get("data", {}).get("value", {}).get("valuation", {}).get("amount")
    assert updated_valuation == str(new_valuation), "Valuation not updated correctly"
    print(f"Updated NFT valuation verified: {updated_valuation} dys")
    
    # Step 5: Creating NFT Classes
    # Create a main NFT class with randomized names
    collection_name = f"Collection-{test_id}"
    collection_symbol = f"COL{random.randint(100, 999)}"
    
    main_class_result = dysond_bin("tx", "nameservice", "save-class", "--class-id", name, "--name", collection_name, "--symbol", collection_symbol, "--description", f"Test Collection {test_id}", "--uri", "https://example.com", "--from", alice_name)
    assert main_class_result["code"] == 0, "Create main NFT class transaction failed"
    print(f"Main NFT class {name} created successfully")
    
    # Create a sub-collection with randomized names
    sub_class_id = f"{name}/sub{random.randint(100, 999)}"
    sub_collection_name = f"SubCollection-{test_id}"
    sub_collection_symbol = f"SUB{random.randint(100, 999)}"
    
    sub_class_result = dysond_bin("tx", "nameservice", "save-class", "--class-id", sub_class_id, "--name", sub_collection_name, "--symbol", sub_collection_symbol, "--description", f"Test Sub-Collection {test_id}", "--uri", "https://example.com/sub", "--from", alice_name)
    assert sub_class_result["code"] == 0, "Create sub NFT class transaction failed"
    print(f"Sub NFT class {sub_class_id} created successfully")
    
    # View all NFT classes
    classes = dysond_bin("query", "nft", "classes")
    assert "classes" in classes, "No NFT classes found"
    
    # Verify our classes were created
    class_ids = [c["id"] for c in classes["classes"]]
    assert name in class_ids, f"Main class {name} not found in class list"
    assert sub_class_id in class_ids, f"Sub class {sub_class_id} not found in class list"
    print(f"Verified {len(classes['classes'])} NFT classes exist, including our new ones")
    
    # Step 6: Minting NFTs
    # Generate random NFT IDs
    nft_id = f"nft-{random.randint(1000, 9999)}"
    mint_result = dysond_bin("tx", "nameservice", "mint-nft", "--class-id", name, "--nft-id", nft_id, "--uri", f"https://example.com/{nft_id}", "--from", alice_name)
    assert mint_result["code"] == 0, "Mint NFT transaction failed"
    print(f"NFT {nft_id} minted successfully in class {name}")
    
    # Mint an NFT in the sub-collection
    sub_nft_id = f"subnft-{random.randint(1000, 9999)}"
    sub_mint_result = dysond_bin("tx", "nameservice", "mint-nft", "--class-id", sub_class_id, "--nft-id", sub_nft_id, "--uri", f"https://example.com/{sub_nft_id}", "--from", alice_name)
    assert sub_mint_result["code"] == 0, "Mint sub-NFT transaction failed"
    print(f"NFT {sub_nft_id} minted successfully in class {sub_class_id}")
    
    # View NFTs in the main collection
    nfts = dysond_bin("query", "nft", "nfts", name)
    assert "nfts" in nfts, "No NFTs found in the main collection"
    assert len(nfts["nfts"]) > 0, "Main collection is empty"
    assert any(n["id"] == nft_id for n in nfts["nfts"]), f"NFT {nft_id} not found in main collection"
    print(f"Verified {len(nfts['nfts'])} NFTs exist in main collection")
    
    # Step 7: NFT Metadata Management
    # Generate random metadata
    colors = ["Red", "Blue", "Green", "Yellow", "Purple", "Orange"]
    random_color = random.choice(colors)
    random_rarity = random.randint(1, 100)
    
    metadata = json.dumps({
        "attributes": [
            {"trait_type": "Color", "value": random_color},
            {"trait_type": "Rarity", "value": random_rarity}
        ]
    })
    
    metadata_result = dysond_bin("tx", "nameservice", "set-nft-metadata", "--class-id", name, "--nft-id", nft_id, "--metadata", f"'{metadata}'", "--from", alice_name)
    assert metadata_result["code"] == 0, "Set NFT metadata transaction failed"
    print(f"Metadata set successfully for NFT {nft_id}")
    
    # Set random extra data for the NFT class
    random_website = f"https://example.com/{test_id}"
    extra_data = json.dumps({
        "website": random_website,
        "created_at": timestamp,
        "creator": "Test Suite"
    })
    
    extra_data_result = dysond_bin("tx", "nameservice", "set-nft-class-extra-data", "--class-id", name, "--extra-data", f"'{extra_data}'", "--from", alice_name)
    assert extra_data_result["code"] == 0, "Set NFT class extra data transaction failed"
    print(f"Extra data set successfully for class {name}")
    
    # View the updated NFT with metadata
    updated_nft = dysond_bin("query", "nft", "nft", name, nft_id)
    assert "nft" in updated_nft, f"NFT {nft_id} not found after metadata update"
    
    # Check if metadata was added correctly
    nft_metadata = updated_nft["nft"].get("data", {}).get("value", {}).get("metadata", "")
    assert random_color in nft_metadata, "Color metadata not set correctly"
    assert str(random_rarity) in nft_metadata, "Rarity metadata not set correctly"
    print(f"Verified metadata was updated with color {random_color} and rarity {random_rarity}")
    
    # Step 8: Custom Coin Operations
    # Generate random coin amounts
    main_coin_amount = random.randint(500, 2000)
    sub_coin_amount = random.randint(100, 500)
    
    # Mint coins with the name as denomination
    mint_coins_result = dysond_bin("tx", "nameservice", "mint-coins", "--amount", f"{main_coin_amount}{name}", "--from", alice_name)
    assert mint_coins_result["code"] == 0, "Mint coins transaction failed"
    print(f"Minted {main_coin_amount} {name} tokens successfully")
    
    # Mint coins with subdenom
    subdenom = f"{name}/token{random.randint(1, 999)}"
    mint_subdenom_result = dysond_bin("tx", "nameservice", "mint-coins", "--amount", f"{sub_coin_amount}{subdenom}", "--from", alice_name)
    assert mint_subdenom_result["code"] == 0, "Mint subdenom coins transaction failed"
    print(f"Minted {sub_coin_amount} {subdenom} tokens successfully")
    
    # Check balances
    updated_alice_balances = dysond_bin("query", "bank", "balances", alice_address)
    assert "balances" in updated_alice_balances, "No balances found"
    
    # Verify minted coins are in the balance
    updated_alice_balances_dict = {b["denom"]: b["amount"] for b in updated_alice_balances["balances"]}
    assert name in updated_alice_balances_dict, f"Minted {name} coins not found in balance"
    assert int(updated_alice_balances_dict[name]) == main_coin_amount, f"Incorrect balance of {name} coins: Expected {main_coin_amount}, got {updated_alice_balances_dict[name]}"
    
    assert subdenom in updated_alice_balances_dict, f"Minted {subdenom} coins not found in balance"
    assert int(updated_alice_balances_dict[subdenom]) == sub_coin_amount, f"Incorrect balance of {subdenom} coins: Expected {sub_coin_amount}, got {updated_alice_balances_dict[subdenom]}"
    print(f"Verified minted coin balances")
    
    # Calculate a random transfer amount (between 10% and 50% of total)
    transfer_amount = random.randint(int(main_coin_amount * 0.1), int(main_coin_amount * 0.5))
    
    # Send custom coins to Bob
    send_coins_result = dysond_bin("tx", "bank", "send", alice_address, bob_address, f"{transfer_amount}{name}")
    assert send_coins_result["code"] == 0, "Send coins transaction failed"
    print(f"Sent {transfer_amount} {name} tokens to Bob successfully")
    
    # Check Bob's balance
    bob_updated_balances = dysond_bin("query", "bank", "balances", bob_address)
    bob_balances_dict = {b["denom"]: b["amount"] for b in bob_updated_balances["balances"]}
    assert name in bob_balances_dict, f"Sent {name} coins not found in Bob's balance"
    assert int(bob_balances_dict[name]) == transfer_amount, f"Incorrect balance of {name} coins in Bob's account: Expected {transfer_amount}, got {bob_balances_dict[name]}"
    print(f"Verified Bob received {transfer_amount} {name} tokens")
    
    # Check Alice's updated balance
    alice_final_balances = dysond_bin("query", "bank", "balances", alice_address)
    alice_final_balances_dict = {b["denom"]: b["amount"] for b in alice_final_balances["balances"]}
    expected_alice_balance = main_coin_amount - transfer_amount
    assert int(alice_final_balances_dict[name]) == expected_alice_balance, f"Alice's {name} balance not correctly reduced after transfer. Expected {expected_alice_balance}, got {alice_final_balances_dict[name]}"
    print(f"Verified Alice's {name} balance decreased to {expected_alice_balance}")
    
    # Step 9: Bidding System Tests
    # Use a new name for bidding tests to keep the original name clean
    bidding_name = f"bid-{test_id}.dys"
    bidding_salt = secrets.token_hex(8)
    
    # Compute hash for commitment
    bidding_hash_result = dysond_bin("query", "nameservice", "compute-hash", "--name", bidding_name, "--salt", bidding_salt, "--committer", alice_address)
    bidding_commitment_hash = bidding_hash_result.get("hex_hash")
    
    # Set bidding name valuation
    bidding_valuation = random.randint(50, 150)
    
    # Commit to registering the bidding test name
    bidding_commit_result = dysond_bin("tx", "nameservice", "commit", "--commitment", bidding_commitment_hash, "--valuation", f"{bidding_valuation}dys", "--from", alice_name)
    assert bidding_commit_result["code"] == 0, "Bidding name commit transaction failed"
    
    # Reveal the bidding test name
    bidding_reveal_result = dysond_bin("tx", "nameservice", "reveal", "--name", bidding_name, "--salt", bidding_salt, "--from", alice_name)
    assert bidding_reveal_result["code"] == 0, "Bidding name reveal transaction failed"
    print(f"Registered bidding test name: {bidding_name}")
    
    # Verify the NFT exists with correct valuation
    bidding_nft_info = dysond_bin("query", "nft", "nft", "nameservice.dys", bidding_name)
    assert "nft" in bidding_nft_info, "Bidding NFT not found after registration"
    assert bidding_nft_info["nft"]["id"] == bidding_name, "Bidding NFT ID doesn't match registered name"
    
    # Step 9.1: Bob places a bid on the NFT
    bob_bid_amount = bidding_valuation + random.randint(10, 30)
    bob_bid_result = dysond_bin("tx", "nameservice", "place-bid", "--nft-class-id", "nameservice.dys", "--nft-id", bidding_name, "--bid-amount", f"{bob_bid_amount}dys", "--from", bob_name)
    assert bob_bid_result["code"] == 0, "Bob's place bid transaction failed"
    print(f"Bob successfully placed a bid of {bob_bid_amount}dys")
    
    # Verify the bid was recorded
    nft_info_after_bid = dysond_bin("query", "nft", "nft", "nameservice.dys", bidding_name)
    nft_data_after_bid = nft_info_after_bid["nft"].get("data", {}).get("value", {})
    assert nft_data_after_bid.get("current_bid", {}).get("amount") == str(bob_bid_amount), "Bob's bid not recorded correctly"
    assert nft_data_after_bid.get("current_bidder") == bob_address, "Bob's address not recorded as bidder"
    print(f"Verified current bid is now {bob_bid_amount}dys from Bob")
    
    # Step 9.2: Alice accepts Bob's bid
    accept_bid_result = dysond_bin("tx", "nameservice", "accept-bid", "--nft-class-id", "nameservice.dys", "--nft-id", bidding_name, "--from", alice_name)
    assert accept_bid_result["code"] == 0, "Accept bid transaction failed"
    print("Alice successfully accepted Bob's bid")
    
    # Verify the NFT is now owned by Bob
    nft_owner_after_accept = dysond_bin("query", "nft", "owner", "nameservice.dys", bidding_name)
    assert nft_owner_after_accept.get("owner") == bob_address, "NFT not transferred to Bob after bid acceptance"
    print(f"Verified NFT ownership transferred to Bob: {nft_owner_after_accept.get('owner')}")
    
    # Step 9.3: Charlie places a bid on Bob's NFT
    # Make sure Charlie has some funds
    try:
        charlie_balance = dysond_bin("query", "bank", "balances", charlie_address)
        has_dys = False
        for balance in charlie_balance.get("balances", []):
            if balance.get("denom") == "dys" and int(balance.get("amount", "0")) > 200:
                has_dys = True
                break
        
        if not has_dys:
            # Send funds to Charlie from Alice
            fund_result = dysond_bin("tx", "bank", "send", "alice", charlie_address, "200dys")
            assert fund_result["code"] == 0, "Failed to fund Charlie's account"
            print("Funded Charlie's account with 200dys")
    except Exception as e:
        print(f"Error checking/funding Charlie's account: {e}")
    
    charlie_bid_amount = bob_bid_amount + random.randint(20, 50)
    charlie_bid_result = dysond_bin("tx", "nameservice", "place-bid", "--nft-class-id", "nameservice.dys", "--nft-id", bidding_name, "--bid-amount", f"{charlie_bid_amount}dys", "--from", charlie_name)
    assert charlie_bid_result["code"] == 0, "Charlie's place bid transaction failed" 
    print(f"Charlie successfully placed a bid of {charlie_bid_amount}dys")
    
    # Step 9.4: Bob rejects Charlie's bid with a higher valuation
    new_bidding_valuation = charlie_bid_amount + random.randint(10, 30)
    reject_bid_result = dysond_bin("tx", "nameservice", "reject-bid", "--nft-class-id", "nameservice.dys", "--nft-id", bidding_name, "--new-valuation", f"{new_bidding_valuation}dys", "--from", bob_name)
    assert reject_bid_result["code"] == 0, "Reject bid transaction failed"
    print(f"Bob successfully rejected Charlie's bid and set new valuation to {new_bidding_valuation}dys")
    
    # Verify Charlie's bid was rejected and valuation was updated
    nft_info_after_reject = dysond_bin("query", "nft", "nft", "nameservice.dys", bidding_name)
    nft_data_after_reject = nft_info_after_reject["nft"].get("data", {}).get("value", {})
    assert nft_data_after_reject.get("valuation", {}).get("amount") == str(new_bidding_valuation), "New valuation not set correctly"
    assert not nft_data_after_reject.get("current_bidder"), "Bid was not cleared after rejection"
    print("Verified bid was rejected and valuation updated")
    
    # Step 9.5: Test claim-bid functionality
    # Set a short bid timeout via governance proposal for fast testing
    set_bid_timeout_via_gov(dysond_bin, alice_name, "100ms")
    
    # Create a separate name for testing bid timeout
    timeout_name = f"timeout-{test_id}.dys"
    timeout_salt = secrets.token_hex(8)
    
    # Compute hash for commitment
    timeout_hash_result = dysond_bin("query", "nameservice", "compute-hash", "--name", timeout_name, "--salt", timeout_salt, "--committer", alice_address)
    timeout_commitment_hash = timeout_hash_result.get("hex_hash")
    
    # Commit to registering the timeout test name
    timeout_commit_result = dysond_bin("tx", "nameservice", "commit", "--commitment", timeout_commitment_hash, "--valuation", f"{bidding_valuation}dys", "--from", alice_name)
    assert timeout_commit_result["code"] == 0, "Timeout name commit transaction failed"
    
    # Reveal the timeout test name
    timeout_reveal_result = dysond_bin("tx", "nameservice", "reveal", "--name", timeout_name, "--salt", timeout_salt, "--from", alice_name)
    assert timeout_reveal_result["code"] == 0, "Timeout name reveal transaction failed"
    print(f"Registered timeout test name: {timeout_name}")
    
    # Charlie places a bid on the timeout test name
    timeout_bid_amount = bidding_valuation + random.randint(10, 30)
    timeout_bid_result = dysond_bin("tx", "nameservice", "place-bid", "--nft-class-id", "nameservice.dys", "--nft-id", timeout_name, "--bid-amount", f"{timeout_bid_amount}dys", "--from", charlie_name)
    assert timeout_bid_result["code"] == 0, "Charlie's place bid on timeout name failed"
    print(f"Charlie successfully placed a bid of {timeout_bid_amount}dys on {timeout_name}")
    
    # Verify the bid was recorded on the timeout test name
    timeout_nft_info = dysond_bin("query", "nft", "nft", "nameservice.dys", timeout_name)
    timeout_nft_data = timeout_nft_info["nft"].get("data", {}).get("value", {})
    assert timeout_nft_data.get("current_bid", {}).get("amount") == str(timeout_bid_amount), "Charlie's bid on timeout name not recorded correctly"
    
    # Wait for bid timeout by polling block time
    block = dysond_bin("query", "block")
    json_block = json.loads(block.split("\n")[1])
    start_block = int(json_block["header"]["height"])

    def timeout_elapsed():
        out = dysond_bin("query", "block")
        json_block = json.loads(out.split("\n")[1])
        current_block = int(json_block["header"]["height"])
        # With 100ms timeout and ~500ms block time, should pass after 1 block
        return (current_block - start_block) >= 1

    poll_until_condition(timeout_elapsed, timeout=10, error_message="Bid timeout did not elapse")

    claim_bid_result = dysond_bin("tx", "nameservice", "claim-bid", "--nft-class-id", "nameservice.dys", "--nft-id", timeout_name, "--from", charlie_name)
    assert claim_bid_result["code"] == 0, claim_bid_result["raw_log"]
    print("Charlie successfully claimed the NFT after timeout")
    
    # Verify Charlie now owns the NFT
    owner_data = dysond_bin("query", "nft", "owner", "nameservice.dys", timeout_name)
    assert owner_data, "Failed to get owner data"
    assert "owner" in owner_data, "Owner data missing owner field"
    assert owner_data["owner"] == charlie_address, f"NFT not transferred to Charlie: {owner_data['owner']} != {charlie_address}"
    print(f"Verified NFT ownership transferred to Charlie: {charlie_address}")

    print("Nameservice E2E test completed successfully!") 

# Helper: Register a name for Alice and return the name dict
@pytest.fixture
def register_name(chainnet, generate_account, faucet):
    dysond_bin = chainnet[0]
    [alice_name, alice_address] = generate_account('alice')
    faucet(alice_address, denom="dys", amount="25000")

    name = f"testname{alice_address[:6]}.dys"
    salt = "testsalt"
    hash_result = dysond_bin("query", "nameservice", "compute-hash", "--name", name, "--salt", salt, "--committer", alice_address)
    hex_hash = hash_result["hex_hash"]
    commit_result = dysond_bin("tx", "nameservice", "commit", "--commitment", hex_hash, "--valuation", "100dys", "--from", alice_name)
    assert commit_result["code"] == 0
    reveal_result = dysond_bin("tx", "nameservice", "reveal", "--name", name, "--salt", salt, "--from", alice_name)
    assert reveal_result["code"] == 0
    return {"name": name, "alice_name": alice_name, "alice_address": alice_address} 

def poll_until_proposal_passes(dysond_bin, proposal_id: str, timeout: int = 60):
    def has_proposal_reached_final_state():
        proposal_result = dysond_bin("query", "gov", "proposal", proposal_id)
        status = proposal_result["proposal"]["status"]
        print(f"Current proposal status: {status}")
        return status in [
            "PROPOSAL_STATUS_PASSED",
            "PROPOSAL_STATUS_REJECTED",
            "PROPOSAL_STATUS_FAILED"
        ]
    poll_until_condition(
        has_proposal_reached_final_state,
        timeout=timeout,
        error_message="Timeout waiting for proposal to reach final state"
    )
    proposal_result = dysond_bin("query", "gov", "proposal", proposal_id)
    final_status = proposal_result.get("status", "")
    return final_status

def set_bid_timeout_via_gov(dysond_bin, proposer_name, bid_timeout_value: str):
    """Set bid timeout via governance proposal"""
    # Get current params
    current_params = dysond_bin("query", "nameservice", "params")
    current_allowed_denoms = current_params.get("params", {}).get("allowed_denoms", ["dys"])
    current_reject_fee_percent = current_params.get("params", {}).get("reject_bid_valuation_fee_percent", "0.03")
    current_minimum_bid_percent_increase = current_params.get("params", {}).get("minimum_bid_percent_increase", "0.01")
    
    # Query gov module account address
    gov_module_response = dysond_bin("query", "auth", "module-account", "gov")
    gov_address = gov_module_response.get("account", {}).get("value", {}).get("address", "")
    assert gov_address

    # Get validator operator address
    validators = dysond_bin("query", "staking", "validators")
    validator_operator = validators["validators"][0]["operator_address"]
    
    # Delegate tokens from Alice to validator so Alice has voting power
    delegate_result = dysond_bin("tx", "staking", "delegate", validator_operator, "20000dys", "--from", "alice", "--yes")
    assert delegate_result["code"] == 0, f"Failed to delegate: {delegate_result['raw_log']}"

    proposal = {
        "messages": [
            {
                "@type": "/dysonprotocol.nameservice.v1.MsgUpdateParams",
                "authority": gov_address,
                "params": {
                    "bid_timeout": bid_timeout_value,
                    "allowed_denoms": current_allowed_denoms,
                    "reject_bid_valuation_fee_percent": current_reject_fee_percent,
                    "minimum_bid_percent_increase": current_minimum_bid_percent_increase
                }
            }
        ],
        "metadata": "ipfs://CID",
        "deposit": "1dys",
        "title": "Update Nameservice Parameters",
        "summary": f"Update bid_timeout to {bid_timeout_value} for testing"
    }

    # Submit proposal using Alice (who now has voting power)
    with tempfile.NamedTemporaryFile(mode="w", suffix=".json") as f:
        json.dump(proposal, f)
        f.flush()  # Ensure data is written to disk
        tx_result = dysond_bin("tx", "gov", "submit-proposal", f.name, "--from", "alice", "--keyring-backend", "test", "--yes")
    
    print(f"Submit proposal result: {tx_result}")
    tx_result = dysond_bin("query", "wait-tx", tx_result["txhash"])
    print(f"Proposal result: {tx_result}")

    # Extract proposal ID
    proposal_id = None
    for event in tx_result.get("events", []):
        if event.get("type") == "submit_proposal":
            for attr in event.get("attributes", []):
                if attr.get("key") == "proposal_id":
                    proposal_id = attr.get("value")
                    break
    assert proposal_id, "Could not extract proposal ID"

    # Vote with Alice (who has voting power through delegation)
    vote_result = dysond_bin("tx", "gov", "vote", proposal_id, "yes", "--from", "alice", "--keyring-backend", "test", "--yes")
    vote_tx_result = dysond_bin("query", "wait-tx", vote_result["txhash"])
    print(f"Vote result: {vote_tx_result}")
    assert vote_tx_result["code"] == 0, vote_tx_result["raw_log"]

    # Wait for proposal to pass
    poll_until_proposal_passes(dysond_bin, proposal_id)
    
    # Check final proposal status
    final_proposal = dysond_bin("query", "gov", "proposal", proposal_id)
    final_status = final_proposal["proposal"]["status"]
    assert final_status == "PROPOSAL_STATUS_PASSED", f"Governance proposal failed with status: {final_status}"

    # Verify params were updated
    params_result = dysond_bin("query", "nameservice", "params")
    print(f"Final nameservice params: {params_result}")
    assert params_result["params"]["bid_timeout"] == bid_timeout_value, f"Bid timeout not updated correctly" 