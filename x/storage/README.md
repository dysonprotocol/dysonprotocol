# Storage CLI Usage Examples

The storage module provides enhanced CLI commands for managing key-value storage on the blockchain.

## Enhanced `set` Command

The `set` command supports both inline data and file input:

### Using `--data` for inline content:
```bash
# Set with inline JSON data
dysond tx storage set --index "user/profile" --data '{"name": "Alice", "age": 30}' --from myaccount

# Set with plain text
dysond tx storage set --index "config/message" --data "Hello World" --from myaccount

# Set with empty data
dysond tx storage set --index "placeholder" --data "" --from myaccount
```

### Using `--data-path` for file input:
```bash
# Set using a JSON file
dysond tx storage set --index "config/settings" --data-path ./settings.json --from myaccount

# Set using any text file
dysond tx storage set --index "docs/readme" --data-path ./README.md --from myaccount
```

## Rules and Validation

- **Mutually Exclusive**: You cannot use both `--data` and `--data-path` together
- **One Required**: You must provide exactly one of the two flags
- **File Reading**: `--data-path` reads the entire file content as a string
- **No Content Validation**: Any string content is accepted (plain text, JSON, XML, etc.)

## Legacy Commands

For backward compatibility, the original auto-generated commands are still available:

```bash
# Original basic set command (only supports --data flag)
dysond tx storage storage-set --index "key" --data "value" --from myaccount

# Delete command
dysond tx storage delete --indexes "key1,key2,key3" --from myaccount
```

## Query Examples

```bash
# Query a specific storage entry
dysond query storage get <owner_address> --index "user/profile"

# Query with JSON output
dysond query storage get <owner_address> --index "config/settings" --output json

# List all storage entries for an owner
dysond query storage list <owner_address>

# List storage entries with a specific prefix
dysond query storage list <owner_address> --index-prefix "config/"

# List with pagination
dysond query storage list <owner_address> --limit 10 --offset 20

# List in reverse order
dysond query storage list <owner_address> --reverse
```

## Old Usage Examples (for reference)

```