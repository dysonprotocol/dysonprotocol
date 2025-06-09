package module

import (
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/store"
	"cosmossdk.io/depinject"
	"cosmossdk.io/depinject/appconfig"
	modulev1 "dysonprotocol.com/api/storage/module/v1"
	"dysonprotocol.com/x/storage"
	"dysonprotocol.com/x/storage/keeper"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
)

var _ depinject.OnePerModuleType = AppModule{}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

func init() {
	appconfig.RegisterModule(
		&modulev1.Module{},
		appconfig.Provide(ProvideModule),
	)
}

type StorageInputs struct {
	depinject.In

	Cdc           codec.Codec
	StoreService  store.KVStoreService
	AccountKeeper authkeeper.AccountKeeper
	Registry      cdctypes.InterfaceRegistry
}

type ModuleOutputs struct {
	depinject.Out

	StorageKeeper keeper.Keeper
	Module        appmodule.AppModule
}

func ProvideModule(in StorageInputs) ModuleOutputs {
	k := keeper.NewKeeper(
		in.StoreService,
		in.Cdc,
		in.AccountKeeper,
		storage.Config{},
	)
	m := NewAppModule(in.Cdc, k, in.AccountKeeper, in.Registry)
	return ModuleOutputs{StorageKeeper: k, Module: m}
}
