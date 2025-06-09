package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	cosmossdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	nameservicev1 "dysonprotocol.com/x/nameservice/types"
)

var (
	// CommitmentsKey is the key for commitments
	CommitmentsKey = collections.NewPrefix(2)

	// ParamsKey is the key for parameters
	ParamsKey = collections.NewPrefix(4)
)

// Keeper defines the nameservice keeper
type Keeper struct {
	nameservicev1.UnimplementedMsgServer

	cdc                 codec.Codec
	storeService        store.KVStoreService
	bankKeeper          nameservicev1.BankKeeper
	accountKeeper       nameservicev1.AccountKeeper
	communityPoolKeeper nameservicev1.CommunityPoolKeeper
	nftKeeper           nameservicev1.NFTKeeper

	// Core services
	Logger log.Logger

	Schema      collections.Schema
	commitments collections.Map[string, nameservicev1.Commitment]
	params      collections.Item[nameservicev1.Params]

	authority string // the address that is authorized to update module parameters
}

// NewKeeper creates a new nameservice Keeper instance
func NewKeeper(
	cdc codec.Codec,
	storeService store.KVStoreService,
	bankKeeper nameservicev1.BankKeeper,
	accountKeeper nameservicev1.AccountKeeper,
	communityPoolKeeper nameservicev1.CommunityPoolKeeper,
	nftKeeper nameservicev1.NFTKeeper,
	logger log.Logger,
	authority string,
) Keeper {
	// Create schema builder
	sb := collections.NewSchemaBuilder(storeService)

	// Create map for commitments
	commitments := collections.NewMap(
		sb,
		CommitmentsKey,
		"commitments",
		collections.StringKey,
		codec.CollValue[nameservicev1.Commitment](cdc),
	)

	// Create item for params
	params := collections.NewItem(
		sb,
		ParamsKey,
		"params",
		codec.CollValue[nameservicev1.Params](cdc),
	)

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}

	return Keeper{
		cdc:                 cdc,
		storeService:        storeService,
		bankKeeper:          bankKeeper,
		accountKeeper:       accountKeeper,
		communityPoolKeeper: communityPoolKeeper,
		nftKeeper:           nftKeeper,
		Logger:              logger,
		Schema:              schema,
		commitments:         commitments,
		params:              params,
		authority:           authority,
	}
}

// GetParams returns the current module parameters
func (k Keeper) GetParams(ctx context.Context) (params nameservicev1.Params) {
	params, err := k.params.Get(ctx)
	if err != nil {
		// If params don't exist, return defaults
		return nameservicev1.DefaultParams()
	}
	return params
}

// SetParams sets the module parameters
func (k Keeper) SetParams(ctx context.Context, params nameservicev1.Params) error {
	if err := params.Validate(); err != nil {
		return err
	}
	// Check if store service is accessible
	if k.storeService == nil {
		k.Logger.Error("SetParams: Store service is nil, cannot set params")
		return fmt.Errorf("store service is nil, cannot set parameters")
	}

	// Additional check to ensure store is initialized
	store := k.storeService.OpenKVStore(ctx)
	if store == nil {
		k.Logger.Error("SetParams: Failed to open KV store, store is nil")
		return fmt.Errorf("failed to open KV store, store is nil")
	}
	return k.params.Set(ctx, params)
}

// GetNFTData gets the NFTData for an NFT with the given class ID and NFT ID
func (k Keeper) GetNFTData(ctx context.Context, classId string, nftId string) (nameservicev1.NFTData, error) {
	k.Logger.Info("GetNFTData: Getting NFT data", "class_id", classId, "nft_id", nftId)

	nft, found := k.nftKeeper.GetNFT(ctx, classId, nftId)
	if !found {
		k.Logger.Error("GetNFTData: NFT not found", "class_id", classId, "nft_id", nftId)
		return nameservicev1.NFTData{}, cosmossdkerrors.Wrapf(sdkerrors.ErrNotFound, "NFT not found: %s", nftId)
	}

	k.Logger.Info("GetNFTData: Found NFT", "class_id", classId, "nft_id", nftId, "has_data", nft.Data != nil)

	var nftData nameservicev1.NFTData

	// Instead of using GetCachedValue() or UnpackAny, directly unmarshal the Value bytes
	k.Logger.Info("GetNFTData: Directly unmarshaling data", "class_id", classId, "nft_id", nftId)

	if nft.Data != nil {
		if err := k.cdc.Unmarshal(nft.Data.Value, &nftData); err != nil {
			k.Logger.Error("GetNFTData: Failed to unmarshal NFT data",
				"class_id", classId,
				"nft_id", nftId,
				"error", err,
				"type_url", nft.Data.TypeUrl,
				"value_len", len(nft.Data.Value))
			return nameservicev1.NFTData{}, cosmossdkerrors.Wrap(err, "failed to unmarshal NFT data")
		}
	}

	k.Logger.Info("GetNFTData: Successfully unmarshaled data", "class_id", classId, "nft_id", nftId, "listed", nftData.Listed)

	return nftData, nil
}

