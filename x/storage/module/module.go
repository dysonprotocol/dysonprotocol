package module

import (
	"context"
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	autocliv1 "cosmossdk.io/client/v2/autocli"
	"cosmossdk.io/core/appmodule"
	"dysonprotocol.com/x/storage"
	"dysonprotocol.com/x/storage/client/cli"
	"dysonprotocol.com/x/storage/keeper"
	storagetypes "dysonprotocol.com/x/storage/types"

	sdkclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const ConsensusVersion = 1

var (
	_ module.AppModuleBasic        = AppModuleBasic{}
	_ module.AppModule             = AppModule{}
	_ module.HasServices           = AppModule{}
	_ module.HasGenesis            = AppModule{}
	_ appmodule.AppModule          = AppModule{}
	_ appmodule.HasEndBlocker      = AppModule{}
	_ autocliv1.HasCustomTxCommand = AppModule{}
)

// AppModuleBasic defines the basic application module used by the storage module.
type AppModuleBasic struct{}

// Name returns the storage module's name.
func (AppModuleBasic) Name() string {
	return storage.ModuleName
}

// RegisterLegacyAminoCodec registers the storage module's types for the given codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	storage.RegisterLegacyAminoCodec(cdc)
}

// RegisterInterfaces registers the storage module's interface types
func (b AppModuleBasic) RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	storage.RegisterInterfaces(registry)
}

// DefaultGenesis returns default genesis state as raw bytes for the storage module.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(storage.DefaultGenesis())
}

// ValidateGenesis performs genesis state validation for the storage module.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config sdkclient.TxEncodingConfig, bz json.RawMessage) error {
	var data storagetypes.GenesisState
	if err := cdc.UnmarshalJSON(bz, &data); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", storage.ModuleName, err)
	}
	return storage.ValidateGenesisState(data)
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the storage module.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx sdkclient.Context, mux *gwruntime.ServeMux) {
	if err := storagetypes.RegisterQueryHandlerClient(context.Background(), mux, storagetypes.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

// GetTxCmd returns the root tx command for the storage module.
func (am AppModule) GetTxCmd() *cobra.Command {
	return cli.NewTxCmd()
}

// GetQueryCmd returns the root query command for the storage module.
func (am AppModule) GetQueryCmd() *cobra.Command {
	return cli.NewQueryCmd()
}

type AppModule struct {
	cdc      codec.Codec
	registry cdctypes.InterfaceRegistry

	keeper     keeper.Keeper
	bankKeeper storage.BankKeeper
	accKeeper  storage.AccountKeeper
}

// NewAppModule creates a new AppModule object
func NewAppModule(cdc codec.Codec, keeper keeper.Keeper, ak storage.AccountKeeper, registry cdctypes.InterfaceRegistry) AppModule {
	return AppModule{
		cdc:       cdc,
		keeper:    keeper,
		accKeeper: ak,
		registry:  registry,
	}
}

// IsAppModule implements the appmodule.AppModule interface.
func (AppModule) IsAppModule() {}

// Name returns the storage module's name.
func (am AppModule) Name() string {
	return storage.ModuleName
}

// DefaultGenesis returns default genesis state as raw bytes for the storage module.
func (am AppModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(storage.DefaultGenesis())
}

// ValidateGenesis performs genesis state validation for the storage module.
func (am AppModule) ValidateGenesis(cdc codec.JSONCodec, config sdkclient.TxEncodingConfig, bz json.RawMessage) error {
	var data storagetypes.GenesisState
	if err := cdc.UnmarshalJSON(bz, &data); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", storage.ModuleName, err)
	}
	return storage.ValidateGenesisState(data)
}

// ExportGenesis returns the exported genesis state as raw bytes for the storage module.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := am.keeper.ExportGenesis(ctx)
	return cdc.MustMarshalJSON(gs)
}

// InitGenesis initializes the storage module's state from a genesis state.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, bz json.RawMessage) {
	var data storagetypes.GenesisState
	if err := cdc.UnmarshalJSON(bz, &data); err != nil {
		panic(fmt.Errorf("failed to unmarshal %s genesis state: %w", storage.ModuleName, err))
	}
	am.keeper.InitGenesis(ctx, &data)
}

// RegisterInterfaces registers the group module's interface types
func (AppModule) RegisterInterfaces(registrar cdctypes.InterfaceRegistry) {
	storage.RegisterInterfaces(registrar)
}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(configurator module.Configurator) {
	storagetypes.RegisterMsgServer(configurator.MsgServer(), am.keeper)
	storagetypes.RegisterQueryServer(configurator.QueryServer(), am.keeper)
}

// RegisterMigrations registers module migrations
func (am AppModule) RegisterMigrations() error {
	return nil
}

// ConsensusVersion implements HasConsensusVersion
func (AppModule) ConsensusVersion() uint64 { return ConsensusVersion }

// EndBlock implements the appmodule.HasEndBlocker interface
func (am AppModule) EndBlock(ctx context.Context) error {
	return nil
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the storage module.
func (am AppModule) RegisterGRPCGatewayRoutes(clientCtx sdkclient.Context, mux *gwruntime.ServeMux) {
	if err := storagetypes.RegisterQueryHandlerClient(context.Background(), mux, storagetypes.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

// RegisterLegacyAminoCodec registers the storage module's types for the given codec.
func (AppModule) RegisterLegacyAminoCodec(registrar *codec.LegacyAmino) {
	storage.RegisterLegacyAminoCodec(registrar)
}
