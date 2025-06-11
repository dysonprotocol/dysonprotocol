package keeper

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"

	"dysonprotocol.com/x/crontask"
	crontasktypes "dysonprotocol.com/x/crontask/types"
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

	// Manual raw KV index prefixes (single-byte for simplicity)
	indexAddrPrefix      = []byte{0xA1}
	indexStatusTsPrefix  = []byte{0xA2}
	indexStatusGasPrefix = []byte{0xA3}
)

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
	Tasks collections.Map[uint64, crontasktypes.Task]

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

	// plain tasks map
	tasks := collections.NewMap(
		sb,
		TasksKey,
		"tasks",
		collections.Uint64Key,
		codec.CollValue[crontasktypes.Task](cdc),
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
	// Load the task first so we can clean up its secondary indexes. If the task
	// does not exist we simply propagate the original collections.ErrNotFound
	// so the caller can decide how to handle it.
	task, err := k.Tasks.Get(ctx, id)
	if err != nil {
		return err
	}

	// Delete secondary-index keys (address, status+timestamp, status+gasPrice)
	k.removeIndexes(ctx, task)

	// Finally remove the primary record from the `Tasks` map.
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

// GetModuleParams returns the current module parameters
func (k Keeper) GetModuleParams(ctx context.Context) crontasktypes.Params {
	return crontasktypes.Params{
		BlockGasLimit:    k.config.BlockGasLimit,
		ExpiryLimit:      k.config.ExpiryLimit,
		MaxScheduledTime: k.config.MaxScheduledTime,
	}
}

// bigEndian encodes uint64 big-endian
func bigEndian(u uint64) []byte {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], u)
	return b[:]
}

// addIndexes writes secondary-index entries for a task
func (k Keeper) addIndexes(ctx context.Context, t crontasktypes.Task) {
	store := k.storeService.OpenKVStore(ctx)

	// address index: prefix | creator | id
	keyAddr := append(append(indexAddrPrefix, []byte(t.Creator)...), bigEndian(t.TaskId)...)
	_ = store.Set(keyAddr, []byte{})

	// status+timestamp index
	tsKey := append(indexStatusTsPrefix, []byte(t.Status)...)
	tsKey = append(tsKey, bigEndian(uint64(t.ScheduledTimestamp))...)
	tsKey = append(tsKey, bigEndian(t.TaskId)...)
	_ = store.Set(tsKey, []byte{})

	// status+gasPrice index
	gpKey := append(indexStatusGasPrefix, []byte(t.Status)...)
	gpKey = append(gpKey, bigEndian(uint64(t.TaskGasPrice.Amount.Int64()))...)
	gpKey = append(gpKey, bigEndian(t.TaskId)...)
	_ = store.Set(gpKey, []byte{})
}

// removeIndexes deletes secondary-index entries for a task
func (k Keeper) removeIndexes(ctx context.Context, t crontasktypes.Task) {
	store := k.storeService.OpenKVStore(ctx)

	keyAddr := append(append(indexAddrPrefix, []byte(t.Creator)...), bigEndian(t.TaskId)...)
	_ = store.Delete(keyAddr)

	tsKey := append(indexStatusTsPrefix, []byte(t.Status)...)
	tsKey = append(tsKey, bigEndian(uint64(t.ScheduledTimestamp))...)
	tsKey = append(tsKey, bigEndian(t.TaskId)...)
	_ = store.Delete(tsKey)

	gpKey := append(indexStatusGasPrefix, []byte(t.Status)...)
	gpKey = append(gpKey, bigEndian(uint64(t.TaskGasPrice.Amount.Int64()))...)
	gpKey = append(gpKey, bigEndian(t.TaskId)...)
	_ = store.Delete(gpKey)
}

// kvStore returns module store adapter for iterator utils
func (k Keeper) kvStore(ctx context.Context) storetypes.KVStore {
	return runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
}

// iterateStatusTimestamp returns iterator over keys for given status, ordered asc/desc
func (k Keeper) iterateStatusTimestamp(ctx context.Context, status string, reverse bool) storetypes.Iterator {
	store := k.kvStore(ctx)
	prefix := append(indexStatusTsPrefix, []byte(status)...)
	if reverse {
		return storetypes.KVStoreReversePrefixIterator(store, prefix)
	}
	return storetypes.KVStorePrefixIterator(store, prefix)
}

// iterateStatusGas returns iterator over status+gasPrice index
func (k Keeper) iterateStatusGas(ctx context.Context, status string, reverse bool) storetypes.Iterator {
	store := k.kvStore(ctx)
	prefix := append(indexStatusGasPrefix, []byte(status)...)
	if reverse {
		return storetypes.KVStoreReversePrefixIterator(store, prefix)
	}
	return storetypes.KVStorePrefixIterator(store, prefix)
}

// iterateAddress returns iterator over address index
func (k Keeper) iterateAddress(ctx context.Context, addr string) storetypes.Iterator {
	store := k.kvStore(ctx)
	prefix := append(indexAddrPrefix, []byte(addr)...)
	return storetypes.KVStorePrefixIterator(store, prefix)
}
