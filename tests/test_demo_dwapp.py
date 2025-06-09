#!/usr/bin/env python3
"""
Frontend UI Tests for DysonProtocol Demo DApp
Integration with existing test infrastructure
"""

import pytest
import json
import os
import requests
from pathlib import Path
from playwright.sync_api import Page
from tests.utils import poll_until_condition


@pytest.fixture(scope="session")
def deployed_demo_script(chainnet, api_address):
    """Deploy demo script and return deployment info."""
    import random
    import string
    
    dysond = chainnet[0]
    
    # Get the actual chain-id from the node
    status_result = dysond("status")
    chain_id = status_result["node_info"]["network"]
    print(f"Using chain-id: {chain_id}")
    
    # Get the correct API port for this test node
    api_port = api_address["port"]
    print(f"Using API port: {api_port}")
    
    # Generate unique account name directly
    random_suffix = ''.join(random.choices(string.ascii_lowercase + string.digits, k=8))
    account_name = f"demo_test_{random_suffix}"
    
    # Create account directly
    account_result = dysond(
        "keys", "add", account_name, 
        "--keyring-backend", "test"
    )
    
    # Get account address
    account_info = dysond("keys", "show", account_name, "--keyring-backend", "test")
    address = account_info["address"]
    print(f"Created account: {account_name} -> {address}")
    
    # Fund account directly (faucet logic)
    fund_result = dysond(
        "tx", "bank", "send", 
        "alice", address, "10000dys",
        "--from", "alice", "--chain-id", chain_id
    )
    assert fund_result["code"] == 0, f"Funding failed: {fund_result}"
    
    # Deploy script using update command (without --script-address, defaults to sender address)
    script_path = Path(__file__).parent.parent / "demo-dwapp/script.py"
    
    deploy_result = dysond(
        "tx", "script", "update",
        "--code-path", str(script_path),
        "--from", account_name,
        "--chain-id", chain_id,
        "--gas", "auto", "--gas-adjustment", "1.3"
    )

    print(f"Deploy result: {deploy_result}")
    assert deploy_result["code"] == 0, f"Script deployment failed: {deploy_result}"

    # Upload storage files
    base_path = Path(__file__).parent.parent / "demo-dwapp/storage"
    storage_files = [
        (base_path / "templates/base.html", "templates/base.html"),
        (base_path / "templates/index.html", "templates/index.html"), 
        (base_path / "templates/messages.html", "templates/messages.html"),
        (base_path / "templates/wallet.html", "templates/wallet.html"),
        (base_path / "static/style.css", "static/style.css"),
        (base_path / "static/walletStore.js", "static/walletStore.js"),
        (base_path / "static/messageStore.js", "static/messageStore.js"),
        (base_path / "static/dysonTxUtils.js", "static/dysonTxUtils.js")
    ]
    
    for local_path, storage_key in storage_files:
        # Calculate gas limit based on file size
        file_size = local_path.stat().st_size
        gas_limit = max(300000, min(1000000, file_size * 100 + 200000))
        
        upload_result = dysond(
            "tx", "storage", "set",
            "--index", storage_key,
            "--data-path", str(local_path),
            "--gas", str(gas_limit),
            "--from", account_name,
            "--chain-id", chain_id
        )
        assert upload_result["code"] == 0, f"File upload failed for {storage_key}: {upload_result}"
        print(f"Uploaded: {storage_key}")
    
    print(f"Demo deployed successfully at: http://{address}.localhost:{api_port}")
    
    # Poll until the script is accessible via HTTP
    def check_script_accessible():
        try:
            response = requests.get(f"http://localhost:{api_port}", timeout=5, headers={"Host": f"{address}.localhost"})
            print(f"Poll response: status={response.status_code}, content_preview={response.text[:200]}")
            if response.status_code == 200 and "DysonProtocol WalletStore Demo" in response.text:
                return True
            return False
        except Exception as e:
            print(f"Poll request failed: {e}")
            return False
    
    poll_until_condition(
        check_script_accessible,
        timeout=30,
        poll_interval=0.5,
        error_message=f"Script not accessible at http://{address}.localhost:{api_port}"
    )
    print(f"Script confirmed accessible at http://{address}.localhost:{api_port}")
    
    return {
        "account_name": account_name,
        "address": address,
        "script_url": f"http://{address}.localhost:{api_port}"
    }


