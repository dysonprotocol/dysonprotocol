package keeper

import (
	"context"
	"fmt"

	cosmossdkerrors "cosmossdk.io/errors"
	nameservicev1 "dysonprotocol.com/x/nameservice/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// SetDenomMetadata updates the denom metadata in the bank module
func (k Keeper) SetDenomMetadata(ctx context.Context, msg *nameservicev1.MsgSetDenomMetadata) (*nameservicev1.MsgSetDenomMetadataResponse, error) {
	// Log the beginning of the function with detailed information
	fmt.Printf("SetDenomMetadata called with authority: %s, denom: %s\n", msg.Authority, msg.Metadata.Base)
	k.Logger.Info("SetDenomMetadata detailed request info",
		"authority", msg.Authority,
		"base", msg.Metadata.Base,
		"display", msg.Metadata.Display,
		"description", msg.Metadata.Description,
		"symbol", msg.Metadata.Symbol)

	// Check if the authority is authorized to set metadata
	// Only the governance module account (or other authorized account) can set denom metadata
	if msg.Authority != k.GetAuthority() {
		k.Logger.Error("Unauthorized authority",
			"expected", k.GetAuthority(),
			"got", msg.Authority)
		return nil, cosmossdkerrors.Wrapf(
			sdkerrors.ErrUnauthorized,
			"invalid authority; expected %s, got %s",
			k.GetAuthority(),
			msg.Authority,
		)
	}

	// Validate the metadata
	if err := msg.Metadata.Validate(); err != nil {
		k.Logger.Error("Metadata validation failed", "error", err)
		return nil, cosmossdkerrors.Wrap(err, "invalid metadata")
	}

	// Set the denom metadata
	k.bankKeeper.SetDenomMetaData(ctx, msg.Metadata)

	// Emit event
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(&nameservicev1.EventDenomMetadataSet{
		Denom: msg.Metadata.Base,
	})
	if err != nil {
		k.Logger.Error("failed to emit event", "error", err)
		return nil, cosmossdkerrors.Wrap(err, "failed to emit event")
	}

	k.Logger.Info("SetDenomMetadata successful",
		"authority", msg.Authority,
		"denom", msg.Metadata.Base)

	fmt.Printf("SetDenomMetadata completed successfully\n")
	return &nameservicev1.MsgSetDenomMetadataResponse{}, nil
}
