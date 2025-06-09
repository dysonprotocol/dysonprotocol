package storage

// Config used to initialize x/group module avoiding using global variable.
type Config struct {
}

// DefaultConfig returns the default config for storage.
func DefaultConfig() Config {
	return Config{}
}