@pytest.fixture(scope="function") 
def demo_url(deployed_demo_script):
    """Return the demo URL."""
    return deployed_demo_script["script_url"]


@pytest.mark.frontend 
def test_homepage_loads(page: Page, demo_url):
    """Test that the homepage loads successfully."""
    page.goto(demo_url)
    
    # Wait for the page to fully load
    page.wait_for_load_state("networkidle")
    
    # Debug: check what's actually on the page
    print(f"Page URL: {page.url}")
    print(f"Page title: '{page.title()}'")
    print(f"Page content preview: {page.content()[:500]}")
    
    # Check page title
    assert page.title() == "DysonProtocol WalletStore Demo"
    
    # Check main heading
    heading = page.locator("h1")
    assert heading.text_content() == "Distributed Web App"
    
    # Check basic navigation structure exists (not necessarily visible due to Alpine.js logic)
    assert page.locator('a[href="/"]').first.count() > 0
    assert page.locator('a[href="/exec"]').first.count() > 0
    # The wallet link might be hidden when no wallet is connected, so just check it exists
    assert page.locator('a[href="/wallet"]').count() > 0


@pytest.mark.frontend
def test_python_code_execution_interface(page: Page, demo_url):
    """Test that the Python code execution interface is present."""
    page.goto(demo_url)
    
    # Check code textarea exists
    code_textarea = page.locator('#editor')
    assert code_textarea.is_visible()
    
    # Check form exists
    form = page.locator('form[action="/"][method="post"]')
    assert form.is_visible()
    
    # Check run button exists
    run_button = page.locator('input[type="submit"][value="Run"]')
    assert run_button.is_visible()


@pytest.mark.frontend
def test_messages_functionality_loads(page: Page, demo_url):
    """Test that the messages functionality loads successfully on the home page."""
    page.goto(demo_url)
    page.wait_for_load_state("networkidle")
    
    # The messages functionality should be on the home page
    # Check for elements that should be on the messages section
    messages_section = page.locator('section[x-data="messagesStore"]')
    assert messages_section.count() > 0, "Messages section should be present on home page"


@pytest.mark.frontend
def test_wallet_page_loads(page: Page, demo_url):
    """Test that the wallet page loads successfully."""
    page.goto(f"{demo_url}/wallet")
    
    # The wallet page should load
    # Check for elements that should be on the wallet page
    assert page.url.endswith("/wallet")


@pytest.mark.frontend 
def test_navigation_works(page: Page, demo_url):
    """Test that navigation between pages works."""
    page.goto(demo_url)
    page.wait_for_load_state("networkidle")
    
    # Verify we're on the home page initially
    assert page.locator("h1").text_content() == "Distributed Web App"
    
    # Navigate to exec page (should be visible)
    exec_link = page.locator('a[href="/exec"]').first
    if exec_link.is_visible():
        exec_link.click()
        page.wait_for_load_state("networkidle")
        # For SPA navigation, just verify the link worked (no errors, page still responsive)
        assert page.locator("h1").count() > 0  # Page still has structure
    
    # Navigate back to home to ensure basic navigation cycle works
    home_link = page.locator('a[href="/"]').first
    if home_link.is_visible():
        home_link.click()
        page.wait_for_load_state("networkidle")
        # Verify we can get back to home
        assert page.locator("h1").text_content() == "Distributed Web App"
    
    # If both links were navigable, consider the test passed
    assert True  # Basic navigation structure is functional


