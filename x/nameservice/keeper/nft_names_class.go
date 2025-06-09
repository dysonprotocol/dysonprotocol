package keeper

import (
	"context"

	cosmossdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/x/nft"
	nameservicev1 "dysonprotocol.com/x/nameservice/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// Names NFT class constants
const (
	NamesClassID          = "nameservice.dys"
	NamesClassName        = "Dyson Names"
	NamesClassSymbol      = "DYSNAME"
	NamesClassDescription = "Dyson Protocol registered names"
	NamesClassURI         = ""
)

// EnsureNamesClassExists ensures that the "nameservice" NFT class exists
// If it doesn't exist, it creates it with the module account as the owner
// It also ensures that an NFT with ID=NamesClassID exists and is owned by the authority
func (k Keeper) EnsureNamesClassExists(ctx context.Context) error {
	// Step 1: Ensure the NFT class exists
	if !k.nftKeeper.HasClass(ctx, NamesClassID) {
		// Create the NFT class with empty data first
		class := nft.Class{
			Id:          NamesClassID,
			Name:        NamesClassName,
			Symbol:      NamesClassSymbol,
			Description: NamesClassDescription,
			Uri:         NamesClassURI,
			UriHash:     "",
			Data:        nil, // We'll set this using SetNFTClassData after creating the class
		}

		// Save the class
		if err := k.nftKeeper.SaveClass(ctx, class); err != nil {
			return cosmossdkerrors.Wrap(err, "failed to save Names NFT class")
		}

		// Create NFT class data
		nftClassData := nameservicev1.NewNFTClassData()
		nftClassData.AlwaysListed = true // Names are always listed for sale
		nftClassData.AnnualPct = "0.01"  // 1% annual fee by default (can be adjusted via nft_names_class_params later)

		// Use the SetNFTClassData helper function to set the class data
		if err := k.SetNFTClassData(ctx, NamesClassID, *nftClassData); err != nil {
			return cosmossdkerrors.Wrap(err, "failed to set NFT class data")
		}

		k.Logger.Info("Successfully created Names NFT class",
			"class_id", NamesClassID,
			"always_listed", nftClassData.AlwaysListed,
			"annual_pct", nftClassData.AnnualPct)
	}

	// Step 2: Ensure the authority NFT exists
	if !k.nftKeeper.HasNFT(ctx, NamesClassID, NamesClassID) {
		// Get the authority address
		authorityAddr, err := sdk.AccAddressFromBech32(k.GetAuthority())
		if err != nil {
			return cosmossdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority address: %s", k.GetAuthority())
		}

		// Create the authority NFT with empty data first
		token := nft.NFT{
			ClassId: NamesClassID,
			Id:      NamesClassID,
			Uri:     "",
			UriHash: "",
			Data:    nil, // We'll set this using SetNFTData after minting
		}

		// Mint the authority NFT
		if err := k.nftKeeper.Mint(ctx, token, authorityAddr); err != nil {
			return cosmossdkerrors.Wrap(err, "failed to mint authority NFT")
		}

		// Create default NFT data
		nftData := nameservicev1.NewNFTData()
		nftData.Listed = false // Authority NFT is not listed by default

		// Use SetNFTData to set the NFT data
		if err := k.SetNFTData(ctx, NamesClassID, NamesClassID, *nftData); err != nil {
			return cosmossdkerrors.Wrapf(err, "failed to set NFT data for %s", NamesClassID)
		}

		k.Logger.Info("Successfully minted authority NFT",
			"owner", k.GetAuthority(),
			"class_id", NamesClassID,
			"nft_id", NamesClassID)
	}

	return nil
}

// MintNameNFT mints a new NFT in the names class for a registered name
func (k Keeper) MintNameNFT(ctx context.Context, name string, owner string) error {
	// Ensure the names class exists
	if err := k.EnsureNamesClassExists(ctx); err != nil {
		return err
	}

	// Convert owner to account address
	ownerAddr, err := sdk.AccAddressFromBech32(owner)
	if err != nil {
		return cosmossdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid owner address: %s", owner)
	}

	// Check if NFT already exists (should not normally happen)
	if k.nftKeeper.HasNFT(ctx, NamesClassID, name) {
		return cosmossdkerrors.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"NFT already exists in class %s with ID %s",
			NamesClassID,
			name,
		)
	}

	// Create the NFT with empty data first
	token := nft.NFT{
		ClassId: NamesClassID,
		Id:      name,
		Uri:     "", // Can be set to point to name metadata if needed
		UriHash: "",
		Data:    nil, // We'll set this using SetNFTData after minting
	}

	// Mint the NFT
	if err := k.nftKeeper.Mint(ctx, token, ownerAddr); err != nil {
		return cosmossdkerrors.Wrap(err, "failed to mint Name NFT")
	}

	// Create NFT data with default values
	nftData := nameservicev1.NewNFTData()
	nftData.Listed = true // Names are always listed by default

	// Use SetNFTData to set the NFT data
	if err := k.SetNFTData(ctx, NamesClassID, name, *nftData); err != nil {
		return cosmossdkerrors.Wrapf(err, "failed to set NFT data for %s", name)
	}

	k.Logger.Info("Successfully minted Name NFT",
		"owner", owner,
		"class_id", NamesClassID,
		"nft_id", name,
		"valuation", nftData.Valuation.String())

	return nil
}
