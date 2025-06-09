package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	storagetypes "dysonprotocol.com/x/storage/types"
)

// NewTxCmd returns a root CLI command handler for all x/storage transaction commands.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        "storage",
		Short:                      "Storage subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(NewStorageSetCmd())
	txCmd.AddCommand(NewStorageDeleteCmd())

	return txCmd
}

// NewStorageSetCmd returns the CLI command handler for setting/updating a storage entry.
func NewStorageSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set --index <index> [--data <data> | --data-path <path to data file>]",
		Short: "Set or update a storage entry with the specified index and data",
		Long: `Set or update a storage entry with the specified index and data. The owner is automatically set to the transaction signer.

You must provide either --data or --data-path but not both.
An empty data string (--data "") is allowed.

Examples:
  # Set using a file
  $ dysond tx storage set --index "config/settings" --data-path ./settings.json --from myaccount

  # Set using inline data
  $ dysond tx storage set --index "user/profile" --data '{"name": "Alice", "age": 30}' --from myaccount
  
  # Set with empty data
  $ dysond tx storage set --index "placeholder" --data "" --from myaccount`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// Get the required index flag
			index, err := cmd.Flags().GetString("index")
			if err != nil {
				return err
			}
			if index == "" {
				return errors.New("--index flag is required")
			}

			// Check which data flags are explicitly set
			dataProvided := cmd.Flags().Changed("data")
			dataPathProvided := cmd.Flags().Changed("data-path")

			// Validate both are not provided
			if dataProvided && dataPathProvided {
				return errors.New("cannot provide both --data and --data-path, use only one")
			}

			// Validate at least one is provided
			if !dataProvided && !dataPathProvided {
				return errors.New("either --data or --data-path must be provided")
			}

			// Get data content based on which flag was used
			var dataContent string
			if dataProvided {
				data, err := cmd.Flags().GetString("data")
				if err != nil {
					return err
				}
				dataContent = data // This can be empty, which is allowed
			} else {
				dataPath, err := cmd.Flags().GetString("data-path")
				if err != nil {
					return err
				}
				// Read file contents
				dataBytes, err := os.ReadFile(dataPath)
				if err != nil {
					return fmt.Errorf("failed to read file %s: %w", dataPath, err)
				}
				dataContent = string(dataBytes)
			}

			// Use the sender address as the owner
			owner := clientCtx.GetFromAddress().String()

			msg := &storagetypes.MsgStorageSet{
				Owner: owner,
				Index: index,
				Data:  dataContent,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("index", "", "The unique identifier/key for the storage entry (required)")
	cmd.Flags().String("data", "", "The data to store as a string")
	cmd.Flags().String("data-path", "", "Path to the data file")

	// Mark index as required
	_ = cmd.MarkFlagRequired("index")

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewStorageDeleteCmd returns the CLI command handler for deleting storage entries.
func NewStorageDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete --indexes <indexes>",
		Short: "Delete one or more storage entries with the specified indexes",
		Long: `Delete one or more storage entries with the specified indexes. Only existing entries will be deleted.
The owner is automatically set to the transaction signer.

Examples:
  # Delete a single entry
  $ dysond tx storage delete --indexes "config/settings" --from myaccount

  # Delete multiple entries
  $ dysond tx storage delete --indexes "user/profile,temp/data,cache/item" --from myaccount`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// Get the required indexes flag
			indexes, err := cmd.Flags().GetStringSlice("indexes")
			if err != nil {
				return err
			}

			if len(indexes) == 0 {
				return errors.New("at least one index must be provided via --indexes")
			}

			// Use the sender address as the owner
			owner := clientCtx.GetFromAddress().String()

			msg := &storagetypes.MsgStorageDelete{
				Owner:   owner,
				Indexes: indexes,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().StringSlice("indexes", []string{}, "Comma-separated list of indexes to delete (required)")

	// Mark indexes as required
	_ = cmd.MarkFlagRequired("indexes")

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
