package keeper

import (
	"context"
	"encoding/json"

	"dysonprotocol.com/x/script/types"
	"github.com/cosmos/cosmos-sdk/codec"
)

func (k Keeper) InitGenesis(ctx context.Context, cdc codec.JSONCodec, data json.RawMessage) error {
	var genesisState types.GenesisState
	err := cdc.UnmarshalJSON(data, &genesisState)
	if err != nil {
		return err
	}

	// Set the parameters
	if err := k.SetParams(ctx, genesisState.Params); err != nil {
		return err
	}

	// Iterate through all the scripts in the genesis state and set them
	for _, s := range genesisState.Scripts {
		// Set the script in the store
		err = k.ScriptMap.Set(ctx, s.Address, *s)
		if err != nil {
			return err
		}
	}

	return nil
}

func (k Keeper) ExportGenesis(ctx context.Context, _ codec.JSONCodec) (*types.GenesisState, error) {
	// Get the current parameters
	params := k.GetParams(ctx)

	scripts := []*types.Script{}

	// Iterate through all scripts and add them to the slice
	scriptRange, err := k.ScriptMap.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer scriptRange.Close()

	for ; scriptRange.Valid(); scriptRange.Next() {
		value, err := scriptRange.Value()
		if err != nil {
			// Skip entries that can't be read (shouldn't happen)
			continue
		}
		scripts = append(scripts, &value)
	}

	// Create and return a new genesis state with the params and scripts
	return &types.GenesisState{
		Params:  params,
		Scripts: scripts,
	}, nil
}
