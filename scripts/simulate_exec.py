#!/usr/bin/env python3

import argparse
import json
import os
import subprocess
import sys
import tempfile
from pathlib import Path

# Import parse_exec_script_tx module directly
sys.path.append(str(Path(__file__).parent))
import parse_exec_script_tx

def generate_unsigned_tx(args,  script_content):
    """Generate an unsigned transaction JSON."""
    cmd = ["dysond", "tx", "script", "exec"]
    
    # Add the required address parameters
    cmd.extend(["--from", args.from_address])
    cmd.extend(["--script-address", args.script_address])
    
    # Add the script content
    if script_content:
        cmd.extend(["--extra-code", script_content])
    
    # Add function name and arguments if provided
    if args.function_name:
        cmd.extend(["--function-name", args.function_name])
    
    if args.args:
        cmd.extend(["--args", args.args])
    
    if args.kwargs:
        cmd.extend(["--kwargs", args.kwargs])
    
    if args.attached_messages:
        for msg in args.attached_messages:
            cmd.extend(["--attached-message", msg])

    # Generate only
    cmd.append("--generate-only")
    
    # Run the command
    result = subprocess.run(cmd, capture_output=True, text=True)
    
    if result.returncode != 0:
        print(f"Error generating unsigned transaction: {result.stderr}")
        sys.exit(1)
    
    return json.loads(result.stdout)

def simulate_transaction(unsigned_tx, from_address, extra_args):
    """Simulate the transaction and return the result."""
    # Write the unsigned transaction to a temporary file
    with tempfile.NamedTemporaryFile(mode='w', delete=False) as temp_file:
        json.dump(unsigned_tx, temp_file)
        temp_file_path = temp_file.name
    
    try:
        # Run the simulate command
        simulate_cmd = ["dysond", "tx", "simulate", temp_file_path, "--from", from_address, "-o", "json"]
        simulate_cmd.extend(extra_args)
        result = subprocess.run(simulate_cmd, capture_output=True, text=True)
        
        if result.returncode != 0:
            print(f"Error simulating transaction: {result.stderr}")
            sys.exit(1)
        
        # Parse the result
        simulation_result = json.loads(result.stdout)
        return simulation_result
    finally:
        # Clean up the temporary file
        os.unlink(temp_file_path)

def parse_simulation_result(simulation_result):
    """Parse the simulation result using parse_exec_script_tx module directly."""
    # Get the result part of the simulation
    tx_response = simulation_result.get("result", {})
    
    # Parse it using the parse_tx_response function
    parsed_result = parse_exec_script_tx.parse_tx_response(tx_response)
    
    return json.dumps(parsed_result, indent=2)

def main():
    epilog_text = """Examples:
    
# Example 1 - Execute a script file:
./scripts/simulate_exec.py ./path/to/script.py --from alice --script-address $ALICE_ADDRESS --function-name my_func --args '["arg1", "arg2"]' -- --gas-adjustment 1.4

# Example 2 - Execute code from HEREDOC:
cat << 'EOF' | ./scripts/simulate_exec.py - --from alice --script-address $ALICE_ADDRESS --function-name test --args "[1,2,3]"
from dys import get_script_address, get_executor_address

def test(*args):
    print("args: ", args) 
    print(f"Executor: {get_executor_address()}")
    print(f"Script: {get_script_address()}")
    return "Success"
EOF
"""
    
    parser = argparse.ArgumentParser(
        description="Simulate executing a Dyson script without submitting the transaction to the blockchain.",
        epilog=epilog_text,
        formatter_class=argparse.RawDescriptionHelpFormatter
    )
    
    # parser.add_argument("-s", "--script-path", type=argparse.FileType('r'), required=False, 
    #                     help="Path to the Dyson script file to execute (use '-' for stdin)")
    parser.add_argument("script_filename", metavar="SCRIPT_FILE_OR_STDIN", nargs='?', default=None,
                        help="Optional path to the Dyson script file to execute (use '-' for stdin). "
                             "If not provided, no --extra-code is used (e.g. for calling an existing script by address).")
    parser.add_argument("--from", dest="from_address", required=True, help="The address executing the script")
    parser.add_argument("--script-address", required=True, help="The address of the script to execute")
    parser.add_argument("--function-name", help="The name of the function to call")
    parser.add_argument("--args", help="JSON array of arguments to pass to the function")
    parser.add_argument("--kwargs", help="JSON object of keyword arguments to pass to the function")
    
    # Attached-messages can be used multiple times in the command line to add multiple messages
    parser.add_argument("--attached-message", dest="attached_messages", help="attached messages to include in the transaction", action="append")
    args, extra_args = parser.parse_known_args()
    

    # Read the script file
    script_content = None
    if args.script_filename:
        if args.script_filename == '-':
            script_content = sys.stdin.read()
        else:
            with open(args.script_filename, 'r') as f:
                script_content = f.read()
    
    # Generate an unsigned transaction
    unsigned_tx = generate_unsigned_tx(args, script_content)
    
    # Simulate the transaction
    simulation_result = simulate_transaction(unsigned_tx, args.from_address, extra_args)
    
    # Parse and print the simulation result
    parsed_result = parse_simulation_result(simulation_result)
    print(parsed_result)

if __name__ == "__main__":
    main()