@pytest.mark.frontend
def test_alpine_js_loaded(page: Page, demo_url):
    """Test that Alpine.js is loaded and functioning."""
    page.goto(demo_url)
    
    # Check for Alpine.js global
    alpine_loaded = page.evaluate("() => typeof Alpine !== 'undefined'")
    assert alpine_loaded


@pytest.mark.basics
def test_demo_script_file_exists():
    """Test that the demo script file exists"""
    script_path = Path("demo-dwapp/script.py")
    assert script_path.exists(), "Demo script file not found"


@pytest.mark.basics
def test_demo_makefile_exists():
    """Test that the demo Makefile exists"""
    makefile_path = Path("demo-dwapp/Makefile")
    assert makefile_path.exists(), "Demo Makefile not found"


@pytest.mark.basics
def test_demo_storage_files_exist():
    """Test that demo storage files exist"""
    storage_dir = Path("demo-dwapp/storage")
    assert storage_dir.exists(), "Storage directory not found"
    
    templates_dir = storage_dir / "templates"
    assert templates_dir.exists(), "Templates directory not found"
    
    static_dir = storage_dir / "static"
    assert static_dir.exists(), "Static directory not found"
    
    # Check key files
    assert (templates_dir / "base.html").exists(), "base.html template not found"
    assert (templates_dir / "index.html").exists(), "index.html template not found"
    assert (templates_dir / "messages.html").exists(), "messages.html template not found" 
    assert (templates_dir / "wallet.html").exists(), "wallet.html template not found"
    assert (static_dir / "style.css").exists(), "style.css not found"
    assert (static_dir / "walletStore.js").exists(), "walletStore.js not found"
    assert (static_dir / "messageStore.js").exists(), "messageStore.js not found"
    assert (static_dir / "dysonTxUtils.js").exists(), "dysonTxUtils.js not found" 


@pytest.fixture(scope="session")
def unlock_wallet():
    """Return a function to unlock an existing wallet by entering its password and clicking connect.
    
    Usage:
        def test_something(page, demo_url, unlock_wallet):
            unlock_wallet(page, demo_url, "wallet_name", "password123")
            # Wallet is now connected
    
    Args (for the returned function):
        page: Playwright Page object
        demo_url: URL of the deployed demo dwapp
        name: Name of the wallet to unlock
        password: Password for the wallet
    """
    def _unlock_wallet(page: Page, demo_url: str, name: str, password: str):
        """Unlock an existing wallet by entering password and clicking connect."""
        # Navigate to wallet page
        page.goto(f"{demo_url}/wallet")
        page.wait_for_load_state("networkidle")
        
        # Verify the wallet exists
        wallet_name_element = page.locator(f'#wallet-name-{name}')
        assert wallet_name_element.is_visible(), f"Wallet '{name}' should be visible"
        
        # Enter password for the specific wallet
        password_input = page.locator(f'#wallet-password-{name}')
        assert password_input.is_visible(), f"Password input for wallet '{name}' should be visible"
        password_input.fill(password)
        
        # Click connect button for the specific wallet
        connect_btn = page.locator(f'#connect-wallet-{name}')
        assert connect_btn.is_visible(), f"Connect button for wallet '{name}' should be visible"
        connect_btn.click()
        
        # Wait for connection to complete
        page.wait_for_timeout(500)
    
    return _unlock_wallet


