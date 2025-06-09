package keeper

import (
	"context"
	"regexp"

	cosmossdkerrors "cosmossdk.io/errors"
	nameservice "dysonprotocol.com/x/nameservice"
	nameservicev1 "dysonprotocol.com/x/nameservice/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// MintCoins implements the MsgServer.MintCoins method
func (k Keeper) MintCoins(ctx context.Context, msg *nameservicev1.MsgMintCoins) (*nameservicev1.MsgMintCoinsResponse, error) {
	// 1. Validate owner address
	ownerAddr, err := sdk.AccAddressFromBech32(msg.Owner)
	if err != nil {
		return nil, cosmossdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid owner address: %s", msg.Owner)
	}

	// 2. Validate coins
	if msg.Amount.Empty() {
		return nil, cosmossdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "no coins to mint")
	}

	// Regex for valid coin denom format - removed the + character
	validDenomPattern := regexp.MustCompile(`^[A-Za-z0-9.\-_]+\.dys(?:/[0-9A-Za-z:\-_]+)*$`)

	// 3. For each coin, verify that the owner owns the root name
	for _, coin := range msg.Amount {
		// Validate coin denom with regex
		if !validDenomPattern.MatchString(coin.Denom) {
			return nil, cosmossdkerrors.Wrapf(
				sdkerrors.ErrInvalidRequest,
				"invalid denom format, must match pattern: %s",
				validDenomPattern.String(),
			)
		}

		// Verify the sender owns the denom
		if err := k.VerifyDenomOwner(sdk.UnwrapSDKContext(ctx), coin.Denom, msg.Owner); err != nil {
			return nil, err
		}
	}

	// 4. Mint the coins to the module account
	err = k.bankKeeper.MintCoins(ctx, nameservice.ModuleName, msg.Amount)
	if err != nil {
		return nil, cosmossdkerrors.Wrap(err, "failed to mint coins")
	}

	// 5. Send the minted coins from the module to the owner
	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, nameservice.ModuleName, ownerAddr, msg.Amount)
	if err != nil {
		return nil, cosmossdkerrors.Wrap(err, "failed to send minted coins to owner")
	}

	// 6. Emit event
	if evErr := sdk.UnwrapSDKContext(ctx).EventManager().EmitTypedEvent(
		&nameservicev1.EventCoinsMinted{
			Amount: msg.Amount,
		},
	); evErr != nil {
		k.Logger.Error("failed to emit coins minted event", "error", evErr)
		return nil, cosmossdkerrors.Wrap(evErr, "failed to emit coins minted event")
	}

	k.Logger.Info("MintCoins: Successfully minted coins", "owner", msg.Owner, "amount", msg.Amount.String())

	return &nameservicev1.MsgMintCoinsResponse{}, nil
}
