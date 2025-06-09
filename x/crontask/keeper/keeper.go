package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"dysonprotocol.com/x/crontask"
	crontasktypes "dysonprotocol.com/x/crontask/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
)

var (
	// TasksKey is the prefix for the tasks collection
	TasksKey = collections.NewPrefix(0)

	// NextTaskIDKey is the key for the next task ID
	NextTaskIDKey = collections.NewPrefix(1)

	// ParamsKey is the prefix for the module parameters
	ParamsKey = collections.NewPrefix(2)

	// Task index prefixes
	TasksByAddressPrefix         = collections.NewPrefix(3)
	TasksByStatusTimestampPrefix = collections.NewPrefix(4)
	TasksByStatusGasPricePrefix  = collections.NewPrefix(5)
)

// TaskIndexes defines the indexes for tasks
type TaskIndexes struct {
	// ByAddress indexes tasks by creator address
	ByAddress *indexes.Multi[string, uint64, crontasktypes.Task]

	// ByStatusTimestamp indexes tasks by status and scheduled timestamp
	ByStatusTimestamp *indexes.Multi[collections.Pair[string, int64], uint64, crontasktypes.Task]

	// ByStatusGasPrice indexes tasks by status and gas price
	ByStatusGasPrice *indexes.Multi[collections.Pair[string, int64], uint64, crontasktypes.Task]
}

// NewTaskIndexes creates new indexes for tasks
func NewTaskIndexes(sb *collections.SchemaBuilder, tasks collections.Map[uint64, crontasktypes.Task], cdc codec.Codec) TaskIndexes {
	return TaskIndexes{
		// ByAddress indexes tasks by creator address
		ByAddress: indexes.NewMulti(
			sb,
			TasksByAddressPrefix,
			"tasks_by_address",
			collections.StringKey,
			collections.Uint64Key,
			func(_ uint64, task crontasktypes.Task) (string, error) {
				return task.Creator, nil
			},
		),

		// Index by status and timestamp
		ByStatusTimestamp: indexes.NewMulti(
			sb,
			TasksByStatusTimestampPrefix,
			"tasks_by_status_timestamp",
			collections.PairKeyCodec(collections.StringKey, collections.Int64Key),
			collections.Uint64Key,
			func(_ uint64, task crontasktypes.Task) (collections.Pair[string, int64], error) {
				return collections.Join(task.Status, task.ScheduledTimestamp), nil
			},
		),

		// ByStatusGasPrice indexes tasks by status and gas price
		ByStatusGasPrice: indexes.NewMulti(
			sb,
			TasksByStatusGasPricePrefix,
			"tasks_by_status_gas_price",
			collections.PairKeyCodec(collections.StringKey, collections.Int64Key),
			collections.Uint64Key,
			func(_ uint64, task crontasktypes.Task) (collections.Pair[string, int64], error) {
				// Convert gas price to Int64 for indexing
				return collections.Join(task.Status, task.TaskGasPrice.Amount.Int64()), nil
			},
		),
	}
}

type Keeper struct {
	cdc           codec.Codec
	storeService  store.KVStoreService
	bankKeeper    crontasktypes.BankKeeper
	accountKeeper crontasktypes.AccountKeeper
	config        crontask.Config

	// Services from the app's depinject setup
	MsgRouterService *baseapp.MsgServiceRouter
	Logger           log.Logger

	Schema collections.Schema

	// Tasks is the primary collection for tasks
	Tasks *collections.IndexedMap[uint64, crontasktypes.Task, TaskIndexes]

	// Indexes provides efficient queries over the Tasks collection
	Indexes TaskIndexes

	// NextTaskID is a sequence for task IDs
	NextTaskID collections.Sequence

	// Params stores module parameters
	Params collections.Item[crontasktypes.Params]
}

