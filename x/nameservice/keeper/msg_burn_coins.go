package keeper

import (
	"context"

	cosmossdkerrors "cosmossdk.io/errors"
	nameservice "dysonprotocol.com/x/nameservice"
	nameservicev1 "dysonprotocol.com/x/nameservice/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// BurnCoins implements MsgServer.BurnCoins
func (k Keeper) BurnCoins(ctx context.Context, msg *nameservicev1.MsgBurnCoins) (*nameservicev1.MsgBurnCoinsResponse, error) {
	k.Logger.Info("BurnCoins: Processing", "owner", msg.Owner, "amount", msg.Amount)

	// Get SDK context from context.Context
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Verify the owner exists and is a valid address
	owner, err := sdk.AccAddressFromBech32(msg.Owner)
	if err != nil {
		k.Logger.Error("BurnCoins: Invalid owner address", "owner", msg.Owner, "error", err)
		return nil, cosmossdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid owner address: %s", msg.Owner)
	}

	// Check that all coins are of valid denom format for burning
	for _, coin := range msg.Amount {
		// Verify the sender owns the denom
		if err := k.VerifyDenomOwner(sdkCtx, coin.Denom, msg.Owner); err != nil {
			k.Logger.Error("BurnCoins: Invalid denom", "denom", coin.Denom, "owner", msg.Owner, "error", err)
			return nil, cosmossdkerrors.Wrapf(err, "cannot burn coin with denom %s, not owned by %s", coin.Denom, msg.Owner)
		}
	}

	// Burn the coins by sending them to the module account
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, owner, nameservice.ModuleName, msg.Amount); err != nil {
		k.Logger.Error("BurnCoins: Failed to transfer coins to module", "error", err)
		return nil, cosmossdkerrors.Wrap(err, "failed to transfer coins to module")
	}

	// Get module address and burn the coins
	moduleAddr := k.accountKeeper.GetModuleAddress(nameservice.ModuleName)
	if err := k.bankKeeper.BurnCoins(ctx, moduleAddr.String(), msg.Amount); err != nil {
		k.Logger.Error("BurnCoins: Failed to burn coins", "error", err)
		return nil, cosmossdkerrors.Wrap(err, "failed to burn coins")
	}

	k.Logger.Info("BurnCoins: Successfully burned coins", "amount", msg.Amount, "owner", msg.Owner)

	// Emit event
	if err := sdkCtx.EventManager().EmitTypedEvent(
		&nameservicev1.EventCoinsBurned{
			Amount: msg.Amount,
		},
	); err != nil {
		k.Logger.Error("BurnCoins: Failed to emit event", "error", err)
		return nil, cosmossdkerrors.Wrap(err, "failed to emit event")
	}

	return &nameservicev1.MsgBurnCoinsResponse{}, nil
}
