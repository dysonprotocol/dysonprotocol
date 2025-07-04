package dysonprotocol

import (
	runtimev1alpha1 "cosmossdk.io/api/cosmos/app/runtime/v1alpha1"
	appv1alpha1 "cosmossdk.io/api/cosmos/app/v1alpha1"
	authmodulev1 "cosmossdk.io/api/cosmos/auth/module/v1"
	authzmodulev1 "cosmossdk.io/api/cosmos/authz/module/v1"
	bankmodulev1 "cosmossdk.io/api/cosmos/bank/module/v1"
	circuitmodulev1 "cosmossdk.io/api/cosmos/circuit/module/v1"
	consensusmodulev1 "cosmossdk.io/api/cosmos/consensus/module/v1"
	distrmodulev1 "cosmossdk.io/api/cosmos/distribution/module/v1"
	epochsmodulev1 "cosmossdk.io/api/cosmos/epochs/module/v1"
	evidencemodulev1 "cosmossdk.io/api/cosmos/evidence/module/v1"
	feegrantmodulev1 "cosmossdk.io/api/cosmos/feegrant/module/v1"
	genutilmodulev1 "cosmossdk.io/api/cosmos/genutil/module/v1"
	govmodulev1 "cosmossdk.io/api/cosmos/gov/module/v1"
	mintmodulev1 "cosmossdk.io/api/cosmos/mint/module/v1"
	nftmodulev1 "cosmossdk.io/api/cosmos/nft/module/v1"
	protocolpoolmodulev1 "cosmossdk.io/api/cosmos/protocolpool/module/v1"
	slashingmodulev1 "cosmossdk.io/api/cosmos/slashing/module/v1"
	stakingmodulev1 "cosmossdk.io/api/cosmos/staking/module/v1"
	txconfigv1 "cosmossdk.io/api/cosmos/tx/config/v1"
	upgrademodulev1 "cosmossdk.io/api/cosmos/upgrade/module/v1"
	vestingmodulev1 "cosmossdk.io/api/cosmos/vesting/module/v1"
	"cosmossdk.io/depinject"
	"cosmossdk.io/depinject/appconfig"
	_ "cosmossdk.io/x/circuit" // import for side-effects
	circuittypes "cosmossdk.io/x/circuit/types"
	_ "cosmossdk.io/x/evidence" // import for side-effects
	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	_ "cosmossdk.io/x/feegrant/module" // import for side-effects
	"cosmossdk.io/x/nft"
	_ "cosmossdk.io/x/nft/module" // import for side-effects
	_ "cosmossdk.io/x/upgrade"    // import for side-effects
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/module"
	_ "github.com/cosmos/cosmos-sdk/x/auth/tx/config" // import for side-effects
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	_ "github.com/cosmos/cosmos-sdk/x/auth/vesting" // import for side-effects
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	_ "github.com/cosmos/cosmos-sdk/x/authz/module" // import for side-effects
	_ "github.com/cosmos/cosmos-sdk/x/bank"         // import for side-effects
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	_ "github.com/cosmos/cosmos-sdk/x/consensus" // import for side-effects
	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	_ "github.com/cosmos/cosmos-sdk/x/distribution" // import for side-effects
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	_ "github.com/cosmos/cosmos-sdk/x/epochs" // import for side-effects
	epochstypes "github.com/cosmos/cosmos-sdk/x/epochs/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	_ "github.com/cosmos/cosmos-sdk/x/mint" // import for side-effects
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	_ "github.com/cosmos/cosmos-sdk/x/protocolpool" // import for side-effects
	protocolpooltypes "github.com/cosmos/cosmos-sdk/x/protocolpool/types"
	_ "github.com/cosmos/cosmos-sdk/x/slashing" // import for side-effects
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	_ "github.com/cosmos/cosmos-sdk/x/staking" // import for side-effects
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	// Custom modules
	crontaskmodulev1 "dysonprotocol.com/api/crontask/module/v1"
	crontaskv1 "dysonprotocol.com/x/crontask"
	crontaskmodule "dysonprotocol.com/x/crontask/module"

	nameservicemodulev1 "dysonprotocol.com/api/nameservice/module/v1"
	nameservicev1 "dysonprotocol.com/x/nameservice"
	nameservicemodule "dysonprotocol.com/x/nameservice/module"

	scriptmodulev1 "dysonprotocol.com/api/script/module/v1"
	scriptv1 "dysonprotocol.com/x/script"
	scriptmodule "dysonprotocol.com/x/script/module"

	storagemodulev1 "dysonprotocol.com/api/storage/module/v1"
	storagev1 "dysonprotocol.com/x/storage"
	storagemodule "dysonprotocol.com/x/storage/module"
)