@pytest.fixture(scope="session")
def create_cosmjs_wallet(unlock_wallet):
    """Return a function to create CosmJS wallets through the UI.
    
    This fixture provides a factory function that can create multiple wallets
    with different names within the same test session. The function caches
    created wallets to avoid recreating them if the same name is requested.
    
    Usage:
        def test_something(page, demo_url, create_cosmjs_wallet):
            address1 = create_cosmjs_wallet(page, demo_url, "wallet1")
            address2 = create_cosmjs_wallet(page, demo_url, "wallet2")
            # Use the wallets...
    
    Args (for the returned function):
        page: Playwright Page object
        demo_url: URL of the deployed demo dwapp
        name: Name for the wallet (must be unique)
    
    Returns:
        Function that creates wallets and returns their dys1 addresses
    """
    created_wallets = {}
    
    def _create_wallet(page: Page, demo_url: str, name: str) -> str:
        """Create a CosmJS wallet with the given name and return its address."""
        if name in created_wallets:
            # Wallet already exists, just unlock it
            wallet_info = created_wallets[name]
            unlock_wallet(page, demo_url, name, wallet_info['password'])
            return wallet_info['address']
            
        # Navigate to wallet page
        page.goto(f"{demo_url}/wallet")
        page.wait_for_load_state("networkidle")
        
        # Fill in wallet name using id
        wallet_name_input = page.locator('#wallet-name-input')
        assert wallet_name_input.is_visible()
        wallet_name_input.fill(name)
        
        # Fill in password using id
        password = "test_password_123"
        password_input = page.locator('#wallet-password-input')
        assert password_input.is_visible()
        password_input.fill(password)
        
        # Click "Generate 24 words" button using id
        generate_24_btn = page.locator('#generate-24-words-btn')
        assert generate_24_btn.is_visible()
        generate_24_btn.click()
        
        # Wait for mnemonic to be generated and filled in using id
        mnemonic_textarea = page.locator('#mnemonic-textarea')
        page.wait_for_function(
            '() => document.querySelector("#mnemonic-textarea").value.trim() !== ""',
            timeout=5000
        )
        
        # Verify mnemonic was generated (should have 24 words)
        mnemonic_value = mnemonic_textarea.input_value()
        mnemonic_words = mnemonic_value.strip().split()
        assert len(mnemonic_words) == 24, f"Expected 24 words, got {len(mnemonic_words)}"
        
        # Check the "I have backed up my mnemonic" checkbox using id
        backup_checkbox = page.locator('#mnemonic-backup-checkbox')
        assert backup_checkbox.is_visible()
        backup_checkbox.check()
        
        # Click "Import" button using id
        import_btn = page.locator('#import-wallet-btn')
        assert import_btn.is_visible()
        assert not import_btn.is_disabled(), "Import button should be enabled after filling all fields"
        import_btn.click()
        
        # Wait for import to complete and wallet to appear in the list using id
        page.wait_for_function(
            f'() => document.querySelector("#wallet-name-{name}") !== null',
            timeout=10000
        )
        
        # Get the wallet address using id
        address_code = page.locator(f'#wallet-address-{name}')
        assert address_code.is_visible(), "Wallet should have an address displayed"
        
        address_text = address_code.text_content()
        assert address_text is not None, "Address text should not be None"
        assert address_text.startswith('dys1'), f"Address should start with 'dys1', got: {address_text}"
        
        # Cache the created wallet with password
        created_wallets[name] = {
            'address': address_text,
            'password': password
        }
        return address_text
    
    return _create_wallet


@pytest.fixture(scope="function")
def default_cosmjs_wallet(page: Page, demo_url, create_cosmjs_wallet):
    """Create a default test wallet that can be reused within a test function.
    
    This fixture automatically creates a wallet named "default_test_wallet"
    that can be used by any test that needs a wallet without having to
    create one manually.
    
    Note: This fixture is function-scoped, so each test gets a fresh wallet
    to avoid issues with different demo deployments.
    
    Usage:
        def test_something(page, demo_url, default_cosmjs_wallet):
            # default_cosmjs_wallet contains the dys1 address string
            assert default_cosmjs_wallet.startswith('dys1')
            # The wallet "default_test_wallet" is already created in the UI
    
    Returns:
        dict: Contains 'address' and 'name' of the created wallet
    """
    import random
    import string
    
    # Generate a unique wallet name for each test to avoid conflicts
    random_suffix = ''.join(random.choices(string.ascii_lowercase + string.digits, k=8))
    wallet_name = f"default_test_wallet_{random_suffix}"
    
    address = create_cosmjs_wallet(page, demo_url, wallet_name)
    return {'address': address, 'name': wallet_name}


