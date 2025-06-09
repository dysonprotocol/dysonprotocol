package keeper

import (
	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	storage "dysonprotocol.com/x/storage"
	storagev1 "dysonprotocol.com/x/storage/types"
	"github.com/cosmos/cosmos-sdk/codec"
)

// We store each entry under (ownerString, indexString).
var StoragePrefix = collections.NewPrefix(0)

type Keeper struct {
	config    storage.Config
	cdc       codec.Codec
	accKeeper storage.AccountKeeper

	Schema collections.Schema

	// The robust store-level map (owner,index)->Storage
	StorageMap collections.Map[collections.Pair[string, string], storagev1.Storage]
}

func NewKeeper(
	storeService store.KVStoreService,
	cdc codec.Codec,
	accKeeper storage.AccountKeeper,
	config storage.Config,
) Keeper {
	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		config:    config,
		cdc:       cdc,
		accKeeper: accKeeper,
		StorageMap: collections.NewMap(
			sb,
			StoragePrefix, // prefix partition
			"storage_map", // name
			collections.PairKeyCodec(collections.StringKey, collections.StringKey),
			codec.CollValue[storagev1.Storage](cdc),
		),
	}
	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema

	return k
}
