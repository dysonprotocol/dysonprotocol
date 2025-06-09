package keeper

import (
	"context"

	cosmossdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"cosmossdk.io/x/nft"
	nameservicev1 "dysonprotocol.com/x/nameservice/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// Reveal implements the MsgServer.Reveal method
func (k Keeper) Reveal(ctx context.Context, msg *nameservicev1.MsgReveal) (*nameservicev1.MsgRevealResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Validate addresses
	committerAddr, err := sdk.AccAddressFromBech32(msg.Committer)
	if err != nil {
		return nil, cosmossdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid committer address: %s", msg.Committer)
	}

	// Validate name is not empty
	if msg.Name == "" {
		return nil, cosmossdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "name cannot be empty")
	}

	// Validate that the name follows the format: lowercase alphanumeric, starts with letter, may contain dashes, ends with ".dys"
	if !nameservicev1.NameRegex.MatchString(msg.Name) {
		return nil, cosmossdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "invalid name format: must be lowercase, start with a letter, contain only alphanumeric and dash characters, and end with .dys")
	}

	// Check if name is already registered
	if k.nftKeeper.HasNFT(ctx, NamesClassID, msg.Name) {
		return nil, cosmossdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "name is already registered")
	}

	// Calculate hash using the common hash function
	hexhash := k.ComputeNameRegistrationHash(msg.Name, msg.Committer, msg.Salt)

	// Get the commitment
	commitment, err := k.GetCommitment(ctx, hexhash)
	if err != nil {
		return nil, cosmossdkerrors.Wrap(err, "commitment not found")
	}

	// Validate committer matches commitment
	if commitment.Owner != msg.Committer {
		return nil, cosmossdkerrors.Wrap(sdkerrors.ErrUnauthorized, "committer does not match commitment")
	}

	// Ensure the Names NFT class exists before calculating fees
	if err := k.EnsureNamesClassExists(ctx); err != nil {
		return nil, cosmossdkerrors.Wrap(err, "failed to ensure names NFT class exists")
	}

	// Use the valuation from the commitment
	valuation := commitment.Valuation

	// Validate that the valuation is not zero after assignment
	if valuation.IsZero() {
		return nil, cosmossdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "valuation cannot be zero - please set a valuation in the commitment")
	}

	// Validate the valuation
	err = k.ValidateValuation(ctx, valuation)
	if err != nil {
		return nil, err
	}

	// Calculate and charge the annual fee based on the valuation
	var fee sdk.Coins

	// Get the annual fee percentage from the NFT class metadata
	feePercentStr, err := k.GetNamesClassAnnualPct(ctx)
	if err != nil {
		return nil, cosmossdkerrors.Wrap(err, "failed to get annual percentage for fee calculation")
	}

	// Convert the percentage string to a decimal
	feePercent, err := math.LegacyNewDecFromStr(feePercentStr)
	if err != nil {
		return nil, cosmossdkerrors.Wrap(err, "failed to parse annual valuation fee percent")
	}

	if !feePercent.IsZero() {
		// Convert the single coin to a DecCoins for precise math operations
		decValuation := sdk.NewDecCoinsFromCoins(valuation)

		// Convert to LegacyDec for compatibility with SDK DecCoins methods
		legacyFeePercent, err := math.LegacyNewDecFromStr(feePercent.String())
		if err != nil {
			return nil, cosmossdkerrors.Wrap(err, "failed to convert to legacy decimal")
		}

		// Calculate fee by multiplying the valuation by fee percentage
		decFees := decValuation.MulDec(legacyFeePercent)

		// Convert back to regular Coins for blockchain transactions
		fee, _ = decFees.TruncateDecimal()

		// Charge the fee
		if !fee.IsZero() {
			err = k.communityPoolKeeper.FundCommunityPool(ctx, fee, committerAddr.Bytes())
			if err != nil {
				return nil, cosmossdkerrors.Wrap(err, "failed to send fee to community pool")
			}
		}
	}

	// Create NFT data
	nftData := &nameservicev1.NFTData{
		Listed:          true,
		Valuation:       valuation,
		ValuationExpiry: sdkCtx.BlockTime().AddDate(1, 0, 0), // 1 year from now
		Metadata:        "",
	}

	// Marshal the NFT data
	nftDataAny, err := codectypes.NewAnyWithValue(nftData)
	if err != nil {
		return nil, cosmossdkerrors.Wrap(err, "failed to marshal NFT data")
	}

	// Create the NFT
	token := nft.NFT{
		ClassId: NamesClassID,
		Id:      msg.Name,
		Uri:     msg.Committer,
		UriHash: "",
		Data:    nftDataAny,
	}

	// Mint the NFT
	if err := k.nftKeeper.Mint(ctx, token, committerAddr); err != nil {
		return nil, cosmossdkerrors.Wrap(err, "failed to mint name NFT")
	}

	// Delete commitment
	err = k.DeleteCommitment(ctx, hexhash)
	if err != nil {
		return nil, cosmossdkerrors.Wrap(err, "failed to delete commitment")
	}

	// Emit event
	err = sdkCtx.EventManager().EmitTypedEvent(
		&nameservicev1.EventNameRegistered{
			Name: msg.Name,
			Fee:  fee,
		})
	if err != nil {
		k.Logger.Error("failed to emit name registered event", "error", err)
		return nil, cosmossdkerrors.Wrap(err, "failed to emit name registered event")
	}

	return &nameservicev1.MsgRevealResponse{}, nil
}