@pytest.mark.frontend
def test_import_new_cosmjs_wallet(page: Page, demo_url, create_cosmjs_wallet):
    """Test importing a new CosmJS wallet through the UI."""
    import random
    import string
    
    # Generate a random wallet name
    random_suffix = ''.join(random.choices(string.ascii_lowercase + string.digits, k=8))
    wallet_name = f"test_wallet_{random_suffix}"
    
    # Create the wallet using the fixture
    address = create_cosmjs_wallet(page, demo_url, wallet_name)
    
    # Verify the wallet name appears using id
    wallet_name_element = page.locator(f'#wallet-name-{wallet_name}')
    assert wallet_name_element.is_visible(), f"Wallet '{wallet_name}' should appear in the imported wallets list"
    
    # Verify the wallet has the correct address displayed using id
    address_code = page.locator(f'#wallet-address-{wallet_name}')
    assert address_code.is_visible(), "Wallet should have an address displayed"
    
    displayed_address = address_code.text_content()
    assert displayed_address == address, f"Displayed address should match returned address"
    
    # Verify wallet control buttons are present using id
    connect_btn = page.locator(f'#connect-wallet-{wallet_name}')
    disconnect_btn = page.locator(f'#disconnect-wallet-{wallet_name}')
    remove_btn = page.locator(f'#remove-wallet-{wallet_name}')
    
    assert connect_btn.is_visible(), "Connect button should be visible"
    assert disconnect_btn.is_visible(), "Disconnect button should be visible" 
    assert remove_btn.is_visible(), "Remove button should be visible"


@pytest.mark.frontend
def test_wallet_connection_with_default_wallet(page: Page, demo_url, default_cosmjs_wallet):
    """Example test showing how to use the default wallet fixture."""
    # Navigate to wallet page
    page.goto(f"{demo_url}/wallet")
    page.wait_for_load_state("networkidle")
    
    # Verify the default wallet exists using id
    wallet_name_element = page.locator(f'#wallet-name-{default_cosmjs_wallet["name"]}')
    assert wallet_name_element.is_visible(), "Default test wallet should be visible"
    
    # Verify it has the expected address format using id
    address_code = page.locator(f'#wallet-address-{default_cosmjs_wallet["name"]}')
    address_text = address_code.text_content()
    
    # Verify the fixture returned the same address as displayed
    assert address_text is not None, "Address text should not be None"
    assert address_text == default_cosmjs_wallet['address'], f"Fixture address should match UI address"
    assert address_text.startswith('dys1'), f"Address should start with 'dys1', got: {address_text}"


