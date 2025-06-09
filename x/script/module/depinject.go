package module

import (
	"cosmossdk.io/core/address"
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/event"
	"cosmossdk.io/core/store"
	"cosmossdk.io/depinject"
	"cosmossdk.io/depinject/appconfig"
	modulev1 "dysonprotocol.com/api/script/module/v1"
	"dysonprotocol.com/x/script/keeper"
	"dysonprotocol.com/x/script/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
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

type ScriptInputs struct {
	depinject.In

	App               *baseapp.BaseApp
	Config            *modulev1.Module
	StoreService      store.KVStoreService
	EventService      event.Service
	Cdc               codec.Codec
	AccountKeeper     types.AccountKeeper
	BankKeeper        types.BankKeeper
	NameserviceKeeper types.NameserviceKeeper
	AuthzKeeper       authzkeeper.Keeper
	Registry          cdctypes.InterfaceRegistry
	AddressCodec      address.Codec
	ValidatorCodec    address.Codec
	MsgServiceRouter  *baseapp.MsgServiceRouter
	QueryRouter       *baseapp.GRPCQueryRouter
}

type ScriptOutputs struct {
	depinject.Out

	ScriptKeeper keeper.Keeper
	Module       appmodule.AppModule
}

func ProvideModule(in ScriptInputs) ScriptOutputs {
	// Use the authority from the config if provided, otherwise default to gov module account
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	k := keeper.NewKeeper(
		in.App,
		in.StoreService,
		in.Cdc,
		in.AccountKeeper,
		in.NameserviceKeeper,
		in.AuthzKeeper,
		in.AddressCodec,
		in.ValidatorCodec,
		in.MsgServiceRouter,
		in.QueryRouter,
		authority,
	)

	m := NewAppModule(in.Cdc, k, in.AccountKeeper, in.BankKeeper, in.Registry)
	return ScriptOutputs{ScriptKeeper: k, Module: m}
}