var (
	// module account permissions
	moduleAccPerms = []*authmodulev1.ModuleAccountPermission{
		{Account: authtypes.FeeCollectorName},
		{Account: distrtypes.ModuleName},
		{Account: minttypes.ModuleName, Permissions: []string{authtypes.Minter}},
		{Account: stakingtypes.BondedPoolName, Permissions: []string{authtypes.Burner, stakingtypes.ModuleName}},
		{Account: stakingtypes.NotBondedPoolName, Permissions: []string{authtypes.Burner, stakingtypes.ModuleName}},
		{Account: govtypes.ModuleName, Permissions: []string{authtypes.Burner}},
		{Account: nft.ModuleName},
		{Account: protocolpooltypes.ModuleName},
		{Account: protocolpooltypes.ProtocolPoolEscrowAccount},
		// Custom modules
		{Account: nameservicev1.ModuleName, Permissions: []string{authtypes.Minter, authtypes.Burner}},
		{Account: scriptv1.ModuleName},
		{Account: storagev1.ModuleName},
		{Account: crontaskv1.ModuleName},
	}

	// blocked account addresses
	blockAccAddrs = []string{
		authtypes.FeeCollectorName,
		distrtypes.ModuleName,
		minttypes.ModuleName,
		stakingtypes.BondedPoolName,
		stakingtypes.NotBondedPoolName,
		nft.ModuleName,
		// Custom modules
		nameservicev1.ModuleName,
		scriptv1.ModuleName,
		storagev1.ModuleName,
		crontaskv1.ModuleName,
		// We allow the following module accounts to receive funds:
		// govtypes.ModuleName
	}

	ModuleConfig = []*appv1alpha1.ModuleConfig{
		{
			Name: runtime.ModuleName,
			Config: appconfig.WrapAny(&runtimev1alpha1.Module{
				AppName: "DysApp",
				// NOTE: upgrade module is required to be prioritized
				PreBlockers: []string{
					upgradetypes.ModuleName,
					authtypes.ModuleName,
				},
				// During begin block slashing happens after distr.BeginBlocker so that
				// there is nothing left over in the validator fee pool, so as to keep the
				// CanWithdrawInvariant invariant.
				// NOTE: staking module is required if HistoricalEntries param > 0
				BeginBlockers: []string{
					minttypes.ModuleName,
					distrtypes.ModuleName,
					protocolpooltypes.ModuleName,
					slashingtypes.ModuleName,
					evidencetypes.ModuleName,
					stakingtypes.ModuleName,
					authz.ModuleName,
					epochstypes.ModuleName,
					// Custom modules
					nameservicev1.ModuleName,
					scriptv1.ModuleName,
					storagev1.ModuleName,
					crontaskv1.ModuleName,
				},
				EndBlockers: []string{
					govtypes.ModuleName,
					stakingtypes.ModuleName,
					feegrant.ModuleName,
					protocolpooltypes.ModuleName,
					// Custom modules
					nameservicev1.ModuleName,
					scriptv1.ModuleName,
					storagev1.ModuleName,
					crontaskv1.ModuleName,
				},
				OverrideStoreKeys: []*runtimev1alpha1.StoreKeyConfig{
					{
						ModuleName: authtypes.ModuleName,
						KvStoreKey: "acc",
					},
				},
				// NOTE: The genutils module must occur after staking so that pools are
				// properly initialized with tokens from genesis accounts.
				// NOTE: The genutils module must also occur after auth so that it can access the params from auth.
				InitGenesis: []string{
					authtypes.ModuleName,
					banktypes.ModuleName,
					distrtypes.ModuleName,
					stakingtypes.ModuleName,
					slashingtypes.ModuleName,
					govtypes.ModuleName,
					minttypes.ModuleName,
					genutiltypes.ModuleName,
					evidencetypes.ModuleName,
					authz.ModuleName,
					feegrant.ModuleName,
					nft.ModuleName,
					upgradetypes.ModuleName,
					vestingtypes.ModuleName,
					circuittypes.ModuleName,
					epochstypes.ModuleName,
					protocolpooltypes.ModuleName,
					// Custom modules
					nameservicev1.ModuleName,
					scriptv1.ModuleName,
					storagev1.ModuleName,
					crontaskv1.ModuleName,
				},
				// When ExportGenesis is not specified, the export genesis module order
				// is equal to the init genesis order
				ExportGenesis: []string{
					consensustypes.ModuleName,
					authtypes.ModuleName,
					protocolpooltypes.ModuleName, // Must be exported before bank
					banktypes.ModuleName,
					distrtypes.ModuleName,
					stakingtypes.ModuleName,
					slashingtypes.ModuleName,
					govtypes.ModuleName,
					minttypes.ModuleName,
					genutiltypes.ModuleName,
					evidencetypes.ModuleName,
					authz.ModuleName,
					feegrant.ModuleName,
					nft.ModuleName,
					upgradetypes.ModuleName,
					vestingtypes.ModuleName,
					circuittypes.ModuleName,
					epochstypes.ModuleName,
					// Custom modules
					nameservicev1.ModuleName,
					scriptv1.ModuleName,
					storagev1.ModuleName,
					crontaskv1.ModuleName,
				},
				// Uncomment if you want to set a custom migration order here.
				// OrderMigrations: []string{},
			}),
		},
		{
			Name: authtypes.ModuleName,
			Config: appconfig.WrapAny(&authmodulev1.Module{
				Bech32Prefix:             "dys",
				ModuleAccountPermissions: moduleAccPerms,
				// By default modules authority is the governance module. This is configurable with the following:
				// Authority: "group", // A custom module authority can be set using a module name
				// Authority: "cosmos1cwwv22j5ca08ggdv9c2uky355k908694z577tv", // or a specific address
			}),
		},
		{
			Name:   vestingtypes.ModuleName,
			Config: appconfig.WrapAny(&vestingmodulev1.Module{}),
		},
		{
			Name: banktypes.ModuleName,
			Config: appconfig.WrapAny(&bankmodulev1.Module{
				BlockedModuleAccountsOverride: blockAccAddrs,
			}),
		},
		{
			Name:   stakingtypes.ModuleName,
			Config: appconfig.WrapAny(&stakingmodulev1.Module{}),
		},
		{
			Name:   slashingtypes.ModuleName,
			Config: appconfig.WrapAny(&slashingmodulev1.Module{}),
		},
		{
			Name: "tx",
			Config: appconfig.WrapAny(&txconfigv1.Config{
				SkipAnteHandler: true, // Enable this to skip the default antehandlers and set custom ante handlers.
			}),
		},
		{
			Name:   genutiltypes.ModuleName,
			Config: appconfig.WrapAny(&genutilmodulev1.Module{}),
		},
		{
			Name:   authz.ModuleName,
			Config: appconfig.WrapAny(&authzmodulev1.Module{}),
		},
		{
			Name:   upgradetypes.ModuleName,
			Config: appconfig.WrapAny(&upgrademodulev1.Module{}),
		},
		{
			Name:   distrtypes.ModuleName,
			Config: appconfig.WrapAny(&distrmodulev1.Module{}),
		},
		{
			Name:   evidencetypes.ModuleName,
			Config: appconfig.WrapAny(&evidencemodulev1.Module{}),
		},
		{
			Name:   minttypes.ModuleName,
			Config: appconfig.WrapAny(&mintmodulev1.Module{}),
		},
		{
			Name:   feegrant.ModuleName,
			Config: appconfig.WrapAny(&feegrantmodulev1.Module{}),
		},
		{
			Name:   govtypes.ModuleName,
			Config: appconfig.WrapAny(&govmodulev1.Module{}),
		},
		{
			Name:   consensustypes.ModuleName,
			Config: appconfig.WrapAny(&consensusmodulev1.Module{}),
		},
		{
			Name:   circuittypes.ModuleName,
			Config: appconfig.WrapAny(&circuitmodulev1.Module{}),
		},
		{
			Name:   epochstypes.ModuleName,
			Config: appconfig.WrapAny(&epochsmodulev1.Module{}),
		},
		{
			Name:   protocolpooltypes.ModuleName,
			Config: appconfig.WrapAny(&protocolpoolmodulev1.Module{}),
		},
		// Custom modules
		{
			Name:   scriptv1.ModuleName,
			Config: appconfig.WrapAny(&scriptmodulev1.Module{}),
		},
		{
			Name: nameservicev1.ModuleName,
			Config: appconfig.WrapAny(&nameservicemodulev1.Module{
				MaxNameLength:     100,
				MaxMetadataLength: 1000,
			}),
		},
		{
			Name:   storagev1.ModuleName,
			Config: appconfig.WrapAny(&storagemodulev1.Module{}),
		},
		{
			Name:   crontaskv1.ModuleName,
			Config: appconfig.WrapAny(&crontaskmodulev1.Module{}),
		},
		{
			Name:   nft.ModuleName,
			Config: appconfig.WrapAny(&nftmodulev1.Module{}),
		},
		// since they don't have proto-based configs
	}

	// AppConfig is application configuration (used by depinject)
	AppConfig = depinject.Configs(appconfig.Compose(&appv1alpha1.Config{
		Modules: ModuleConfig,
	}),
		depinject.Supply(
			// supply custom module basics
			map[string]module.AppModuleBasic{
				genutiltypes.ModuleName: genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
				govtypes.ModuleName: gov.NewAppModuleBasic(
					[]govclient.ProposalHandler{},
				),
				// Add explicit references to our custom modules to ensure they're registered
				scriptv1.ModuleName:      scriptmodule.AppModuleBasic{},
				nameservicev1.ModuleName: nameservicemodule.AppModuleBasic{},
				storagev1.ModuleName:     storagemodule.AppModuleBasic{},
				crontaskv1.ModuleName:    crontaskmodule.AppModuleBasic{},
			},
		),
	)
)
