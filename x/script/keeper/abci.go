package keeper

import (
	"context"
	"fmt"

	scripttypes "dysonprotocol.com/x/script/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BeginBlocker enforces nodes to keep the required historical blocks
// It queries the module parameters to determine the maximum historical blocks
// that should be available and panics if the required blocks are not accessible
func (k Keeper) BeginBlocker(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get module parameters to determine the maximum historical blocks
	params := k.GetParams(ctx)

	return k.validateHistoricalBlocks(sdkCtx, params)
}

// validateHistoricalBlocks checks if the required historical blocks are accessible
func (k Keeper) validateHistoricalBlocks(ctx sdk.Context, params scripttypes.Params) error {
	currentHeight := ctx.BlockHeight()

	// Skip validation for genesis block and very early blocks
	if currentHeight <= 1 {
		return nil
	}

	// Skip validation if maxRelativeHistoricalBlocks is disabled (0 or negative)
	if params.MaxRelativeHistoricalBlocks < 1 {
		return nil
	}

	// Calculate the oldest block we should be able to query using the AbsoluteHistoricalBlockCutoff
	// The oldest required height is the maximum of:
	// 1. current_height - max_relative_historical_blocks
	// 2. absolute_historical_block_cutoff (the absolute minimum we require)
	oldestFromBlocks := currentHeight - params.MaxRelativeHistoricalBlocks
	oldestRequiredHeight := params.AbsoluteHistoricalBlockCutoff
	if oldestFromBlocks > oldestRequiredHeight {
		oldestRequiredHeight = oldestFromBlocks
	}

	// Ensure we never try to query height 0 or negative heights
	if oldestRequiredHeight < 1 {
		oldestRequiredHeight = 1
	}

	// Try to create a query context for the oldest required height
	// This will fail if the node doesn't have the historical blocks
	_, err := k.App.CreateQueryContextWithCheckHeader(oldestRequiredHeight, false, false)
	if err != nil {
		// This is a critical error - the node doesn't have required historical blocks
		// Log the error instead of panicking
		k.Logger(ctx).Error(
			"CRITICAL: Node missing required historical blocks - this will cause a consensus error",
			"current_height", currentHeight,
			"oldest_required_height", oldestRequiredHeight,
			"max_relative_historical_blocks", params.MaxRelativeHistoricalBlocks,
			"absolute_historical_block_cutoff", params.AbsoluteHistoricalBlockCutoff,
			"error", err,
			"message", fmt.Sprintf(
				"WARNING: This error will cause a consensus failure. "+
					"Please ensure your node is configured to retain at least %d historical blocks. "+
					"Check your app.toml `min-retain-blocks=%d`",
				params.MaxRelativeHistoricalBlocks,
				params.MaxRelativeHistoricalBlocks+1,
			),
		)
		return err
	}

	// Log successful validation for monitoring
	k.Logger(ctx).Info(
		"Historical block retention validation passed",
		"current_height", currentHeight,
		"oldest_required_height", oldestRequiredHeight,
		"max_relative_historical_blocks", params.MaxRelativeHistoricalBlocks,
		"absolute_historical_block_cutoff", params.AbsoluteHistoricalBlockCutoff,
	)

	return nil
}

func (k Keeper) EndBlocker(ctx context.Context) error {
	return nil
}
