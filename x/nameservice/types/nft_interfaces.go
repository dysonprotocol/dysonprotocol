package types

// NFTDataI defines an interface for any NFT data
type NFTDataI interface{}

// NFTClassDataI defines an interface for any NFT class data
type NFTClassDataI interface{}

// Ensure implementation
var (
	_ NFTDataI      = &NFTData{}
	_ NFTClassDataI = &NFTClassData{}
)
