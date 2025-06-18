# Dyson Protocol – Everything On-chain

**Host Python scripts, serve decentralized websites, and run scheduled tasks with trustless, censorship-resistant execution. Trade names in a dynamic on-chain market, mint custom tokens and NFTs, and store arbitrary data—all fully on-chain.**

---

## What & Why

### Problem 
  - DApp UIs still load from centralized servers—developers host them off-chain, and end-users can't self-host or audit the code.

### Solution  
  - Store HTML/CSS/JS assets in the chain's storage so browsers load UI from the ledger.  
  - Push application logic on-chain and execute periodic jobs (crontasks) without any off-chain trigger.  
  - Run a dynamic on-chain name market using Harberger-style fees.  
  - Mint custom tokens and NFT classes based on on-chain names.  
  - Store arbitrary data in the chain's storage module.

### Key Use Cases  
  - **Autonomous payouts**: schedule hourly dividend distributions without users having to claim.  
  - **Timed auctions**: start and end bids exactly on-chain, with no external cron.  
  - **Game rounds**: progress players automatically through time-boxed stages.  
  - **Price oracles**: post market data at fixed intervals, fully on-chain.  
  - **Nameservice-driven assets**: register and trade domain-backed NFTs in a live marketplace.

### Outcome  
  - DWapp developers host, verify, and update every layer—from UI to scheduling—directly on the chain. No servers, no hidden dependencies, fully trustless.

---

## Install

### Requirements

Before building Dyson Protocol, ensure you have the following installed:

- **Go** (>= 1.24) - [Installation Guide](https://golang.org/doc/install)
- **Git** - For cloning repositories and submodules
- **Make** - Build automation tool
- **Python 3.11+ with venv** - `python3.11-venv` package (or equivalent for your system)
- **Docker** (optional) - [Installation Guide](https://docs.docker.com/get-docker/) (platform-specific)

> **🐧 Debian/Ubuntu Users:** For a complete automated installation, run:
> ```bash
> curl -fsSL https://raw.githubusercontent.com/dysonprotocol/dysonprotocol/master/install-debian.sh | bash
> ```
> This script installs all dependencies and builds Dyson Protocol automatically. See [`install-debian.sh`](./install-debian.sh) for details.

### Local Installation

1. **Clone the repository with submodules:**
   ```bash
   git clone --depth 1 --recurse-submodules https://github.com/dysonprotocol/dysonprotocol.git
   cd dysonprotocol
   ```

2. **Build DYSVM and install the binary:**
   ```bash
   make dysvm install
   ```

   This command will:
   - Verify system requirements
   - Build the custom Python VM (DYSVM) with embedded Python runtime
   - Compile and install the `dysond` binary to your `$GOPATH/bin`

3. **Verify installation:**
   ```bash
   dysond version --long
   ```

### Building and Running with Docker

For a containerized setup that includes all dependencies:

1. **Build the Docker image:**
   ```bash
   docker build -t dysonprotocol .
   ```

2. **Run with Docker:**
   ```bash
   # Start an interactive shell in the container
   docker run -it dysonprotocol

   # Or run specific commands
   docker run dysonprotocol -c "dysond version"
   ```

3. **Development with Docker:**
   ```bash
   # Mount your workspace for development
   docker run -it -v $(pwd):/app/dysonprotocol dysonprotocol
   ```

The Docker image includes:
- Go 1.24+ build environment
- Python 3 with development headers
- All build dependencies pre-installed
- Pre-built DYSVM and dysond binary

