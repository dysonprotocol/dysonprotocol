package keeper

import (
	"context"
	"time"

	crontasktypes "dysonprotocol.com/x/crontask/types"
)

// InitGenesis initializes the module's state from a genesis state.
func (k Keeper) InitGenesis(ctx context.Context, genState *crontasktypes.GenesisState) error {
	// Validate the genesis state
	if err := crontasktypes.ValidateGenesis(genState); err != nil {
		return err
	}

	// Set module parameters - always set them regardless of genesis state
	var moduleParams crontasktypes.Params
	if genState.Params == nil {
		// Use default params if not provided
		moduleParams = crontasktypes.DefaultParams()
	} else {
		moduleParams = *genState.Params
	}

	// Set the parameters
	if err := k.Params.Set(ctx, moduleParams); err != nil {
		return err
	}

	// Set the next task ID
	if err := k.NextTaskID.Set(ctx, genState.NextTaskId); err != nil {
		return err
	}

	// Import all tasks
	for _, task := range genState.Tasks {
		if err := k.Tasks.Set(ctx, task.TaskId, *task); err != nil {
			return err
		}
	}

	return nil
}

// ExportGenesis exports the module's state to a genesis state.
func (k Keeper) ExportGenesis(ctx context.Context) (*crontasktypes.GenesisState, error) {
	// Get params - simple error handling following SDK pattern
	params, err := k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	// Get next task ID
	nextTaskID, err := k.NextTaskID.Peek(ctx)
	if err != nil {
		return nil, err // Direct error propagation
	}

	// Get all tasks
	var tasks []*crontasktypes.Task
	if err := k.Tasks.Walk(ctx, nil, func(taskID uint64, task crontasktypes.Task) (bool, error) {
		taskCopy := task // Create a copy to avoid modifying the same memory
		tasks = append(tasks, &taskCopy)
		return false, nil
	}); err != nil {
		return nil, err // Direct error propagation
	}

	return &crontasktypes.GenesisState{
		Tasks:      tasks,
		NextTaskId: nextTaskID,
		Params:     &params,
	}, nil
}

// DefaultGenesis returns default genesis state as raw bytes for the crontask module.
func DefaultGenesis() *crontasktypes.GenesisState {
	return &crontasktypes.GenesisState{
		Params: &crontasktypes.Params{
			BlockGasLimit:    1000000000000000000,
			ExpiryLimit:      int64(time.Hour * 24 * 7), // 7 days
			MaxScheduledTime: int64(time.Hour * 24 * 7), // 7 days
		},
		Tasks:      []*crontasktypes.Task{},
		NextTaskId: 1,
	}
}
