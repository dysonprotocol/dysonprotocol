package script

import (
	"dysonprotocol.com/x/script/types"
	gogoprotoany "github.com/cosmos/gogoproto/types/any"
)

// NewGenesisState creates a new genesis state
func NewGenesisState() *types.GenesisState {
	return &types.GenesisState{
		Params: types.DefaultParams(),
	}
}

// ValidateGenesis validates the genesis state
func ValidateGenesis(s *types.GenesisState) error {
	return nil
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func UnpackGenesisInterfaces(s *types.GenesisState, unpacker gogoprotoany.AnyUnpacker) error {
	return nil
}
