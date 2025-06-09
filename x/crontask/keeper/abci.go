package keeper

import (
	"context"
	"fmt"
	"runtime/debug"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	crontasktypes "dysonprotocol.com/x/crontask/types"
)

// BeginBlocker is called at the beginning of every block
func (k Keeper) BeginBlocker(ctx sdk.Context) error {
	// Get current block time directly from SDK context
	currentTime := ctx.BlockTime().Unix()
	k.Logger.Info("Current block time", "unix_time", currentTime)

	// First check for any scheduled tasks that have expired and mark them as expired
	k.checkExpiredTasks(ctx, currentTime)

	// Process tasks with status SCHEDULED
	status := crontasktypes.TaskStatus_PENDING

	// Get tasks that are scheduled and due for execution
	iter, err := k.Indexes.ByStatusTimestamp.Iterate(ctx,
		collections.NewPrefixedPairRange[collections.Pair[string, int64], uint64](
			collections.PairPrefix[string, int64](status)))
	if err != nil {
		return fmt.Errorf("failed to iterate tasks: %w", err)
	}
	defer iter.Close()

	// Process each due task
	for ; iter.Valid(); iter.Next() {
		taskId, err := iter.PrimaryKey()
		if err != nil {
			k.Logger.Error("failed to get task ID", "error", err)
			continue
		}

		// Get the timestamp from the full key
		fullKey, err := iter.FullKey()
		if err != nil {
			k.Logger.Error("failed to get task full key", "task_id", taskId, "error", err)
			continue
		}

		// Get the task
		task, err := k.GetTask(ctx, taskId)
		if err != nil {
			k.Logger.Error("failed to get task", "task_id", taskId, "error", err)
			continue
		}

		// Skip tasks that aren't due yet
		scheduledTime := fullKey.K1().K2()
		if scheduledTime > currentTime {
			continue
		}

		// Calculate the total gas fee
		gasFee := sdk.NewCoins(task.TaskGasFee)

		// Get creator address
		creatorAddr, err := sdk.AccAddressFromBech32(task.Creator)
		if err != nil {
			k.Logger.Error("failed to parse creator address", "task_id", taskId, "creator", task.Creator, "error", err)
			task.Status = crontasktypes.TaskStatus_FAILED
			task.ErrorLog = fmt.Sprintf("failed to parse creator address: %s", err.Error())
			if err := k.SetTask(ctx, task); err != nil {
				k.Logger.Error("failed to update task status", "task_id", taskId, "error", err)
			}
			continue
		}

		// Deduct the gas fee and send it to the fee collector
		err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, creatorAddr, "fee_collector", gasFee)
		if err != nil {
			k.Logger.Error("failed to deduct gas fee", "task_id", taskId, "creator", task.Creator, "gas_fee", gasFee, "error", err)
			task.Status = crontasktypes.TaskStatus_FAILED
			task.ErrorLog = fmt.Sprintf("failed to deduct gas fee: %s", err.Error())
			if err := k.SetTask(ctx, task); err != nil {
				k.Logger.Error("failed to update task status", "task_id", taskId, "error", err)
			}
			continue
		}

		// Execute the task
		if err := k.executeTask(ctx, &task); err != nil {
			k.Logger.Error("failed to execute task", "task_id", taskId, "error", err)
		}
	}

	return nil
}

// checkExpiredTasks finds and marks expired tasks that haven't been executed yet
func (k Keeper) checkExpiredTasks(ctx context.Context, currentTime int64) {
	status := crontasktypes.TaskStatus_PENDING

	// Get all scheduled tasks
	iter, err := k.Indexes.ByStatusTimestamp.Iterate(ctx,
		collections.NewPrefixedPairRange[collections.Pair[string, int64], uint64](
			collections.PairPrefix[string, int64](status)))
	if err != nil {
		k.Logger.Error("failed to iterate tasks for expiry check", "error", err)
		return
	}
	defer iter.Close()

	// Check each scheduled task for expiry
	for ; iter.Valid(); iter.Next() {
		taskId, err := iter.PrimaryKey()
		if err != nil {
			k.Logger.Error("failed to get task ID during expiry check", "error", err)
			continue
		}

		// Get the task
		task, err := k.GetTask(ctx, taskId)
		if err != nil {
			k.Logger.Error("failed to get task during expiry check", "task_id", taskId, "error", err)
			continue
		}

		// Check if the task is expired
		if task.ExpiryTimestamp <= currentTime {
			// Mark task as expired
			task.Status = crontasktypes.TaskStatus_EXPIRED
			task.ErrorLog = "Task expired before execution"

			// Save the expired task
			if err := k.SetTask(ctx, task); err != nil {
				k.Logger.Error("failed to update task status to expired", "task_id", taskId, "error", err)
				continue
			}

			k.Logger.Info("Task marked as expired",
				"task_id", taskId,
				"scheduled_time", task.ScheduledTimestamp,
				"expiry_time", task.ExpiryTimestamp,
				"current_time", currentTime)
		}
	}
}