// GetNameOwner gets the owner of a name
func (k Keeper) GetNameOwner(ctx context.Context, name string) (string, bool) {
	if !k.nftKeeper.HasNFT(ctx, NamesClassID, name) {
		return "", false
	}

	owner := k.nftKeeper.GetOwner(ctx, NamesClassID, name)
	return owner.String(), true
}

// GetCommitment gets a commitment
func (k Keeper) GetCommitment(ctx context.Context, hexhash string) (nameservicev1.Commitment, error) {
	return k.commitments.Get(ctx, hexhash)
}

// SetCommitment sets a commitment
func (k Keeper) SetCommitment(ctx context.Context, commitment nameservicev1.Commitment) error {
	return k.commitments.Set(ctx, commitment.Hexhash, commitment)
}

// DeleteCommitment deletes a commitment
func (k Keeper) DeleteCommitment(ctx context.Context, hexhash string) error {
	return k.commitments.Remove(ctx, hexhash)
}

// GetAllCommitmentHashes returns all commitment hashes
func (k Keeper) GetAllCommitmentHashes(ctx context.Context) ([]string, error) {
	var hashes []string

	if err := k.commitments.Walk(ctx, nil, func(hexhash string, _ nameservicev1.Commitment) (bool, error) {
		hashes = append(hashes, hexhash)
		return false, nil
	}); err != nil {
		return nil, err
	}

	return hashes, nil
}

// GetAllCommitments returns all commitments
func (k Keeper) GetAllCommitments(ctx context.Context) ([]nameservicev1.Commitment, error) {
	var commitments []nameservicev1.Commitment

	if err := k.commitments.Walk(ctx, nil, func(_ string, commitment nameservicev1.Commitment) (bool, error) {
		commitments = append(commitments, commitment)
		return false, nil
	}); err != nil {
		return nil, err
	}

	return commitments, nil
}

// GetNFTClassData gets the NFTClassData for a class
func (k Keeper) GetNFTClassData(ctx context.Context, classID string) (nameservicev1.NFTClassData, error) {
	k.Logger.Info("GetNFTClassData: Getting class data", "class_id", classID)

	class, found := k.nftKeeper.GetClass(ctx, classID)
	if !found {
		k.Logger.Error("GetNFTClassData: Class not found", "class_id", classID)
		return nameservicev1.NFTClassData{}, cosmossdkerrors.Wrapf(sdkerrors.ErrNotFound, "NFT class not found: %s", classID)
	}

	k.Logger.Info("GetNFTClassData: Found class", "class_id", classID, "name", class.Name, "has_data", class.Data != nil)

	var nftClassData nameservicev1.NFTClassData
	if class.Data != nil {
		// Instead of UnpackAny, directly unmarshal the Value bytes
		if err := k.cdc.Unmarshal(class.Data.Value, &nftClassData); err != nil {
			k.Logger.Error("GetNFTClassData: Failed to unmarshal NFT class data",
				"class_id", classID,
				"error", err,
				"type_url", class.Data.TypeUrl,
				"value_len", len(class.Data.Value))
			return nameservicev1.NFTClassData{}, cosmossdkerrors.Wrap(err, "failed to unmarshal class data")
		}

		k.Logger.Info("GetNFTClassData: Successfully unmarshaled data", "class_id", classID, "always_listed", nftClassData.AlwaysListed, "annual_pct", nftClassData.AnnualPct)

	} else {
		k.Logger.Info("GetNFTClassData: Class data is nil", "class_id", classID)
		nftClassData = nameservicev1.NFTClassData{}
	}

	return nftClassData, nil
}

// SetNFTClassData updates the class data for an NFT class
func (k Keeper) SetNFTClassData(ctx context.Context, classID string, classData nameservicev1.NFTClassData) error {
	// Get the NFT class
	class, found := k.nftKeeper.GetClass(ctx, classID)
	if !found {
		return cosmossdkerrors.Wrapf(sdkerrors.ErrNotFound, "NFT class not found: %s", classID)
	}

	// Marshal the NFT class data to an Any
	classDataAny, err := codectypes.NewAnyWithValue(&classData)
	if err != nil {
		k.Logger.Error("SetNFTClassData: Failed to marshal NFT class data", "class", classID, "error", err)
		return cosmossdkerrors.Wrap(err, "failed to marshal NFT class data")
	}

	// Update the class
	class.Data = classDataAny
	if err := k.nftKeeper.UpdateClass(ctx, class); err != nil {
		k.Logger.Error("SetNFTClassData: Failed to update NFT class data", "class", classID, "error", err)
		return cosmossdkerrors.Wrap(err, "failed to update NFT class data")
	}

	return nil
}

