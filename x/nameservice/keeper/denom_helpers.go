package keeper

import (
	"strings"

	cosmossdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// GetDenomOwner extracts the owner address for a given denom
// A denom in the nameservice follows the pattern "name.dys" or "name.dys/subdenom"
// where the name before any slash is a registered name in the nameservice
// Returns the owner address and nil if successful, or empty string and error if not
func (k Keeper) GetDenomOwner(ctx sdk.Context, denom string) (string, string, error) {
	k.Logger.Info("GetDenomOwner called", "denom", denom)

	// Extract the root name from the denom (everything before the first '/')
	rootName := denom
	if idx := strings.Index(denom, "/"); idx > 0 {
		rootName = denom[:idx]
	}

	k.Logger.Info("Extracted root name", "rootName", rootName)

	// Check if it's a valid nameservice name (must end with ".dys")
	if !strings.HasSuffix(rootName, ".dys") {
		k.Logger.Error("Invalid denom format", "rootName", rootName)
		return "", "", cosmossdkerrors.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"invalid denom format, root name must be a valid .dys name: %s",
			rootName,
		)
	}

	// Check if the name exists and get its owner
	owner, found := k.GetNameOwner(ctx, rootName)
	if !found {
		k.Logger.Error("Root name not found", "rootName", rootName)
		return "", "", cosmossdkerrors.Wrapf(
			sdkerrors.ErrNotFound,
			"root name not found: %s",
			rootName,
		)
	}

	k.Logger.Info("Found name", "name", rootName, "owner", owner)

	// Return the owner of the name
	return owner, rootName, nil
}

// VerifyDenomOwner checks if the provided address is the owner of the denom
// Returns nil if the address is the owner, or an error if not
func (k Keeper) VerifyDenomOwner(ctx sdk.Context, denom string, address string) error {
	k.Logger.Info("VerifyDenomOwner called", "denom", denom, "address", address)

	owner, _, err := k.GetDenomOwner(ctx, denom)
	if err != nil {
		return err
	}

	k.Logger.Info("Comparing owners", "owner from record", owner, "requesting address", address)

	if owner != address {
		k.Logger.Error("Authorization failed", "denom", denom, "owner", owner, "requester", address)
		return cosmossdkerrors.Wrapf(
			sdkerrors.ErrUnauthorized,
			"you do not own the name required to use denom %s, owner: %s, sender: %s",
			denom,
			owner,
			address,
		)
	}

	k.Logger.Info("Authorization successful", "denom", denom, "address", address)
	return nil
}
