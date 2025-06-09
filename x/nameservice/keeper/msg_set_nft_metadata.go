package keeper

import (
	"context"

	cosmossdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	nameservicev1 "dysonprotocol.com/x/nameservice/types"
)

// SetNFTMetadata handles a MsgSetNFTMetadata message
func (k Keeper) SetNFTMetadata(ctx context.Context, msg *nameservicev1.MsgSetNFTMetadata) (*nameservicev1.MsgSetNFTMetadataResponse, error) {
	// Verify authorization: only the owner of the root name (class owner) can set metadata
	if err := k.verifyClassIDOwner(ctx, msg.ClassId, msg.Owner); err != nil {
		return nil, cosmossdkerrors.Wrapf(
			err,
			"Error verifying class ID owner when setting NFT metadata",
		)
	}

	// Get current NFT data
	nftData, err := k.GetNFTData(ctx, msg.ClassId, msg.NftId)
	if err != nil {
		k.Logger.Error("SetNFTMetadata: Failed to get NFT data",
			"class_id", msg.ClassId,
			"nft_id", msg.NftId,
			"error", err)
		return nil, cosmossdkerrors.Wrapf(
			err,
			"NFT not found class %s, id %s",
			msg.ClassId,
			msg.NftId,
		)
	}
	k.Logger.Info("SetNFTMetadata: setting metadata", "class_id", msg.ClassId, "nft_id", msg.NftId, "nftData", nftData)
	// Update the metadata field
	nftData.Metadata = msg.Metadata

	k.Logger.Info("SetNFTMetadata: Validate Basic")
	// Validate the updated NFT data
	if err := nftData.ValidateBasic(); err != nil {
		return nil, cosmossdkerrors.Wrapf(err, "invalid NFT data for class %s, id %s", msg.ClassId, msg.NftId)
	}

	k.Logger.Info("SetNFTMetadata: SetNFTData")
	// Set the updated NFT data
	if err := k.SetNFTData(ctx, msg.ClassId, msg.NftId, nftData); err != nil {
		return nil, cosmossdkerrors.Wrapf(err, "failed to update NFT data for class %s, id %s", msg.ClassId, msg.NftId)
	}

	// Emit event
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if evErr := sdkCtx.EventManager().EmitTypedEvent(
		&nameservicev1.EventNFTMetadataUpdated{
			ClassId: msg.ClassId,
			NftId:   msg.NftId,
		},
	); evErr != nil {
		k.Logger.Error("failed to emit NFT metadata updated event", "error", evErr)
		return nil, cosmossdkerrors.Wrap(evErr, "failed to emit NFT metadata updated event")
	}

	k.Logger.Info("Successfully updated NFT metadata",
		"class_id", msg.ClassId,
		"nft_id", msg.NftId,
		"owner", msg.Owner)

	return &nameservicev1.MsgSetNFTMetadataResponse{}, nil
}
