package keeper

import (
	"context"

	cosmossdkerrors "cosmossdk.io/errors"
	nameservicev1 "dysonprotocol.com/x/nameservice/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// SetDestination implements the MsgServer.SetDestination method
func (k Keeper) SetDestination(ctx context.Context, msg *nameservicev1.MsgSetDestination) (*nameservicev1.MsgSetDestinationResponse, error) {
	k.Logger.Info("SetDestination: Processing request", "name", msg.Name, "owner", msg.Owner, "destination", msg.Destination)

	// Get the name NFT
	nameNFT, found := k.nftKeeper.GetNFT(ctx, NamesClassID, msg.Name)
	if !found {
		k.Logger.Error("SetDestination: Name NFT not found", "name", msg.Name)
		return nil, cosmossdkerrors.Wrap(sdkerrors.ErrNotFound, "name not found")
	}

	// Get the owner of the NFT
	owner := k.nftKeeper.GetOwner(ctx, NamesClassID, msg.Name)
	ownerStr := owner.String()
	k.Logger.Info("SetDestination: Found name NFT", "name", msg.Name, "owner", ownerStr)

	// Verify owner
	if ownerStr != msg.Owner {
		k.Logger.Error("SetDestination: Unauthorized - not the owner", "name", msg.Name, "nft_owner", ownerStr, "msg_owner", msg.Owner)
		return nil, cosmossdkerrors.Wrap(sdkerrors.ErrUnauthorized, "only the owner can set the destination")
	}

	// Update the NFT
	nameNFT.Uri = msg.Destination
	if err := k.nftKeeper.Update(ctx, nameNFT); err != nil {
		k.Logger.Error("SetDestination: Failed to update NFT", "name", msg.Name, "error", err)
		return nil, cosmossdkerrors.Wrap(err, "failed to update NFT")
	}

	k.Logger.Info("SetDestination: Successfully updated name NFT", "name", msg.Name, "destination", nameNFT.Uri)

	// Emit event
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(
		&nameservicev1.EventNameDestinationSet{
			Name:        msg.Name,
			Destination: nameNFT.Uri,
		})
	if err != nil {
		k.Logger.Error("failed to emit name destination set event", "error", err)
		return nil, cosmossdkerrors.Wrap(err, "failed to emit name destination set event")
	}

	return &nameservicev1.MsgSetDestinationResponse{}, nil
}
