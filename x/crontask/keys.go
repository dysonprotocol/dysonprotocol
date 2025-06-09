package crontask

// ModuleName defines the module name
const ModuleName = "crontask"

// These keys are used in the keeper's KVStore
const (
	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// QuerierRoute defines the module's query routing key
	QuerierRoute = ModuleName
)
