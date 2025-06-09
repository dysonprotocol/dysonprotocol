package nameservice

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"

	nameservicev1 "dysonprotocol.com/x/nameservice/types"
)

// RegisterLegacyAminoCodec registers all the necessary nameservice module concrete
// types and interfaces with the provided codec reference.
// These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	nameservicev1.RegisterLegacyAminoCodec(cdc)
}

// RegisterInterfaces registers the interfaces types with the interface registry.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	nameservicev1.RegisterInterfaces(registry)
}
