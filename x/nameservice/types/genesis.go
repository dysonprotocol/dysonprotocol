package types

import (
	"fmt"
	"time"
)

// DefaultGenesis returns default genesis state as raw bytes for the nameservice module
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: Params{
			BidTimeout:                   time.Hour * 24 * 7, // 7 days
			AllowedDenoms:                DefaultAllowedDenoms,
			RejectBidValuationFeePercent: DefaultRejectBidValuationFeePercent, // 3%
			MinimumBidPercentIncrease:    DefaultMinimumBidPercentIncrease,    // 1%
		},
		Commitments: []Commitment{},
	}
}

// ValidateGenesis validates the provided genesis state to ensure the
// expected invariants holds.
func ValidateGenesis(data *GenesisState) error {
	// Validate params
	if err := data.Params.Validate(); err != nil {
		return err
	}

	// Validate commitments
	commitmentMap := make(map[string]bool)
	for _, commitment := range data.Commitments {
		if commitment.Hexhash == "" {
			return fmt.Errorf("empty commitment hash")
		}
		if commitmentMap[commitment.Hexhash] {
			return fmt.Errorf("duplicate commitment found: %s", commitment.Hexhash)
		}
		commitmentMap[commitment.Hexhash] = true
	}

	return nil
}
