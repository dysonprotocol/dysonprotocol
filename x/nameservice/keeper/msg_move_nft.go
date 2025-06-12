package keeper

import (
	"context"

	cosmossdkerrors "cosmossdk.io/errors"
	nameservicev1 "dysonprotocol.com/x/nameservice/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// MoveNft transfers an NFT if signer owns the class and both accounts are non-module
func (k Keeper) MoveNft(ctx context.Context, msg *nameservicev1.MsgMoveNft) (*nameservicev1.MsgMoveNftResponse, error) {
	// validate addresses
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return nil, cosmossdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid owner address: %s", msg.Owner)
	}
	fromAddr, err := sdk.AccAddressFromBech32(msg.FromAddress)
	if err != nil {
		return nil, cosmossdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid from address: %s", msg.FromAddress)
	}
	toAddr, err := sdk.AccAddressFromBech32(msg.ToAddress)
	if err != nil {
		return nil, cosmossdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid to address: %s", msg.ToAddress)
	}

	if fromAcc := k.accountKeeper.GetAccount(sdk.UnwrapSDKContext(ctx), fromAddr); fromAcc != nil {
		if _, ok := fromAcc.(sdk.ModuleAccountI); ok {
			return nil, cosmossdkerrors.Wrap(sdkerrors.ErrInvalidAddress, "from_address is a module account")
		}
	}
	if acc := k.accountKeeper.GetAccount(sdk.UnwrapSDKContext(ctx), toAddr); acc != nil {
		if _, ok := acc.(sdk.ModuleAccountI); ok {
			return nil, cosmossdkerrors.Wrap(sdkerrors.ErrInvalidAddress, "to_address is a module account")
		}
	}

	// verify owner owns the root name of the class
	if err := k.VerifyDenomOwner(sdk.UnwrapSDKContext(ctx), msg.ClassId, msg.Owner); err != nil {
		return nil, err
	}

	// perform transfer via nftKeeper (class_id, nft_id)
	if err := k.nftKeeper.Transfer(ctx, msg.ClassId, msg.NftId, toAddr); err != nil {
		return nil, cosmossdkerrors.Wrap(err, "failed to move nft")
	}

	// emit event
	if evErr := sdk.UnwrapSDKContext(ctx).EventManager().EmitTypedEvent(
		&nameservicev1.EventNftMoved{
			ClassId:     msg.ClassId,
			NftId:       msg.NftId,
			FromAddress: msg.FromAddress,
			ToAddress:   msg.ToAddress,
		},
	); evErr != nil {
		k.Logger.Error("failed to emit nft moved event", "error", evErr)
	}

	k.Logger.Info("MoveNft: moved nft", "class", msg.ClassId, "id", msg.NftId)
	return &nameservicev1.MsgMoveNftResponse{}, nil
}
