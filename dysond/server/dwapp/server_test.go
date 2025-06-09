package dwapp

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestServerConfig(t *testing.T) {
	testCases := []struct {
		name           string
		setupFunc      func() *Config
		expectedConfig *Config
	}{
		{
			name: "Default configuration, no custom configuration",
			setupFunc: func() *Config {
				s := &Server[sdk.Tx]{}
				return s.Config().(*Config)
			},
			expectedConfig: DefaultConfig(),
		},
		{
			name: "Custom configuration",
			setupFunc: func() *Config {
				s := NewWithConfigOptions[sdk.Tx](func(config *Config) {
					config.Enable = false
				})
				return s.Config().(*Config)
			},
			expectedConfig: &Config{
				Enable:  false, // Custom configuration
				Address: DefaultDwAppAddress,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := tc.setupFunc()
			require.Equal(t, tc.expectedConfig, config)
		})
	}
}

func TestReplaceHandlerFuncNotMatched(t *testing.T) {
	s := &Server[sdk.Tx]{}
	// ... existing code ...
}

func TestContains(t *testing.T) {
	s := NewWithConfigOptions[sdk.Tx](func(config *Config) {
		// ... existing code ...
	})
	// ... existing code ...
}
