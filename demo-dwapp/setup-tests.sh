#!/bin/bash
set -e

echo "🚀 Setting up Frontend Testing for DysonProtocol Demo DApp"
echo "========================================================="

# Install Playwright and dependencies
echo "📦 Installing Python test dependencies..."
pip install playwright pytest pytest-asyncio

# Install Playwright browsers
echo "🌐 Installing Playwright browsers..."
playwright install chromium

# Create test reports directory
echo "📁 Creating test directories..."
mkdir -p ../test-reports/screenshots

echo "✅ Setup complete!"
echo ""
echo "🧪 To run frontend tests:"
echo "   cd .."
echo "   pytest tests/test_demo_dwapp.py -v"
echo ""
echo "🧪 To run specific test categories:"
echo "   pytest tests/test_demo_dwapp.py -m frontend -v"
echo "   pytest tests/test_demo_dwapp.py -m visual -v"
echo ""
echo "🧪 To run with HTML report:"
echo "   pytest tests/test_demo_dwapp.py --html=test-reports/demo-dwapp-report.html -v" 