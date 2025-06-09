#!/usr/bin/env python3

import json
import sys

def try_parse_json_maybe_trailing(data: str):
    """
    Tries to parse JSON from the string, ignoring any trailing text after the last '}'.
    Returns None if parsing fails or there's no final '}'.
    """
    data = data.strip()
    if not data:
        return None
    opening_idx = data.find("{")
    if opening_idx == -1:
        return None
    closing_idx = data.rfind("}")
    if closing_idx == -1:
        return None
    candidate = data[opening_idx: closing_idx + 1]
    try:
        return json.loads(candidate)
    except json.JSONDecodeError:
        return None

def parse_script_event(events):
    """
    Looks for a 'dysonprotocol.script.v1.EventExecScript' event and
    tries to parse its 'response' attribute as JSON, ignoring trailing text.
    Returns the parsed dict, or None if not found / parsing fails.
    """
    for ev in events:
        if ev.get("type") == "dysonprotocol.script.v1.EventExecScript":
            for attr in ev.get("attributes", []):
                if attr.get("key") == "response":
                    parsed = try_parse_json_maybe_trailing(attr.get("value", ""))
                    if parsed is not None:
                        return parsed
    return None

def parse_nested_json_strings(obj):
    """
    Recursively parse any string values that are JSON strings.
    """
    if isinstance(obj, dict):
        for key, value in obj.items():
            if isinstance(value, str):
                try:
                    obj[key] = json.loads(value)
                except json.JSONDecodeError:
                    pass  # Keep as string if parsing fails
            elif isinstance(value, dict):
                obj[key] = parse_nested_json_strings(value)
            elif isinstance(value, list):
                obj[key] = [parse_nested_json_strings(item) if isinstance(item, (dict, list)) else item for item in value]
    elif isinstance(obj, list):
        return [parse_nested_json_strings(item) if isinstance(item, (dict, list)) else item for item in obj]
    return obj

def parse_tx_response(tx_response: dict) -> dict:
    """
    Returns a unified structure with fields:
      {
        "code": <int>,
        "script_result": <dict or str or null>,
        "raw_log": <str>,
        "events": <list of dicts>
      }
    """
    code = tx_response.get("code", 0)
    raw_log = tx_response.get("raw_log", "")
    events = tx_response.get("events", [])

    # 1) Attempt to parse raw_log
    script_result = try_parse_json_maybe_trailing(raw_log)

    # 2) If raw_log isn't parsed, try parse the event
    if script_result is None:
        script_result = parse_script_event(events)
    
    # 3) Parse any nested JSON strings if a result was found
    if isinstance(script_result, (dict, list)):
        script_result = parse_nested_json_strings(script_result)

    return {
        "code": code,
        "script_result": script_result,
        "raw_log": raw_log,
        "events": events
    }

def main():
    """
    Reads a JSON object from stdin or from a file specified as an argument,
    parses it, and prints out the unified result.
    
    Usage:
        python3 script.py < txresponse.json
        python3 script.py txresponse.json
    """
    # Check if a file was specified as an argument
    if len(sys.argv) > 1:
        try:
            with open(sys.argv[1], 'r') as file:
                raw_data = file.read()
        except FileNotFoundError:
            print(f"Error: File '{sys.argv[1]}' not found.")
            sys.exit(1)
    else:
        # Read from stdin if no file was specified
        raw_data = sys.stdin.read()
    
    if not raw_data.strip():
        print("Error: No input data provided.")
        sys.exit(1)
    
    try:
        tx_response = json.loads(raw_data)
        result = parse_tx_response(tx_response)
        print(json.dumps(result, indent=2))
    except json.JSONDecodeError as e:
        print(f"Error parsing JSON: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main() 