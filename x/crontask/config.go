package crontask

// Config defines the configuration for the crontask module.
type Config struct {
	// BlockGasLimit is the maximum gas allowed for executing tasks per block
	BlockGasLimit uint64 `mapstructure:"block_gas_limit"`

	// ExpiryLimit is the default expiry limit in seconds (24 hours)
	ExpiryLimit int64 `mapstructure:"expiry_limit"`

	// MaxScheduledTime is the maximum allowed scheduled time in seconds from task creation (24 hours)
	MaxScheduledTime int64 `mapstructure:"max_scheduled_time"`

	// Add configuration parameters here as needed
}

// DefaultConfig returns the default configuration for the crontask module.
func DefaultConfig() *Config {
	return &Config{
		BlockGasLimit:    100000000, // 100M gas limit per block for tasks
		ExpiryLimit:      86400,     // 24 hours in seconds
		MaxScheduledTime: 86400,     // 24 hours in seconds
	}
}
