import pytest
import json
import base64
import random
import string



def test_storage_set_get(chainnet, generate_account, faucet):
    """Test setting and retrieving a storage value."""
    dysond = chainnet[0]
    
    # Create Alice account and fund it
    [alice_name, alice_addr] = generate_account('alice')
    faucet(alice_addr)
    
    # Set a storage value for testing with unique suffix
    suffix = ''.join(random.choices(string.ascii_lowercase + string.digits, k=8))
    test_key = f"test_key_{suffix}"
    test_value = "test_value"
    
    # Set the storage value using Alice's account
    tx_result = dysond("tx", "storage", "set",
        "--from", alice_name,
        "--index", test_key,
        "--data", test_value)
    
    # Verify the transaction was successful
    assert tx_result["code"] == 0, f"Transaction failed: {tx_result['raw_log']}"
    
    # Query the storage value
    get_result = dysond("query", "storage", "get",
        alice_addr,
        "--index", test_key)
    
    # Print the result for inspection
    print(f"Storage get result: {json.dumps(get_result, indent=2)}")
    
    # Check if result has entry, storageValue or direct fields
    if "entry" in get_result:
        entry = get_result["entry"]
        assert entry["data"] == test_value, f"Retrieved value doesn't match: {entry}"
        assert entry["owner"] == alice_addr, f"Owner doesn't match: {entry}"
        assert entry["index"] == test_key, f"Index doesn't match: {entry}"
    elif "storageValue" in get_result:
        storage_value = get_result["storageValue"]
        assert storage_value.get("value", storage_value.get("data", "")) == test_value, \
            f"Retrieved value doesn't match: {storage_value}"
        assert storage_value["owner"] == alice_addr, f"Owner doesn't match: {storage_value}"
        assert storage_value["index"] == test_key, f"Index doesn't match: {storage_value}"
    else:
        # Direct field access
        assert get_result.get("data") == test_value or get_result.get("value") == test_value, \
            f"Retrieved value doesn't match: {get_result}"
        assert get_result.get("owner") == alice_addr, f"Owner doesn't match: {get_result}"
        assert get_result.get("index") == test_key, f"Index doesn't match: {get_result}"


def test_storage_list(chainnet, generate_account, faucet):
    """Test listing storage values with a prefix."""
    dysond = chainnet[0]
    
    # Create Alice account and fund it
    [alice_name, alice_addr] = generate_account('alice')
    faucet(alice_addr)
    
    # Set multiple storage values with a common prefix
    suffix = ''.join(random.choices(string.ascii_lowercase + string.digits, k=8))
    prefix = f"list_test_{suffix}_"
    values = {
        f"{prefix}1": "value1",
        f"{prefix}2": "value2",
        f"{prefix}3": "value3"
    }
    
    # Set each value in storage
    for key, value in values.items():
        dysond("tx", "storage", "set",
            "--from", alice_name,
            "--index", key,
            "--data", value)
    
    # List all keys with the given prefix
    list_result = dysond("query", "storage", "list",
        alice_addr,
        "--index-prefix", prefix,
        "-o", "json")
    
    # Print the result for inspection
    print(f"Storage list result: {json.dumps(list_result, indent=2)}")
    
    # Check different possible structures for the list result
    if "entries" in list_result:
        storage_items = list_result["entries"]
    elif "storageValues" in list_result:
        storage_items = list_result["storageValues"]
    elif "storage_values" in list_result:
        storage_items = list_result["storage_values"]
    else:
        # Assume the result itself is the list of items
        storage_items = list_result
    
    # Extract the values with a more flexible approach
    found_items = {}
    for item in storage_items:
        if isinstance(item, dict):
            if "index" in item and ("data" in item or "value" in item):
                # Direct structure
                index = item.get("index", "")
                value = item.get("data", item.get("value", ""))
                found_items[index] = value
            elif "entry" in item:
                # Entry wrapper
                entry = item["entry"]
                index = entry.get("index", "")
                value = entry.get("data", entry.get("value", ""))
                found_items[index] = value
    
    # Check that all our values were found
    for key, value in values.items():
        assert key in found_items, f"Key {key} not found in storage list"
        assert found_items[key] == value, f"Value mismatch for key {key}: expected {value}, got {found_items[key]}"
    
    # Verify the count
    assert len(storage_items) >= len(values), "Not all values were listed"


