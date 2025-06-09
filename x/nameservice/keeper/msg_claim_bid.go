package keeper

import (
	"context"
	"time"

	cosmossdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
	nameservicev1 "dysonprotocol.com/x/nameservice/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// ClaimBid implements the MsgServer.ClaimBid method
func (k Keeper) ClaimBid(ctx context.Context, msg *nameservicev1.MsgClaimBid) (*nameservicev1.MsgClaimBidResponse, error) {
	k.Logger.Info("ClaimBid: Processing claim request", "nft_class_id", msg.NftClassId, "nft_id", msg.NftId, "bidder", msg.Bidder)

	// Get current NFT data
	nftData, err := k.GetNFTData(ctx, msg.NftClassId, msg.NftId)
	if err != nil {
		return nil, cosmossdkerrors.Wrapf(err, "failed to get NFT data for class %s, id %s", msg.NftClassId, msg.NftId)
	}

	// Check if there is an active bid
	if nftData.CurrentBidder == "" {
		k.Logger.Error("ClaimBid: No active bid found", "nft_class_id", msg.NftClassId, "nft_id", msg.NftId)
		return nil, cosmossdkerrors.Wrap(sdkerrors.ErrNotFound, "no active bid found for this NFT")
	}

	// Verify bidder is the one who placed the bid
	if nftData.CurrentBidder != msg.Bidder {
		k.Logger.Error("ClaimBid: Unauthorized - not the bidder",
			"nft_class_id", msg.NftClassId,
			"nft_id", msg.NftId,
			"record_bidder", nftData.CurrentBidder,
			"msg_bidder", msg.Bidder)
		return nil, cosmossdkerrors.Wrap(sdkerrors.ErrUnauthorized, "only the bidder can claim the NFT")
	}

	// Check if bid timeout has elapsed
	currentTime := sdk.UnwrapSDKContext(ctx).BlockTime()
	params := k.GetParams(ctx)

	// Check if BidTimestamp is set
	if nftData.BidTimestamp == nil {
		k.Logger.Error("ClaimBid: No bid timestamp set", "nft_class_id", msg.NftClassId, "nft_id", msg.NftId)
		return nil, cosmossdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "no bid timestamp set")
	}

	// Calculate timeout time by adding the BidTimeout duration to the bid timestamp
	timeoutTime := nftData.BidTimestamp.Add(params.BidTimeout)

	// Check if current time is before the timeout time
	if currentTime.Before(timeoutTime) {
		k.Logger.Error("ClaimBid: Bid timeout has not elapsed",
			"current_time", currentTime.Format(time.RFC3339),
			"bid_timestamp", nftData.BidTimestamp.Format(time.RFC3339),
			"timeout_time", timeoutTime.Format(time.RFC3339))
		return nil, cosmossdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "bid timeout has not elapsed")
	}

	k.Logger.Info("ClaimBid: Bid timeout has elapsed",
		"current_time", currentTime.Format(time.RFC3339),
		"bid_timestamp", nftData.BidTimestamp.Format(time.RFC3339),
		"timeout_time", timeoutTime.Format(time.RFC3339))

	// Get the current owner of the NFT
	currentOwnerAddr := k.nftKeeper.GetOwner(ctx, msg.NftClassId, msg.NftId)
	currentOwner := currentOwnerAddr.String()

	// Convert bidder string to AccAddress
	bidderAddr, err := sdk.AccAddressFromBech32(msg.Bidder)
	if err != nil {
		k.Logger.Error("ClaimBid: Invalid bidder address", "bidder", msg.Bidder, "error", err)
		return nil, cosmossdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid bidder address: %s", msg.Bidder)
	}

	// Update the NFT data - use the current bid as the valuation
	var valuationCoin sdk.Coin
	if !nftData.CurrentBid.IsZero() {
		valuationCoin = nftData.CurrentBid
	} else {
		// If CurrentBid is invalid, use the first allowed denomination with zero amount
		if len(params.AllowedDenoms) > 0 {
			valuationCoin = sdk.NewCoin(params.AllowedDenoms[0], math.ZeroInt())
		} else {
			k.Logger.Error("ClaimBid: No allowed denominations configured")
			return nil, cosmossdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "no allowed denominations configured")
		}
	}

	// Update the NFT data
	nftData.Valuation = valuationCoin
	nftData.ValuationExpiry = *nftData.BidTimestamp // Reset expiry to the bid timestamp

	// Clear the bid information
	nftData.CurrentBidder = ""
	nftData.CurrentBid = sdk.Coin{}
	nftData.BidTimestamp = nil

	// Reset NFT data for the new owner
	if err := k.SetNFTData(ctx, msg.NftClassId, msg.NftId, nftData); err != nil {
		return nil, cosmossdkerrors.Wrapf(err, "failed to update NFT data")
	}

	// Transfer the NFT to the bidder
	err = k.nftKeeper.Transfer(ctx, msg.NftClassId, msg.NftId, bidderAddr)
	if err != nil {
		k.Logger.Error("ClaimBid: Failed to transfer NFT to bidder", "error", err)
		return nil, cosmossdkerrors.Wrap(err, "failed to transfer NFT to bidder")
	}

	// Emit event
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err = sdkCtx.EventManager().EmitTypedEvent(
		&nameservicev1.EventBidClaimed{
			ClassId: msg.NftClassId,
			NftId:   msg.NftId,
			Bidder:  msg.Bidder,
		})
	if err != nil {
		k.Logger.Error("ClaimBid: Failed to emit event", "error", err)
		return nil, cosmossdkerrors.Wrap(err, "failed to emit event")
	}

	k.Logger.Info("ClaimBid: Successfully claimed NFT",
		"nft_class_id", msg.NftClassId,
		"nft_id", msg.NftId,
		"bidder", msg.Bidder,
		"prev_owner", currentOwner)

	return &nameservicev1.MsgClaimBidResponse{}, nil
}
