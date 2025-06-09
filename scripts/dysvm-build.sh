#!/bin/bash

# Enable verbose mode and exit on error
set -ex

# Script for building custom Python distributions

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DYSVM_DIR="$REPO_ROOT/dysvm"
CPYTHON_DIR="$DYSVM_DIR/cpython"
PYTHON_BUILD_STANDALONE_DIR="$DYSVM_DIR/python-build-standalone"

# Python version for DYSVM
PYTHON_VERSION="3.11.8"
PYTHON_VERSION_SHORT=$(echo $PYTHON_VERSION | cut -d. -f1,2)


echo "Building custom Python distributions..."

# Build for current architecture (macOS or Linux)
if [ "$(uname -s)" = "Darwin" ]; then
    cd "$PYTHON_BUILD_STANDALONE_DIR" && \
    PYBUILD_PYTHON_VERSION="$PYTHON_VERSION" \
    python3 build-macos.py \
    --python "cpython-${PYTHON_VERSION_SHORT}" \
    --python-source "$CPYTHON_DIR" \
    --target-triple $([ "$(uname -m)" = "arm64" ] && echo "aarch64-apple-darwin" || echo "x86_64-apple-darwin") \
    --options pgo+lto
else
    cd "$PYTHON_BUILD_STANDALONE_DIR" && \
    PYBUILD_PYTHON_VERSION="$PYTHON_VERSION" \
    python3 build-linux.py \
    --python "cpython-${PYTHON_VERSION_SHORT}" \
    --python-source "$CPYTHON_DIR" \
    --target-triple $([ "$(uname -m)" = "aarch64" ] && echo "aarch64-unknown-linux-gnu" || echo "x86_64-unknown-linux-gnu") \
    --options pgo+lto
fi

echo "Python distributions built successfully" 