// executeTask executes a task and updates its status based on the result
func (k Keeper) executeTask(ctx context.Context, task *crontasktypes.Task) error {
	// Reset results fields
	task.MsgResults = nil
	task.ErrorLog = ""

	// Create a cache context with gas meter
	cacheCtx, write := sdk.UnwrapSDKContext(ctx).CacheContext()
	cacheCtx = cacheCtx.WithGasMeter(storetypes.NewGasMeter(task.TaskGasLimit))

	k.Logger.Info("Executing task",
		"task_id", task.TaskId,
		"gas_limit", task.TaskGasLimit,
		"msg_count", len(task.Msgs))

	// Execute messages
	err := k.executeMsgs(sdk.WrapSDKContext(cacheCtx), task)

	// Calculate gas used
	gasUsed := cacheCtx.GasMeter().GasConsumed()

	if err != nil {
		// Update task status to failed
		task.Status = crontasktypes.TaskStatus_FAILED

		// Log the failure details
		k.Logger.Info("Task execution failed",
			"task_id", task.TaskId,
			"gas_used", gasUsed,
			"error", err,
			"error_log", task.ErrorLog)
	} else {
		// Update task status to done and write changes
		task.Status = crontasktypes.TaskStatus_DONE
		write() // Write changes to parent context

		// Get result count for logging
		resultCount := len(task.MsgResults)

		// Log the success details
		k.Logger.Info("Task execution successful",
			"task_id", task.TaskId,
			"gas_used", gasUsed,
			"results_count", resultCount)
	}

	// Save the updated task
	if saveErr := k.SetTask(ctx, *task); saveErr != nil {
		return fmt.Errorf("failed to save task after execution: %w", saveErr)
	}

	// Return the original execution error, if any
	return err
}

// executeMsgs processes all messages in a task
func (k Keeper) executeMsgs(ctx context.Context, task *crontasktypes.Task) error {
	// Get messages from task using the GetMessages method
	msgs, err := task.GetMessages()
	if err != nil {
		task.ErrorLog = fmt.Sprintf("failed to unpack messages: %s", err.Error())
		return fmt.Errorf("failed to unpack messages: %w", err)
	}

	// Create a slice to collect successful results
	var results []sdk.Msg

	for i, msg := range msgs {
		// Safely invoke the message
		resultMsg, err := k.safeInvokeMsg(ctx, msg)
		if err != nil {
			// Set error in the task
			task.ErrorLog = fmt.Sprintf("message at index %d failed: %s", i, err.Error())

			// Return detailed error
			return fmt.Errorf("message at index %d failed: %w", i, err)
		}

		// Add the result to our collection (if not nil)
		if resultMsg != nil {
			results = append(results, resultMsg)
		}
	}

	// Set all results at once using the new SetMessageResults method
	if len(results) > 0 {
		if err := task.SetMessageResults(results); err != nil {
			task.ErrorLog = fmt.Sprintf("failed to set message results: %s", err.Error())
			return fmt.Errorf("failed to set message results: %w", err)
		}
	} else {
		// Ensure empty results
		task.MsgResults = nil
	}

	k.Logger.Info("All messages executed successfully",
		"task_id", task.TaskId,
		"result_count", len(results))

	return nil
}

// safeInvokeMsg safely executes a message, catching any panics that might occur
func (k Keeper) safeInvokeMsg(ctx context.Context, msg sdk.Msg) (sdk.Msg, error) {
	var result sdk.Msg
	var err error

	// Use defer-recover pattern to catch panics
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())

			// Log the full error with stack trace
			k.Logger.Error("Message execution panicked",
				"msg_type", sdk.MsgTypeURL(msg),
				"panic", r,
				"stack", stack)

			err = fmt.Errorf("message execution panicked: %v", r)
		}
	}()

	// Add debug logging to check if MsgRouterService is nil
	msgType := sdk.MsgTypeURL(msg)
	k.Logger.Info("Attempting to invoke message",
		"msg_type", msgType,
		"msg_router_is_nil", k.MsgRouterService == nil)

	// Invoke the message handler - use `HandleDeliver` method
	handler := k.MsgRouterService.Handler(msg)
	if handler == nil {
		return nil, fmt.Errorf("no handler found for message type: %s", sdk.MsgTypeURL(msg))
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	msgResult, err := handler(sdkCtx, msg)
	if err != nil {
		return nil, err
	}

	if msgResult != nil && msgResult.MsgResponses != nil && len(msgResult.MsgResponses) > 0 {
		resp, ok := msgResult.MsgResponses[0].GetCachedValue().(sdk.Msg)
		if ok {
			result = resp
		}
	}

	return result, nil
}
