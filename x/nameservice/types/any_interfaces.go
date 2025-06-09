package types

import (
	gogoprotoany "github.com/cosmos/gogoproto/types/any"
)

// Ensure our types implement the UnpackInterfacesMessage interface
var (
	_ gogoprotoany.UnpackInterfacesMessage = &NFTClassData{}
	_ gogoprotoany.UnpackInterfacesMessage = &NFTData{}
)

// UnpackInterfaces implements the UnpackInterfacesMessage.UnpackInterfaces method
func (m NFTClassData) UnpackInterfaces(unpacker gogoprotoany.AnyUnpacker) error {
	// NFTClassData doesn't contain Any fields, so nothing to unpack
	return nil
}

// UnpackInterfaces implements the UnpackInterfacesMessage.UnpackInterfaces method
func (m NFTData) UnpackInterfaces(unpacker gogoprotoany.AnyUnpacker) error {
	// NFTData doesn't contain Any fields, so nothing to unpack
	return nil
}
