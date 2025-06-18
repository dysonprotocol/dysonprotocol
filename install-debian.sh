#!/bin/bash

# Debian/Ubuntu Installation Script for Dyson Protocol
# This script installs all required dependencies and builds Dyson Protocol

set -e

echo "🚀 Installing Dyson Protocol on Debian/Ubuntu..."
echo "This script will install: Git, Make, Python 3.11+, Go 1.24.4, Docker, and build Dyson Protocol"
echo ""

# Update package list
echo "📦 Updating package list..."
sudo apt update

# Install basic dependencies
echo "📦 Installing Git, Make, and Python 3.11 with venv..."
sudo apt install -y git make python3.11-venv

# Install Go 1.24.4
echo "🐹 Installing Go 1.24.4..."
wget https://go.dev/dl/go1.24.4.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.24.4.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.profile
export PATH=$PATH:/usr/local/go/bin

# Clean up Go tarball
rm go1.24.4.linux-amd64.tar.gz

echo "✅ Go 1.24.4 installed successfully"
go version

# Install Docker
echo "🐳 Installing Docker..."

# Add Docker's official GPG key
sudo apt-get update
sudo apt-get install -y ca-certificates curl
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc

# Add the repository to Apt sources
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

echo "✅ Docker installed successfully"

# Clone and build Dyson Protocol
echo "🔗 Cloning Dyson Protocol repository..."
git clone --depth 1 --recurse-submodules https://github.com/dysonprotocol/dysonprotocol.git
cd dysonprotocol

echo "🔨 Building Dyson Protocol..."
make dysvm install

echo ""
echo "🎉 Installation completed successfully!"
echo ""
echo "Next steps:"
echo "1. Restart your terminal or run: source ~/.profile"
echo "2. Verify installation: dysond version --long"
echo "3. (Optional) Add your user to docker group: sudo usermod -aG docker $USER"
echo "4. (Optional) Log out and back in for docker group changes to take effect"
echo ""
echo "Repository location: $(pwd)" 