// NewKeeper creates a new crontask Keeper instance
func NewKeeper(
	cdc codec.Codec,
	storeService store.KVStoreService,
	accountKeeper crontasktypes.AccountKeeper,
	bankKeeper crontasktypes.BankKeeper,
	msgRouter *baseapp.MsgServiceRouter,
	config crontask.Config,
	logger log.Logger,
) Keeper {
	// Add the module name to the logger
	logger = logger.With(log.ModuleKey, "x/"+crontask.ModuleName)

	sb := collections.NewSchemaBuilder(storeService)

	// Create the task indexes
	taskIndexes := TaskIndexes{
		// ByAddress indexes tasks by creator address
		ByAddress: indexes.NewMulti(
			sb,
			TasksByAddressPrefix,
			"tasks_by_address",
			collections.StringKey,
			collections.Uint64Key,
			func(taskId uint64, task crontasktypes.Task) (string, error) {
				return task.Creator, nil
			},
		),

		// Index by status and timestamp
		ByStatusTimestamp: indexes.NewMulti(
			sb,
			TasksByStatusTimestampPrefix,
			"tasks_by_status_timestamp",
			collections.PairKeyCodec(collections.StringKey, collections.Int64Key),
			collections.Uint64Key,
			func(taskId uint64, task crontasktypes.Task) (collections.Pair[string, int64], error) {
				return collections.Join(task.Status, task.ScheduledTimestamp), nil
			},
		),

		// ByStatusGasPrice indexes tasks by status and gas price
		ByStatusGasPrice: indexes.NewMulti(
			sb,
			TasksByStatusGasPricePrefix,
			"tasks_by_status_gas_price",
			collections.PairKeyCodec(collections.StringKey, collections.Int64Key),
			collections.Uint64Key,
			func(taskId uint64, task crontasktypes.Task) (collections.Pair[string, int64], error) {
				// Convert gas price to Int64 for indexing
				return collections.Join(task.Status, task.TaskGasPrice.Amount.Int64()), nil
			},
		),
	}

	// Create indexed map with the indexes
	tasks := collections.NewIndexedMap(
		sb,
		TasksKey,
		"tasks",
		collections.Uint64Key,
		codec.CollValue[crontasktypes.Task](cdc),
		taskIndexes,
	)

	nextTaskID := collections.NewSequence(
		sb,
		NextTaskIDKey,
		"next_task_id",
	)

	// Create a Item params item for parameters storage
	params := collections.NewItem(
		sb,
		ParamsKey,
		"params",
		codec.CollValue[crontasktypes.Params](cdc),
	)

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}

	keeper := Keeper{
		cdc:              cdc,
		storeService:     storeService,
		bankKeeper:       bankKeeper,
		accountKeeper:    accountKeeper,
		config:           config,
		Tasks:            tasks,
		Indexes:          taskIndexes,
		NextTaskID:       nextTaskID,
		Params:           params,
		Schema:           schema,
		Logger:           logger,
		MsgRouterService: msgRouter,
	}

	// Add debug logging about keeper initialization
	logger.Debug("Crontask keeper initialized",
		"msg_router_service_set", keeper.MsgRouterService != nil,
		"bank_keeper_set", keeper.bankKeeper != nil,
		"account_keeper_set", keeper.accountKeeper != nil)

	return keeper
}

// GetNextTaskID gets and increments the global task ID counter
func (k Keeper) GetNextTaskID(ctx context.Context) (uint64, error) {
	return k.NextTaskID.Next(ctx)
}

// SetTask sets a task in the store
func (k Keeper) SetTask(ctx context.Context, task crontasktypes.Task) error {
	return k.Tasks.Set(ctx, task.TaskId, task)
}

// GetTask gets a task by ID
func (k Keeper) GetTask(ctx context.Context, id uint64) (crontasktypes.Task, error) {
	return k.Tasks.Get(ctx, id)
}

// DeleteTask deletes a task from the store
func (k Keeper) RemoveTask(ctx context.Context, id uint64) error {
	return k.Tasks.Remove(ctx, id)
}

// SetParams sets the crontask module parameters
func (k Keeper) SetParams(ctx context.Context, params crontasktypes.Params) error {
	fmt.Printf("SetParams called with: BlockGasLimit=%d, ExpiryLimit=%d, MaxScheduledTime=%d\n",
		params.BlockGasLimit, params.ExpiryLimit, params.MaxScheduledTime)

	// Validate parameters before attempting to set them
	if err := params.Validate(); err != nil {
		fmt.Printf("SetParams validation error: %v\n", err)
		return fmt.Errorf("invalid parameters: %w", err)
	}

	err := k.Params.Set(ctx, params)
	if err != nil {
		fmt.Printf("SetParams error when setting params: %v\n", err)
		return err
	}

	fmt.Printf("SetParams completed successfully\n")
	return nil
}

// GetParams gets the crontask module parameters
func (k Keeper) GetParams(ctx context.Context) (crontasktypes.Params, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		// For collections.ErrNotFound, return empty params similar to SDK modules
		if errors.Is(err, collections.ErrNotFound) {
			return crontasktypes.DefaultParams(), nil
		}
		// For any other error, propagate it upward
		return crontasktypes.Params{}, err
	}
	return params, nil
}

// GetExpiredTasks returns tasks with the EXPIRED status
func (k Keeper) GetExpiredTasks(ctx context.Context, limit int) ([]crontasktypes.Task, error) {
	status := crontasktypes.TaskStatus_EXPIRED

	// Iterate through tasks with EXPIRED status
	iter, err := k.Indexes.ByStatusTimestamp.Iterate(ctx,
		collections.NewPrefixedPairRange[collections.Pair[string, int64], uint64](
			collections.PairPrefix[string, int64](status)))
	if err != nil {
		return nil, fmt.Errorf("failed to iterate expired tasks: %w", err)
	}
	defer iter.Close()

	tasks := make([]crontasktypes.Task, 0, limit)
	count := 0

	// Collect expired tasks up to the limit
	for ; iter.Valid() && count < limit; iter.Next() {
		taskId, err := iter.PrimaryKey()
		if err != nil {
			k.Logger.Error("failed to get task ID", "error", err)
			continue
		}

		task, err := k.GetTask(ctx, taskId)
		if err != nil {
			k.Logger.Error("failed to get task", "task_id", taskId, "error", err)
			continue
		}

		tasks = append(tasks, task)
		count++
	}

	return tasks, nil
}

// GetModuleParams returns the current module parameters
func (k Keeper) GetModuleParams(ctx context.Context) crontasktypes.Params {
	return crontasktypes.Params{
		BlockGasLimit:    k.config.BlockGasLimit,
		ExpiryLimit:      k.config.ExpiryLimit,
		MaxScheduledTime: k.config.MaxScheduledTime,
	}
}
