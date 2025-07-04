package keeper

import (
	"context"

	cosmossdkerrors "cosmossdk.io/errors"
	math "cosmossdk.io/math"
	nameservicev1 "dysonprotocol.com/x/nameservice/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	SecondsInYear = 31536000
)

// Minimum valuation amount is dynamically computed as 1 / annual_pct for the
// NFT class. This ties the minimum valuation to the economic parameters of the
// class instead of a hard-coded constant.

// SetValuation implements the MsgServer.SetValuation method
func (k Keeper) SetValuation(ctx context.Context, msg *nameservicev1.MsgSetValuation) (*nameservicev1.MsgSetValuationResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	k.Logger.Info("SetValuation: Processing", "class_id", msg.NftClassId, "nft_id", msg.NftId, "owner", msg.Owner, "valuation", msg.Valuation.String())

	// Extract the NFT data
	nftData, err := k.GetNFTData(ctx, msg.NftClassId, msg.NftId)
	if err != nil {
		k.Logger.Error("SetValuation: NFT not found", "class_id", msg.NftClassId, "nft_id", msg.NftId, "error", err)
		return nil, cosmossdkerrors.Wrapf(err, "failed to get NFT data")
	}

	// Get the current owner of the NFT
	ownerAddr := k.nftKeeper.GetOwner(ctx, msg.NftClassId, msg.NftId)
	nftOwner := ownerAddr.String()

	// Convert message sender address
	msgOwnerAddr, err := sdk.AccAddressFromBech32(msg.Owner)
	if err != nil {
		k.Logger.Error("SetValuation: Invalid owner address", "owner", msg.Owner, "error", err)
		return nil, cosmossdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid owner address: %s", msg.Owner)
	}

	// Verify that message owner is the NFT owner
	if nftOwner != msg.Owner {
		return nil, cosmossdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "only the NFT owner can set valuation: NFT owner %s != message sender %s", nftOwner, msg.Owner)
	}

	// Validate the valuation using the keeper's validation method
	if err := k.ValidateValuation(ctx, msg.Valuation); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------
	// Active Bid Check: Disallow valuation change if a bid exists
	// -------------------------------------------------------------------
	if nftData.CurrentBidder != "" {
		return nil, cosmossdkerrors.Wrapf(sdkerrors.ErrInvalidRequest,
			"cannot set valuation while there is an active bid, use MsgRejectBid to reject the bid with a new valuation")
	}

	// -------------------------------------------------------------------
	// Dynamic Minimum Valuation Check
	// -------------------------------------------------------------------

	// Fetch NFT class data to obtain annual_pct
	classData, err := k.GetNFTClassData(ctx, msg.NftClassId)
	if err != nil {
		return nil, cosmossdkerrors.Wrap(err, "failed to get NFT class data for minimum valuation check")
	}

	if classData.AnnualPct == "" {
		return nil, cosmossdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "NFT class has no annual_pct set; cannot compute minimum valuation")
	}

	// Convert annual_pct to decimal. Expecting a value like "0.05" for 5%.
	pctDec, err := math.LegacyNewDecFromStr(classData.AnnualPct)
	if err != nil {
		return nil, cosmossdkerrors.Wrap(err, "invalid annual_pct in NFT class data")
	}

	if pctDec.IsZero() {
		return nil, cosmossdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "annual_pct cannot be zero when computing minimum valuation")
	}

	// minVal = ceil(1 / annual_pct)
	minValDec := math.LegacyOneDec().Quo(pctDec)
	minValInt := minValDec.Ceil().TruncateInt()

	if msg.Valuation.Amount.LT(minValInt) {
		return nil, cosmossdkerrors.Wrapf(sdkerrors.ErrInvalidRequest,
			"valuation amount must be at least %s %s (computed from 1/%s [annual_pct])",
			minValInt.String(), msg.Valuation.Denom, classData.AnnualPct)
	}

	// Calculate the incremental valuation (only if increasing)
	var oldValuation sdk.Coins
	if nftData.Valuation.Amount.IsNil() || nftData.Valuation.Denom == "" {
		oldValuation = sdk.NewCoins()
	} else {
		oldValuation = sdk.NewCoins(nftData.Valuation)
	}
	newValuation := sdk.NewCoins(msg.Valuation)

	// Only charge fee if there's an incremental valuation increase
	if newValuation.IsAllGT(oldValuation) {
		// Calculate the difference between new and old valuation
		incrementalValuation := newValuation.Sub(oldValuation...)

		// Get current and expiry time
		currentTime := sdkCtx.BlockTime()
		expiryTime := nftData.ValuationExpiry

		// Calculate time proportion - using LegacyDec for decimal precision
		remainingSeconds := expiryTime.Unix() - currentTime.Unix()
		if remainingSeconds <= 0 {
			k.Logger.Info("SetValuation: Valuation already expired, skipping fee calculation")
			remainingSeconds = 0
		}
		portionRemaining := math.LegacyNewDecFromInt(math.NewInt(remainingSeconds)).
			Quo(math.LegacyNewDecFromInt(math.NewInt(SecondsInYear)))

		// Get the annual fee percentage from the NFT class metadata
		feePercentStr, err := k.GetNamesClassAnnualPct(ctx)
		if err != nil {
			k.Logger.Error("SetValuation: Failed to get annual percentage from class metadata", "error", err)
			return nil, cosmossdkerrors.Wrap(err, "failed to get annual percentage for valuation fee calculation")
		}

		// Convert the percentage string to a decimal
		feePercent, err := math.LegacyNewDecFromStr(feePercentStr)
		if err != nil {
			return nil, cosmossdkerrors.Wrap(err, "failed to parse annual valuation fee percent")
		}

		// Calculate fee: incremental_valuation * fee_percent * time_proportion
		decValuationDiff := sdk.NewDecCoinsFromCoins(incrementalValuation...)
		decAnnualFee := decValuationDiff.MulDec(feePercent)
		decProportionalFee := decAnnualFee.MulDec(portionRemaining)
		feeCoins, _ := decProportionalFee.TruncateDecimal()

		k.Logger.Info("Charging proportional annual fee for increased valuation",
			"class_id", msg.NftClassId,
			"nft_id", msg.NftId,
			"old_valuation", oldValuation.String(),
			"new_valuation", newValuation.String(),
			"incremental_valuation", incrementalValuation.String(),
			"time_proportion", portionRemaining.String(),
			"fee", feeCoins.String())

		err = k.communityPoolKeeper.FundCommunityPool(ctx, feeCoins, msgOwnerAddr)
		if err != nil {
			return nil, cosmossdkerrors.Wrap(err, "failed to send fee to community pool")
		}
	}

	// Update NFT data
	nftData.Valuation = msg.Valuation
	// Keep the same expiry time - we're just updating the valuation, not extending

	// Update the NFT data
	if err := k.SetNFTData(ctx, msg.NftClassId, msg.NftId, nftData); err != nil {
		k.Logger.Error("SetValuation: Failed to update NFT data", "error", err)
		return nil, cosmossdkerrors.Wrapf(err, "failed to update NFT data")
	}

	// Emit event
	err = sdkCtx.EventManager().EmitTypedEvent(
		&nameservicev1.EventNameValuationUpdated{
			Name:         msg.NftId,
			NewValuation: &msg.Valuation,
		})
	if err != nil {
		k.Logger.Error("failed to emit valuation updated event", "error", err)
		return nil, cosmossdkerrors.Wrap(err, "failed to emit valuation updated event")
	}

	k.Logger.Info("SetValuation: Completed successfully",
		"class_id", msg.NftClassId,
		"nft_id", msg.NftId,
		"new_valuation", msg.Valuation.String())

	return &nameservicev1.MsgSetValuationResponse{}, nil
}
