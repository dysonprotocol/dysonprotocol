"""
Test demonstrating how to wait for a specific number of blocks to be produced.
"""
import pytest
import time


def test_wait_for_blocks(chainnet):
    """Demonstrate waiting for blocks to be produced."""
    dysond = chainnet[0]
    
    # Get current height
    status_data = dysond("status")
    current_height = int(status_data.get("sync_info", {}).get("latest_block_height", 0))
    
    # Get initial block data
    initial_block_data = dysond("query", "block", "--type=height", str(current_height))
    initial_height = int(initial_block_data.get("header", {}).get("height", 0))
    
    # Wait for blocks to be produced by polling
    blocks_to_wait = 2
    target_height = initial_height + blocks_to_wait
    timeout = 5  # seconds
    start_time = time.time()
    
    while True:
        status_data = dysond("status")

        current_height = int(status_data.get("sync_info", {}).get("latest_block_height", 0))
        
        if current_height >= target_height:
            break
            
        if time.time() - start_time > timeout:
            pytest.fail(f"Timeout waiting for blocks. Current height: {current_height}, Target: {target_height}")
    
    # Get final block data
    final_block_data = dysond("query", "block", "--type=height", str(current_height))
    final_height = int(final_block_data.get("header", {}).get("height", 0))
    
    # Verify the height increased by at least the number of blocks we waited for
    assert final_height >= initial_height + blocks_to_wait, \
        f"Block height didn't increase as expected. Initial: {initial_height}, Final: {final_height}"
    
    print(f"Successfully waited for {blocks_to_wait} blocks to be produced.")
    print(f"Initial height: {initial_height}, Final height: {final_height}") 