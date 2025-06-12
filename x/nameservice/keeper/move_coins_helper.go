package keeper

import (
	"context"

	cosmossdkerrors "cosmossdk.io/errors"
	nameservice "dysonprotocol.com/x/nameservice"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// validateMoveCoins ensures total input == total output and no negative coins
func validateMoveCoins(inputs []banktypes.Input, outputs []banktypes.Output) error {
	if len(inputs) == 0 || len(outputs) == 0 {
		return sdkerrors.ErrInvalidRequest.Wrapf("inputs (%d) and outputs (%d) cannot be empty", len(inputs), len(outputs))
	}

	totalIn := sdk.NewCoins()
	for idx, in := range inputs {
		if in.Coins.IsAnyNegative() {
			return sdkerrors.ErrInvalidRequest.Wrapf("input[%d] has negative coins: %s", idx, in.Coins.String())
		}
		totalIn = totalIn.Add(in.Coins...)
	}

	totalOut := sdk.NewCoins()
	for idx, out := range outputs {
		if out.Coins.IsAnyNegative() {
			return sdkerrors.ErrInvalidRequest.Wrapf("output[%d] has negative coins: %s", idx, out.Coins.String())
		}
		totalOut = totalOut.Add(out.Coins...)
	}

	if !totalIn.Equal(totalOut) {
		return sdkerrors.ErrInvalidRequest.Wrapf("total input %s does not equal total output %s", totalIn.String(), totalOut.String())
	}
	return nil
}

// moveCoins executes the multi-send via the nameservice module account
func (k Keeper) moveCoins(ctx context.Context, inputs []banktypes.Input, outputs []banktypes.Output) error {
	if err := validateMoveCoins(inputs, outputs); err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// first transfer all inputs to module account
	for _, in := range inputs {
		fromAddr, err := sdk.AccAddressFromBech32(in.Address)
		if err != nil {
			return cosmossdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid input address: %s", in.Address)
		}
		if err := k.bankKeeper.SendCoinsFromAccountToModule(sdkCtx, fromAddr, nameservice.ModuleName, in.Coins); err != nil {
			return err
		}
	}

	// then distribute to outputs
	for _, out := range outputs {
		toAddr, err := sdk.AccAddressFromBech32(out.Address)
		if err != nil {
			return cosmossdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid output address: %s", out.Address)
		}
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(sdkCtx, nameservice.ModuleName, toAddr, out.Coins); err != nil {
			return err
		}
	}

	return nil
}