// SetNFTData updates the NFT data for a classId/nftId
func (k Keeper) SetNFTData(ctx context.Context, classId string, nftId string, nftData nameservicev1.NFTData) error {
	// Get the NFT
	nft, found := k.nftKeeper.GetNFT(ctx, classId, nftId)
	if !found {
		return cosmossdkerrors.Wrapf(sdkerrors.ErrNotFound, "NFT not found: %s", nftId)
	}

	// Marshal the NFT data to an Any
	nftDataAny, err := codectypes.NewAnyWithValue(&nftData)
	if err != nil {
		k.Logger.Error("SetNFTData: Failed to marshal NFT data", "class_id", classId, "nft_id", nftId, "error", err)
		return cosmossdkerrors.Wrap(err, "failed to marshal NFT data")
	}

	// Update the NFT
	nft.Data = nftDataAny
	if err := k.nftKeeper.Update(ctx, nft); err != nil {
		k.Logger.Error("SetNFTData: Failed to update NFT data", "class_id", classId, "nft_id", nftId, "error", err)
		return cosmossdkerrors.Wrap(err, "failed to update NFT data")
	}

	return nil
}

// ValidateValuation validates that a valuation coin meets all requirements:
// - not zero
// - has a denom
// - amount > 0
// - denom is in the allowed denoms list from params
func (k Keeper) ValidateValuation(ctx context.Context, valuation sdk.Coin) error {
	// Validate that the valuation is not zero
	if valuation.IsZero() {
		return cosmossdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "valuation cannot be zero")
	}

	// Validate denom is not empty
	if valuation.Denom == "" {
		return cosmossdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "valuation denom cannot be empty")
	}

	// Validate amount is greater than 0
	if valuation.Amount.IsZero() || valuation.Amount.IsNegative() {
		return cosmossdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "valuation amount must be greater than 0")
	}

	// Get module parameters
	params := k.GetParams(ctx)

	// Check if the valuation denomination is allowed
	denomAllowed := false
	for _, denom := range params.AllowedDenoms {
		if valuation.Denom == denom {
			denomAllowed = true
			break
		}
	}
	if !denomAllowed {
		return cosmossdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "valuation denomination is not in the list of allowed denoms")
	}

	return nil
}

// ComputeNameRegistrationHash computes the hash for a name registration using name, committer, and salt
func (k Keeper) ComputeNameRegistrationHash(name, committer, salt string) string {
	hash := sha256.Sum256([]byte(name + ":" + committer + ":" + salt))
	return hex.EncodeToString(hash[:])
}

// GetAuthority returns the module's authority
func (k Keeper) GetAuthority() string {
	return k.authority
}

// ResolveNameOrAddress takes a string that could be either a nameservice name or an address
// and returns the corresponding address. If it's an address, it returns it directly.
// If it's a nameservice name, it resolves it to an address.
func (k Keeper) ResolveNameOrAddress(ctx context.Context, nameOrAddress string) (string, error) {
	// Check if the input is already a valid address
	_, err := sdk.AccAddressFromBech32(nameOrAddress)
	if err == nil {
		// It's a valid address, return it as is
		return nameOrAddress, nil
	}

	// Check if it has a .dys suffix (nameservice name)
	if !strings.HasSuffix(nameOrAddress, ".dys") {
		return "", cosmossdkerrors.Wrap(sdkerrors.ErrInvalidAddress,
			fmt.Sprintf("input must be either a valid bech32 address or a name ending in .dys: got %s", nameOrAddress))
	}

	// Try to resolve it as a nameservice name NFT
	nft, found := k.nftKeeper.GetNFT(ctx, NamesClassID, nameOrAddress)
	if !found {
		return "", cosmossdkerrors.Wrap(sdkerrors.ErrNotFound,
			fmt.Sprintf("name not found: %s", nameOrAddress))
	}

	// Validate that the URI is a valid address
	_, err = sdk.AccAddressFromBech32(nft.Uri)
	if err != nil {
		return "", cosmossdkerrors.Wrap(sdkerrors.ErrInvalidAddress,
			fmt.Sprintf("name %s has invalid destination address: %s", nameOrAddress, nft.Uri))
	}

	return nft.Uri, nil
}

// ExportGenesis returns the exported genesis state as raw bytes for the gov
func (k Keeper) ExportGenesis(ctx context.Context) (*nameservicev1.GenesisState, error) {
	commitments, err := k.GetAllCommitments(ctx)
	if err != nil {
		return nil, err
	}

	gs := &nameservicev1.GenesisState{
		Params:      k.GetParams(ctx),
		Commitments: commitments,
	}
	return gs, nil
}