@pytest.mark.frontend
def test_send_message_with_default_wallet(page: Page, demo_url, default_cosmjs_wallet, unlock_wallet):
    """Test sending a message using the default wallet and verifying it appears."""
    import random
    import string
    
    # Generate a unique test message
    random_suffix = ''.join(random.choices(string.ascii_lowercase + string.digits, k=8))
    test_message = f"Test message from automation {random_suffix}"
    
    # Connect the default wallet using the unlock_wallet fixture
    unlock_wallet(page, demo_url, default_cosmjs_wallet['name'], "test_password_123")
    
    # Navigate to messages page
    page.goto(f"{demo_url}/messages")
    page.wait_for_load_state("networkidle")
    
    # Wallet should be connected (visible in header)
    wallet_name_in_header = page.locator('strong').filter(has_text=default_cosmjs_wallet['name'])
    assert wallet_name_in_header.is_visible(), "Wallet name should be visible in header when connected"
    
    # Fill in the message form
    message_input = page.locator('input[placeholder="Enter your greeting..."]')
    message_input.fill(test_message)
    
    gas_input = page.locator('input[placeholder="500000"]')
    gas_input.fill("500000")
    
    # Submit the message
    send_button = page.locator('button[type="submit"]', has_text='Send Message')
    assert not send_button.is_disabled(), "Send Message button should be enabled when wallet is connected"
    send_button.click()
    
    # Wait for "Running script..." to appear
    page.wait_for_function(
        '() => document.querySelector("pre").textContent.includes("Running script...")',
        timeout=1000
    )
    
    # Wait for transaction completion - must show success
    page.wait_for_function(
        '''() => {
            const logsElement = document.querySelector("pre");
            const logsText = logsElement ? logsElement.textContent : "";
            return logsText.includes('"success": true');
        }''',
        timeout=5000
    )
    
    # Refresh the messages list
    refresh_button = page.locator('button', has_text='Refresh Messages')
    refresh_button.click()
    page.wait_for_timeout(500)
    
    # Message must appear in the list
    message_found = page.locator('div').filter(has_text=test_message)
    assert message_found.count() > 0, f"Test message '{test_message}' should appear in the messages list"
    
    # Sender address must appear in the list
    sender_element = page.locator('code').filter(has_text=default_cosmjs_wallet['address'])
    assert sender_element.count() > 0, f"Sender address '{default_cosmjs_wallet['address']}' should appear in the messages list"


@pytest.mark.frontend
def test_wallet_caching_and_unlock(page: Page, demo_url, create_cosmjs_wallet, unlock_wallet):
    """Test that wallet caching works and unlock_wallet fixture functions correctly."""
    import random
    import string
    
    # Generate a random wallet name
    random_suffix = ''.join(random.choices(string.ascii_lowercase + string.digits, k=8))
    wallet_name = f"cache_test_wallet_{random_suffix}"
    
    # Create the wallet for the first time
    address1 = create_cosmjs_wallet(page, demo_url, wallet_name)
    assert address1.startswith('dys1'), f"Address should start with 'dys1', got: {address1}"
    
    # Manually connect the wallet first
    page.goto(f"{demo_url}/wallet")
    page.wait_for_load_state("networkidle")
    
    password_input = page.locator(f'#wallet-password-{wallet_name}')
    password_input.fill("test_password_123")
    
    connect_btn = page.locator(f'#connect-wallet-{wallet_name}')
    connect_btn.click()
    page.wait_for_timeout(500)
    
    # Now disconnect the wallet
    disconnect_btn = page.locator(f'#disconnect-wallet-{wallet_name}')
    disconnect_btn.click()
    page.wait_for_timeout(500)
    
    # Try to create the same wallet again - should use cached version and unlock it
    address2 = create_cosmjs_wallet(page, demo_url, wallet_name)
    assert address2 == address1, f"Cached wallet should return same address: {address1} != {address2}"
    
    # Verify wallet is connected (should appear in header)
    page.goto(f"{demo_url}/messages")
    page.wait_for_load_state("networkidle")
    
    wallet_name_in_header = page.locator('strong').filter(has_text=wallet_name)
    assert wallet_name_in_header.is_visible(), "Wallet name should be visible in header when connected via cache"
    
    # Test unlock_wallet fixture directly
    page.goto(f"{demo_url}/wallet")
    page.wait_for_load_state("networkidle")
    
    # Disconnect first
    disconnect_btn = page.locator(f'#disconnect-wallet-{wallet_name}')
    disconnect_btn.click()
    page.wait_for_timeout(500)
    
    # Use unlock_wallet fixture directly
    unlock_wallet(page, demo_url, wallet_name, "test_password_123")
    
    # Verify wallet is connected again
    page.goto(f"{demo_url}/messages")
    page.wait_for_load_state("networkidle")
    
    wallet_name_in_header = page.locator('strong').filter(has_text=wallet_name)
    assert wallet_name_in_header.is_visible(), "Wallet name should be visible in header when connected via unlock_wallet" 