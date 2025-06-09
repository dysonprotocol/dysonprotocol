package keeper

import (
	"context"
	"fmt"

	cosmossdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
	nameservice "dysonprotocol.com/x/nameservice"
	nameservicev1 "dysonprotocol.com/x/nameservice/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// PlaceBid implements the MsgServer.PlaceBid method
func (k Keeper) PlaceBid(ctx context.Context, msg *nameservicev1.MsgPlaceBid) (*nameservicev1.MsgPlaceBidResponse, error) {
	k.Logger.Info("PlaceBid: Processing bid", "nft_class_id", msg.NftClassId, "nft_id", msg.NftId, "bidder", msg.Bidder, "bid_amount", msg.BidAmount)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Validate the bid amount using shared validation logic
	if err := k.ValidateValuation(ctx, msg.BidAmount); err != nil {
		return nil, cosmossdkerrors.Wrap(err, "invalid bid amount")
	}

	// Load the NFT data
	nftData, err := k.GetNFTData(ctx, msg.NftClassId, msg.NftId)
	if err != nil {
		return nil, cosmossdkerrors.Wrapf(err, "failed to get NFT data for class %s, ID %s", msg.NftClassId, msg.NftId)
	}

	// Get the current owner of the NFT
	nftOwnerAddr := k.nftKeeper.GetOwner(ctx, msg.NftClassId, msg.NftId)

	// Check if the owner is the authority address
	authorityAddr, err := sdk.AccAddressFromBech32(k.GetAuthority())
	if err == nil && nftOwnerAddr.Equals(authorityAddr) {
		k.Logger.Error("PlaceBid: Cannot place bid on NFT owned by the authority",
			"nft_class_id", msg.NftClassId, "nft_id", msg.NftId,
			"nft_owner", nftOwnerAddr.String())
		return nil, cosmossdkerrors.Wrap(sdkerrors.ErrUnauthorized, "cannot place bid on NFT owned by the authority")
	}

	// Get module parameters
	params := k.GetParams(ctx)
	k.Logger.Info("PlaceBid: Got params", "allowed_denoms", params.AllowedDenoms)

	// Get current timestamp
	currentTime := sdkCtx.BlockTime()

	// Check if valuation has expired
	valuationExpired := false
	if !nftData.Valuation.IsZero() {
		// Check the NFT data for valuation expiry
		if nftData.ValuationExpiry.Before(currentTime) {
			k.Logger.Info("PlaceBid: Valuation has expired",
				"nft_class_id", msg.NftClassId, "nft_id", msg.NftId,
				"valuation_expiry", nftData.ValuationExpiry.String())
			valuationExpired = true
		}
	}

	// Determine if valuation is active and applicable
	hasActiveValuation := !nftData.Valuation.IsZero() && !valuationExpired

	// ---- Validation Phase ----

	// Validate denomination
	if hasActiveValuation && nftData.Valuation.Denom != msg.BidAmount.Denom {
		k.Logger.Error("PlaceBid: Bid denomination does not match valuation denomination",
			"bid_denom", msg.BidAmount.Denom, "valuation_denom", nftData.Valuation.Denom)
		return nil, cosmossdkerrors.Wrap(sdkerrors.ErrInvalidRequest,
			fmt.Sprintf("bid denomination (%s) must match valuation denomination (%s)",
				msg.BidAmount.Denom, nftData.Valuation.Denom))
	}

	// Validate bid amount based on whether there's an existing bid
	if nftData.CurrentBidder != "" {
		// Case: Subsequent bid
		if !nftData.CurrentBid.IsZero() {
			// Ensure bid denomination matches current bid denomination
			if nftData.CurrentBid.Denom != msg.BidAmount.Denom {
				k.Logger.Error("PlaceBid: Bid denomination does not match current bid denomination",
					"bid_denom", msg.BidAmount.Denom, "current_bid_denom", nftData.CurrentBid.Denom)
				return nil, cosmossdkerrors.Wrap(sdkerrors.ErrInvalidRequest,
					fmt.Sprintf("bid denomination (%s) must match current bid denomination (%s)",
						msg.BidAmount.Denom, nftData.CurrentBid.Denom))
			}

			// Get the minimum bid percentage increase from params
			minBidIncrease, err := params.GetMinimumBidPercentIncreaseAsDec()
			if err != nil {
				k.Logger.Error("PlaceBid: Failed to parse minimum bid percent increase", "error", err)
				return nil, cosmossdkerrors.Wrap(err, "failed to parse minimum bid percent increase")
			}

			// First check if the bid is higher at all
			if !msg.BidAmount.IsGT(nftData.CurrentBid) {
				k.Logger.Error("PlaceBid: Bid amount not higher than current bid",
					"bid_amount", msg.BidAmount, "current_bid", nftData.CurrentBid)
				return nil, cosmossdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "bid amount must be higher than current bid")
			}

			// Then check if it meets the minimum percentage increase
			// Convert existing bid amount to Dec for percentage calculation
			currentBidAmountLegacy, err := math.LegacyNewDecFromStr(nftData.CurrentBid.Amount.String())
			if err != nil {
				k.Logger.Error("PlaceBid: Failed to convert current bid amount to decimal", "error", err)
				return nil, cosmossdkerrors.Wrap(err, "failed to convert current bid amount to decimal")
			}

			// Calculate minimum required bid amount: current_bid * (1 + min_increase)
			// Need to convert minBidIncrease (Dec) to LegacyDec
			minBidIncreaseLegacy, err := math.LegacyNewDecFromStr(minBidIncrease.String())
			if err != nil {
				k.Logger.Error("PlaceBid: Failed to convert minimum bid increase to legacy decimal", "error", err)
				return nil, cosmossdkerrors.Wrap(err, "failed to convert minimum bid increase to legacy decimal")
			}

			onePlusIncrease := math.LegacyOneDec().Add(minBidIncreaseLegacy)
			minRequiredBidAmount := currentBidAmountLegacy.Mul(onePlusIncrease).Ceil()

			// Convert new bid amount to Dec for comparison
			newBidAmountLegacy, err := math.LegacyNewDecFromStr(msg.BidAmount.Amount.String())
			if err != nil {
				k.Logger.Error("PlaceBid: Failed to convert new bid amount to decimal", "error", err)
				return nil, cosmossdkerrors.Wrap(err, "failed to convert new bid amount to decimal")
			}

			// Compare the new bid with the minimum required amount
			if newBidAmountLegacy.LT(minRequiredBidAmount) {
				k.Logger.Error("PlaceBid: Bid amount does not meet minimum percentage increase",
					"bid_amount", msg.BidAmount.Amount.String(),
					"current_bid", nftData.CurrentBid.Amount.String(),
					"min_required", minRequiredBidAmount.String(),
					"min_increase_percent", params.MinimumBidPercentIncrease)
				return nil, cosmossdkerrors.Wrap(sdkerrors.ErrInvalidRequest,
					fmt.Sprintf("bid amount (%s) must be at least %s%% higher than current bid (%s) with a minimum bid of (%s)",
						msg.BidAmount.Amount.String(), params.MinimumBidPercentIncrease, nftData.CurrentBid.Amount.String(), minRequiredBidAmount.String()))
			}

			k.Logger.Info("PlaceBid: New bid meets minimum percentage increase requirement",
				"new_bid", msg.BidAmount, "current_bid", nftData.CurrentBid,
				"min_increase_percent", params.MinimumBidPercentIncrease)
		}
		k.Logger.Info("PlaceBid: New bid is higher than current bid",
			"new_bid", msg.BidAmount, "current_bid", nftData.CurrentBid)
	} else {
		// Case: First bid - must be >= valuation if valuation is active
		if hasActiveValuation && !msg.BidAmount.IsGTE(nftData.Valuation) {
			k.Logger.Error("PlaceBid: First bid amount must be >= valuation",
				"bid_amount", msg.BidAmount, "valuation", nftData.Valuation)
			return nil, cosmossdkerrors.Wrap(sdkerrors.ErrInvalidRequest,
				"first bid amount must be greater than or equal to name valuation")
		}
		k.Logger.Info("PlaceBid: First bid meets or exceeds valuation",
			"bid_amount", msg.BidAmount, "valuation", nftData.Valuation)
	}

	// Convert bidder string to AccAddress
	bidder, err := sdk.AccAddressFromBech32(msg.Bidder)
	if err != nil {
		k.Logger.Error("PlaceBid: Invalid bidder address", "bidder", msg.Bidder, "error", err)
		return nil, cosmossdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid bidder address: %s", msg.Bidder)
	}

	// ---- Transaction Phase ----

	// If there's an existing bid, refund the previous bidder
	if nftData.CurrentBidder != "" && nftData.CurrentBidder != msg.Bidder {
		previousBidder, err := sdk.AccAddressFromBech32(nftData.CurrentBidder)
		if err != nil {
			k.Logger.Error("PlaceBid: Invalid previous bidder address", "previous_bidder", nftData.CurrentBidder, "error", err)
			return nil, cosmossdkerrors.Wrapf(err, "invalid previous bidder address: %s", nftData.CurrentBidder)
		}

		// Return the escrowed funds to the previous bidder if current bid is not zero
		if !nftData.CurrentBid.IsZero() {
			prevBidCoins := sdk.NewCoins(nftData.CurrentBid)
			err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, nameservice.ModuleName, previousBidder, prevBidCoins)
			if err != nil {
				k.Logger.Error("PlaceBid: Failed to refund previous bidder",
					"previous_bidder", nftData.CurrentBidder, "amount", nftData.CurrentBid, "error", err)
				return nil, cosmossdkerrors.Wrap(err, "failed to refund previous bidder")
			}
			k.Logger.Info("PlaceBid: Refunded previous bidder",
				"previous_bidder", nftData.CurrentBidder, "amount", nftData.CurrentBid)
		} else {
			k.Logger.Info("PlaceBid: No refund needed for previous bidder (empty bid)",
				"previous_bidder", nftData.CurrentBidder)
		}
	}

	// Escrow the bid amount from the bidder
	bidCoins := sdk.NewCoins(msg.BidAmount)
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, bidder, nameservice.ModuleName, bidCoins)
	if err != nil {
		k.Logger.Error("PlaceBid: Failed to escrow bid amount", "bidder", msg.Bidder, "amount", msg.BidAmount, "error", err)
		return nil, cosmossdkerrors.Wrap(err, "failed to escrow bid amount")
	}
	k.Logger.Info("PlaceBid: Successfully escrowed bid amount", "bidder", msg.Bidder, "amount", msg.BidAmount)

	// ---- State Update Phase ----

	// Update the NFT bid information
	nftData.CurrentBidder = msg.Bidder
	nftData.CurrentBid = msg.BidAmount
	bidTimestamp := sdkCtx.BlockTime()

	nftData.BidTimestamp = &bidTimestamp

	// Update the NFT data in the store
	if err := k.SetNFTData(ctx, msg.NftClassId, msg.NftId, nftData); err != nil {
		k.Logger.Error("PlaceBid: Failed to update NFT data", "error", err)
		return nil, cosmossdkerrors.Wrap(err, "failed to update NFT data")
	}

	k.Logger.Info("PlaceBid: Successfully updated NFT bid information",
		"nft_class_id", msg.NftClassId,
		"nft_id", msg.NftId,
		"bidder", msg.Bidder,
		"bid_amount", msg.BidAmount)

	// Emit event
	if err := sdkCtx.EventManager().EmitTypedEvent(
		&nameservicev1.EventBidPlaced{
			ClassId:   msg.NftClassId,
			NftId:     msg.NftId,
			Bidder:    msg.Bidder,
			BidAmount: &msg.BidAmount,
		},
	); err != nil {
		k.Logger.Error("PlaceBid: Failed to emit event", "error", err)
		return nil, cosmossdkerrors.Wrap(err, "failed to emit event")
	}

	return &nameservicev1.MsgPlaceBidResponse{}, nil
}