def test_storage_delete(chainnet, generate_account, faucet):
    """Test deleting storage values."""
    dysond = chainnet[0]
    
    # Create Alice account and fund it
    [alice_name, alice_addr] = generate_account('alice')
    faucet(alice_addr)
    
    # First set a storage value
    suffix = ''.join(random.choices(string.ascii_lowercase + string.digits, k=8))
    test_key = f"delete_test_key_{suffix}"
    test_value = "delete_test_value"
    
    # Set the storage value
    dysond("tx", "storage", "set",
        "--from", alice_name,
        "--index", test_key,
        "--data", test_value)
    
    # Verify it was set correctly
    get_result = dysond("query", "storage", "get",
        alice_addr,
        "--index", test_key)
    
    assert get_result["entry"]["data"] == test_value, f"Value not set correctly for deletion test: expected {test_value}, got {get_result['entry']['data']}"
    
    # Delete the storage value
    delete_result = dysond("tx", "storage", "delete",
        "--from", alice_name,
        "--indexes", test_key)
    
    # Verify the deletion was successful
    assert delete_result["code"] == 0, f"Delete transaction failed: {delete_result['raw_log']}"
    
    # Query the deleted value - should return error message for deleted entries
    get_result = dysond("query", "storage", "get",
        alice_addr,
        "--index", test_key)
    
    # When a storage entry doesn't exist, the query returns an error string
    assert isinstance(get_result, str), f"Expected string error message for deleted entry, got: {type(get_result)}"
    assert "doesn't exist" in get_result, f"Expected 'doesn't exist' error, got: {get_result}"


def test_storage_multi_user(chainnet, generate_account, faucet):
    """Test storage with multiple users and access control."""
    dysond = chainnet[0]
    
    # Create accounts for Alice and Bob
    [alice_name, alice_addr] = generate_account('alice')
    [bob_name, bob_addr] = generate_account('bob')
    
    # Fund both accounts for transactions
    faucet(alice_addr)
    faucet(bob_addr)
    
    # Create unique test keys for each user
    suffix = ''.join(random.choices(string.ascii_lowercase + string.digits, k=8))
    alice_key = f"alice_storage_key_{suffix}"
    bob_key = f"bob_storage_key_{suffix}"
    alice_value = "alice_value"
    bob_value = "bob_value"
    
    # Alice sets her storage value
    alice_set_result = dysond("tx", "storage", "set",
        "--from", alice_name,
        "--index", alice_key,
        "--data", alice_value)
    assert alice_set_result["code"] == 0, f"Alice failed to set storage: {alice_set_result['raw_log']}"
    
    # Bob sets his storage value
    bob_set_result = dysond("tx", "storage", "set",
        "--from", bob_name,
        "--index", bob_key,
        "--data", bob_value)
    assert bob_set_result["code"] == 0, f"Bob failed to set storage: {bob_set_result['raw_log']}"
    
    # Verify Alice's value is retrievable
    alice_result = dysond("query", "storage", "get",
        alice_addr,
        "--index", alice_key)
    
    assert alice_result["entry"]["data"] == alice_value, f"Alice's value not set correctly: expected {alice_value}, got {alice_result['entry']['data']}"
    assert alice_result["entry"]["owner"] == alice_addr, f"Alice's owner not correct: expected {alice_addr}, got {alice_result['entry']['owner']}"
    assert alice_result["entry"]["index"] == alice_key, f"Alice's index not correct: expected {alice_key}, got {alice_result['entry']['index']}"
    
    # Verify Bob's value is retrievable
    bob_result = dysond("query", "storage", "get",
        bob_addr,
        "--index", bob_key)
    
    assert bob_result["entry"]["data"] == bob_value, f"Bob's value not set correctly: expected {bob_value}, got {bob_result['entry']['data']}"
    assert bob_result["entry"]["owner"] == bob_addr, f"Bob's owner not correct: expected {bob_addr}, got {bob_result['entry']['owner']}"
    assert bob_result["entry"]["index"] == bob_key, f"Bob's index not correct: expected {bob_key}, got {bob_result['entry']['index']}"
    
    # Verify that Alice cannot delete Bob's value - should fail with error code
    delete_result = dysond("tx", "storage", "delete",
        "--from", alice_name,
        "--indexes", bob_key)
    
    assert delete_result["code"] != 0, f"Alice should not be able to delete Bob's storage, but transaction succeeded: {delete_result}"
    assert "no entries were deleted" in delete_result["raw_log"], f"Expected 'no entries were deleted' error, got: {delete_result['raw_log']}"


def test_storage_binary_data(chainnet, generate_account, faucet):
    """Test storing and retrieving binary data."""
    dysond = chainnet[0]
    
    # Create Alice account and fund it
    [alice_name, alice_addr] = generate_account('alice')
    faucet(alice_addr)
    
    # Create binary data (base64 encoded)
    binary_data = base64.b64encode(b"Binary test data").decode('utf-8')
    suffix = ''.join(random.choices(string.ascii_lowercase + string.digits, k=8))
    test_key = f"binary_data_key_{suffix}"
    
    # Set the binary data
    dysond("tx", "storage", "set",
        "--from", alice_name,
        "--index", test_key,
        "--data", binary_data)
    
    # Retrieve the binary data
    get_result = dysond("query", "storage", "get",
        alice_addr,
        "--index", test_key)
    
    # Print result for inspection
    print(f"Binary data result: {json.dumps(get_result, indent=2)}")
    
    # Get the value with flexible structure handling
    value = ""
    if "entry" in get_result:
        value = get_result["entry"].get("data", get_result["entry"].get("value", ""))
    elif "storageValue" in get_result:
        value = get_result["storageValue"].get("value", get_result["storageValue"].get("data", ""))
    else:
        value = get_result.get("value", get_result.get("data", ""))
    
    # Verify the data
    assert value == binary_data, "Binary data not retrieved correctly"
    
    # Verify we can decode it back
    decoded = base64.b64decode(value)
    assert decoded == b"Binary test data", "Binary data corrupted in storage"


