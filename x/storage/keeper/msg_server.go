package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	storagetypes "dysonprotocol.com/x/storage/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// We assume your keeper implements storagetypes.MsgServer
var _ storagetypes.MsgServer = Keeper{}

func (k Keeper) StorageSet(ctx context.Context, msg *storagetypes.MsgStorageSet) (*storagetypes.MsgStorageSetResponse, error) {
	// Validate the owner address is properly formatted
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return nil, err
	}

	// Create the key directly using strings
	key := collections.Join(msg.Owner, msg.Index)

	entry := storagetypes.Storage{
		Owner: msg.Owner,
		Data:  msg.Data,
		Index: msg.Index,
	}

	if err := k.StorageMap.Set(ctx, key, entry); err != nil {
		return nil, err
	}

	// Emit the StorageUpdated event
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	event := storagetypes.EventStorageUpdated{
		Address: msg.Owner,
		Index:   msg.Index,
	}
	fmt.Println("StorageSet event", event)
	if err := sdkCtx.EventManager().EmitTypedEvent(&event); err != nil {
		return nil, err
	}

	return &storagetypes.MsgStorageSetResponse{}, nil
}

func (k Keeper) StorageDelete(ctx context.Context, msg *storagetypes.MsgStorageDelete) (*storagetypes.MsgStorageDeleteResponse, error) {
	// Validate the owner address is properly formatted
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return nil, err
	}

	// Track how many entries were actually deleted
	var deletedCount uint64

	// Track the indexes that were deleted
	var deletedIndexes []string

	// Delete each index in the list
	for _, index := range msg.Indexes {
		// Create the key for this index
		key := collections.Join(msg.Owner, index)

		// Check if the entry exists first
		exists, err := k.StorageMap.Has(ctx, key)
		if err != nil {
			return nil, err
		}

		// Only try to delete if it exists
		if exists {
			if err := k.StorageMap.Remove(ctx, key); err != nil {
				return nil, err
			}
			deletedCount++
			deletedIndexes = append(deletedIndexes, index)
		}
	}

	if len(deletedIndexes) == 0 {
		return nil, errorsmod.Wrap(sdkerrors.ErrNotFound, "no entries were deleted")
	}

	// Emit the StorageDelete event
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	event := storagetypes.EventStorageDelete{
		Owner:          msg.Owner,
		DeletedIndexes: deletedIndexes,
	}
	if err := sdkCtx.EventManager().EmitTypedEvent(&event); err != nil {
		return nil, err
	}
	return &storagetypes.MsgStorageDeleteResponse{
		DeletedIndexes: deletedIndexes,
	}, nil
}
