package keeper

import (
	"context"

	cosmossdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	nameservicev1 "dysonprotocol.com/x/nameservice/types"
)

// SetNFTClassAlwaysListed handles a MsgSetNFTClassAlwaysListed message
func (k Keeper) SetNFTClassAlwaysListed(ctx context.Context, msg *nameservicev1.MsgSetNFTClassAlwaysListed) (*nameservicev1.MsgSetNFTClassAlwaysListedResponse, error) {
	// Verify authorization: only the owner of the root name (class owner) can set always_listed
	if err := k.verifyClassIDOwner(ctx, msg.ClassId, msg.Owner); err != nil {
		return nil, cosmossdkerrors.Wrapf(
			sdkerrors.ErrUnauthorized,
			"only the owner [%s] of the class root name can set always_listed for this NFT class",
			msg.Owner,
		)
	}

	// Get current NFT class data
	classData, err := k.GetNFTClassData(ctx, msg.ClassId)
	if err != nil {
		k.Logger.Error("SetNFTClassAlwaysListed: Failed to get NFT class data",
			"class_id", msg.ClassId,
			"error", err)
		return nil, cosmossdkerrors.Wrapf(
			err,
			"NFT class not found: %s",
			msg.ClassId,
		)
	}

	// Update the always_listed field
	classData.AlwaysListed = msg.AlwaysListed

	// Set the updated NFT class data
	if err := k.SetNFTClassData(ctx, msg.ClassId, classData); err != nil {
		return nil, cosmossdkerrors.Wrapf(err, "failed to update NFT class data for class %s", msg.ClassId)
	}

	// Emit event
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if evErr := sdkCtx.EventManager().EmitTypedEvent(
		&nameservicev1.EventNFTClassAlwaysListedUpdated{
			ClassId: msg.ClassId,
		},
	); evErr != nil {
		k.Logger.Error("failed to emit NFT class always listed updated event", "error", evErr)
		return nil, cosmossdkerrors.Wrap(evErr, "failed to emit NFT class always listed updated event")
	}

	k.Logger.Info("Successfully updated NFT class always listed",
		"class_id", msg.ClassId,
		"owner", msg.Owner,
		"always_listed", msg.AlwaysListed)

	return &nameservicev1.MsgSetNFTClassAlwaysListedResponse{}, nil
}
