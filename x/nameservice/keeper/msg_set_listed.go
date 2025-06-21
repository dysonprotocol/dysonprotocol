package keeper

import (
	"context"

	cosmossdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	nameservicev1 "dysonprotocol.com/x/nameservice/types"
)

// SetListed handles a MsgSetListed message
func (k Keeper) SetListed(ctx context.Context, msg *nameservicev1.MsgSetListed) (*nameservicev1.MsgSetListedResponse, error) {
	// Get the owner of the NFT to verify authorization
	owner := k.nftKeeper.GetOwner(ctx, msg.NftClassId, msg.NftId)
	if owner.Empty() {
		k.Logger.Error("SetListed: NFT not found",
			"class_id", msg.NftClassId,
			"nft_id", msg.NftId)
		return nil, cosmossdkerrors.Wrapf(
			sdkerrors.ErrNotFound,
			"NFT not found: class_id=%s, nft_id=%s",
			msg.NftClassId,
			msg.NftId,
		)
	}

	// Verify authorization: only the NFT owner can set the listed status
	if owner.String() != msg.NftOwner {
		return nil, cosmossdkerrors.Wrapf(
			sdkerrors.ErrUnauthorized,
			"only the NFT owner [%s] can set listed status, got [%s]",
			owner.String(),
			msg.NftOwner,
		)
	}

	// Get current NFT data to update the listed field
	nftData, err := k.GetNFTData(ctx, msg.NftClassId, msg.NftId)
	if err != nil {
		k.Logger.Error("SetListed: Failed to get NFT data",
			"class_id", msg.NftClassId,
			"nft_id", msg.NftId,
			"error", err)
		return nil, cosmossdkerrors.Wrapf(
			err,
			"NFT not found class %s, id %s",
			msg.NftClassId,
			msg.NftId,
		)
	}

	// Update the listed field
	nftData.Listed = msg.Listed

	// Validate the updated NFT data
	if err := nftData.ValidateBasic(); err != nil {
		return nil, cosmossdkerrors.Wrapf(err, "invalid NFT data for class %s, id %s", msg.NftClassId, msg.NftId)
	}

	// Set the updated NFT data
	if err := k.SetNFTData(ctx, msg.NftClassId, msg.NftId, nftData); err != nil {
		return nil, cosmossdkerrors.Wrapf(err, "failed to update NFT data for class %s, id %s", msg.NftClassId, msg.NftId)
	}

	// Emit event
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if evErr := sdkCtx.EventManager().EmitTypedEvent(
		&nameservicev1.EventNFTListedUpdated{
			ClassId: msg.NftClassId,
			NftId:   msg.NftId,
			Listed:  msg.Listed,
		},
	); evErr != nil {
		k.Logger.Error("failed to emit NFT listed updated event", "error", evErr)
		return nil, cosmossdkerrors.Wrap(evErr, "failed to emit NFT listed updated event")
	}

	k.Logger.Info("Successfully updated NFT listed status",
		"class_id", msg.NftClassId,
		"nft_id", msg.NftId,
		"owner", msg.NftOwner,
		"listed", msg.Listed)

	return &nameservicev1.MsgSetListedResponse{}, nil
}
