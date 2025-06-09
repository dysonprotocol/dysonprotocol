package types

var (
	// KeyPrefixNameRecord defines prefix for storing name records
	KeyPrefixNameRecord = []byte{0x01}

	// KeyPrefixCommitment defines prefix for storing name registration commitments
	KeyPrefixCommitment = []byte{0x02}

	// KeyPrefixBid defines prefix for storing bids
	KeyPrefixBid = []byte{0x03}

	// ParamsStoreKey is the store key for module parameters
	ParamsStoreKey = []byte{0x04}
)

// GetNameKey returns the store key to retrieve a NameRecord by name
func GetNameKey(name string) []byte {
	return append(KeyPrefixNameRecord, []byte(name)...)
}

// GetCommitmentKey returns the store key to retrieve a commitment by hash
func GetCommitmentKey(hash []byte) []byte {
	return append(KeyPrefixCommitment, hash...)
}

// GetBidKey returns the store key to retrieve a bid by name
func GetBidKey(name string) []byte {
	return append(KeyPrefixBid, []byte(name)...)
}