def test_storage_extract_and_filter(chainnet, generate_account, faucet):
    """Test the new --extract and --filter query flags."""
    dysond = chainnet[0]

    # Create user and fund
    [user_name, user_addr] = generate_account('extractor')
    faucet(user_addr)

    # Prepare JSON payloads
    json_entry_1 = {
        "title": "First Post",
        "category": "blog",
        "meta": {"views": 10}
    }
    json_entry_2 = {
        "title": "Second Post",
        "category": "blog",
        "meta": {"views": 20}
    }
    json_entry_3 = {
        "title": "Draft Note",
        "meta": {"views": 0}
    }

    # Helper to set entry
    def set_json(index: str, data: dict):
        dysond(
            "tx",
            "storage",
            "set",
            "--from",
            user_name,
            "--index",
            index,
            "--data",
            json.dumps(data),
        )

    prefix = "posts/"
    set_json(prefix + "1", json_entry_1)
    set_json(prefix + "2", json_entry_2)
    set_json(prefix + "draft", json_entry_3)

    # Test --extract on single get
    get_res = dysond(
        "query",
        "storage",
        "get",
        user_addr,
        "--index",
        prefix + "1",
        "--extract",
        "title",
    )
    # entry.data should now be the string "First Post" (with quotes)
    extracted = get_res["entry"]["data"]
    # Remove surrounding quotes if present
    if extracted.startswith("\"") and extracted.endswith("\""):
        extracted = json.loads(extracted)
    assert extracted == "First Post", f"extract failed: {extracted}"

    # Test --filter when listing
    list_res = dysond(
        "query",
        "storage",
        "list",
        user_addr,
        "--index-prefix",
        prefix,
        "--filter",
        "category",
        "-o",
        "json",
    )
    entries = list_res.get("entries", [])
    # Should contain only 2 items (those with category)
    assert len(entries) == 2, f"filter expected 2 entries, got {len(entries)}"

    # Validate extract works in list too
    list_extract = dysond(
        "query",
        "storage",
        "list",
        user_addr,
        "--index-prefix",
        prefix,
        "--filter",
        "category",
        "--extract",
        "meta.views",
        "-o",
        "json",
    )
    entries_views = list_extract.get("entries", [])
    views_values = [json.loads(e["data"]) if isinstance(e["data"], str) and e["data"].startswith("\"") else int(e["data"]) for e in entries_views]
    assert set(views_values) == {10, 20}, f"extract in list failed, got {views_values}"


def test_storage_invalid_extract_and_filter(chainnet, generate_account, faucet):
    """Ensure invalid extract path raises error and unmatched filter returns empty list."""
    dysond = chainnet[0]

    [u_name, u_addr] = generate_account('neg')
    faucet(u_addr)

    entry = {"foo": {"bar": 1}}
    dysond("tx", "storage", "set", "--from", u_name, "--index", "neg/1", "--data", json.dumps(entry))

    # Attempt to extract missing path -> expect string error (gRPC NotFound propagated to CLI)
    res = dysond("query", "storage", "get", u_addr, "--index", "neg/1", "--extract", "foo.baz")
    assert isinstance(res, str), "Expected error string when extract path missing"
    assert "not found" in res.lower(), f"Unexpected error message: {res}"

    # Filter that matches nothing should return 0 entries
    list_res = dysond(
        "query", "storage", "list", u_addr, "--index-prefix", "neg/", "--filter", "nonexistent", "-o", "json"
    )
    entries = list_res.get("entries", [])
    assert entries == [] or len(entries) == 0, f"Expected empty list, got {entries}"


def test_storage_extract_filter_too_long(chainnet, generate_account, faucet):
    dysond = chainnet[0]
    [name, addr] = generate_account('toolong')
    faucet(addr)

    long_path = 'a' * 101
    dysond("tx", "storage", "set", "--from", name, "--index", "toolong/1", "--data", "{}")

    res = dysond("query", "storage", "get", addr, "--index", "toolong/1", "--extract", long_path)
    assert isinstance(res, str) and "too long" in res.lower(), f"Expected length error, got {res}"

    list_res = dysond("query", "storage", "list", addr, "--index-prefix", "toolong/", "--filter", long_path, "-o", "json")
    # For list, CLI likely surfaces error string instead of json when InvalidArgument
    if isinstance(list_res, dict):
        assert False, "Expected error string for too long filter"
    else:
        assert "too long" in list_res.lower(), f"Expected length error, got {list_res}" 