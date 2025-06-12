package keeper

import (
	"context"
	"encoding/binary"
	"fmt"
	"runtime/debug"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	crontasktypes "dysonprotocol.com/x/crontask/types"
)

// BeginBlocker is called at the beginning of every block
func (k Keeper) BeginBlocker(ctx sdk.Context) error {
	// Get current block time directly from SDK context
	currentTime := ctx.BlockTime().Unix()
	k.Logger.Info("Current block time", "unix_time", currentTime)

	// Get module parameters to access BlockGasLimit
	params, err := k.GetParams(ctx)
	if err != nil {
		return fmt.Errorf("failed to get module params: %w", err)
	}

	// Track total gas consumed in this block
	var totalGasConsumed uint64 = 0

	// 1. expire overdue SCHEDULED tasks
	k.checkExpiredTasks(ctx, currentTime)

	// 2. move due SCHEDULED tasks to PENDING
	k.moveDueTasks(ctx, currentTime)

	// 3. process PENDING tasks ordered by gas price (desc)
	iter := k.iterateStatusGas(ctx, crontasktypes.TaskStatus_PENDING, true)
	defer iter.Close()

	var pendingIDs []uint64
	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		id := binary.BigEndian.Uint64(key[len(key)-8:])
		pendingIDs = append(pendingIDs, id)
	}

	// Execute each pending task respecting block gas limit
	for _, taskId := range pendingIDs {
		task, err := k.GetTask(ctx, taskId)
		if err != nil {
			k.Logger.Error("failed to get pending task", "task_id", taskId, "error", err)
			continue
		}

		// Check gas limit
		if totalGasConsumed+task.TaskGasLimit > params.BlockGasLimit {
			k.Logger.Info("Stopping task execution - would exceed block gas limit",
				"total_gas_consumed", totalGasConsumed,
				"task_gas_limit", task.TaskGasLimit,
				"block_gas_limit", params.BlockGasLimit,
				"task_id", taskId)
			break
		}

		// Collect fee
		gasFee := sdk.NewCoins(task.TaskGasFee)
		creatorAddr, err := sdk.AccAddressFromBech32(task.Creator)
		if err != nil {
			task.Status = crontasktypes.TaskStatus_FAILED
			task.ErrorLog = fmt.Sprintf("invalid creator address: %s", err)
			if err := k.SetTask(ctx, task); err != nil {
				k.Logger.Error("failed to set task failed due to invalid creator address", "task_id", task.TaskId, "error", err)
			}
			continue
		}

		if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, creatorAddr, "fee_collector", gasFee); err != nil {
			task.Status = crontasktypes.TaskStatus_FAILED
			task.ErrorLog = fmt.Sprintf("fee deduction failed: %s", err)
			if err := k.SetTask(ctx, task); err != nil {
				k.Logger.Error("failed to set task failed due to fee deduction failure", "task_id", task.TaskId, "error", err)
			}
			continue
		}

		// Execute
		if err := k.executeTask(ctx, &task); err != nil {
			k.Logger.Error("execution error", "task_id", taskId, "error", err)
		}

		totalGasConsumed += task.TaskGasConsumed
		k.Logger.Info("Task executed", "task_id", taskId, "gas_used", task.TaskGasConsumed)

		// Set the task data
		if err := k.SetTask(ctx, task); err != nil {
			k.Logger.Error("failed to set task", "task_id", taskId, "error", err)
		}
	}

	// 4. clean up old tasks beyond retention window
	if err := k.removeOldTasks(ctx, currentTime); err != nil {
		k.Logger.Error("failed to clean up old tasks", "error", err)
	}

	return nil
}

// checkExpiredTasks finds and marks expired tasks that haven't been executed yet
func (k Keeper) checkExpiredTasks(ctx context.Context, currentTime int64) {
	iter := k.iterateStatusTimestamp(ctx, crontasktypes.TaskStatus_SCHEDULED, false)
	defer iter.Close()

	statusPrefix := append(indexStatusTsPrefix, []byte(crontasktypes.TaskStatus_SCHEDULED)...)

	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) < len(statusPrefix)+8+8 {
			continue
		}
		expTimestamp := int64(binary.BigEndian.Uint64(key[len(statusPrefix) : len(statusPrefix)+8]))
		if expTimestamp > currentTime {
			// further tasks are scheduled in future
			break
		}
		id := binary.BigEndian.Uint64(key[len(key)-8:])

		task, err := k.GetTask(ctx, id)
		if err != nil {
			k.Logger.Error("failed to load task", "id", id, "err", err)
			continue
		}

		if task.ExpiryTimestamp <= currentTime {
			task.Status = crontasktypes.TaskStatus_EXPIRED
			task.ErrorLog = "Task expired before execution"
			if err := k.SetTask(ctx, task); err != nil {
				k.Logger.Error("failed to set task expired", "task_id", task.TaskId, "error", err)
			}
		}
	}
}

// moveDueTasks moves tasks from SCHEDULED to PENDING when their scheduled time has arrived
func (k Keeper) moveDueTasks(ctx context.Context, currentTime int64) {
	iter := k.iterateStatusTimestamp(ctx, crontasktypes.TaskStatus_SCHEDULED, false)
	defer iter.Close()

	statusPrefix := append(indexStatusTsPrefix, []byte(crontasktypes.TaskStatus_SCHEDULED)...)

	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) < len(statusPrefix)+8+8 {
			continue
		}
		scheduledTime := int64(binary.BigEndian.Uint64(key[len(statusPrefix) : len(statusPrefix)+8]))
		if scheduledTime > currentTime {
			break
		}
		id := binary.BigEndian.Uint64(key[len(key)-8:])

		task, err := k.GetTask(ctx, id)
		if err != nil {
			k.Logger.Error("failed to load task", "id", id, "err", err)
			continue
		}

		task.Status = crontasktypes.TaskStatus_PENDING
		_ = k.SetTask(ctx, task)
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
	err := k.executeMsgs(cacheCtx, task)

	// Calculate gas used and store it in the task
	gasUsed := cacheCtx.GasMeter().GasConsumed()
	task.TaskGasConsumed = gasUsed

	if err != nil {
		// Update task status to failed
		task.Status = crontasktypes.TaskStatus_FAILED
		task.ExecutionTimestamp = sdk.UnwrapSDKContext(ctx).BlockTime().Unix()

		// Log the failure details
		k.Logger.Info("Task execution failed",
			"task_id", task.TaskId,
			"gas_used", gasUsed,
			"error", err,
			"error_log", task.ErrorLog)
	} else {
		// Update task status to done and write changes
		task.Status = crontasktypes.TaskStatus_DONE
		task.ExecutionTimestamp = sdk.UnwrapSDKContext(ctx).BlockTime().Unix()
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

	// Only log success if we get here without errors
	k.Logger.Info("All messages executed successfully",
		"task_id", task.TaskId,
		"result_count", len(results))

	return nil
}

// safeInvokeMsg safely executes a message, catching any panics that might occur
func (k Keeper) safeInvokeMsg(ctx context.Context, msg sdk.Msg) (result sdk.Msg, err error) {
	// Use defer-recover pattern to catch panics
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())

			// Check for specific gas-related panics from Cosmos SDK
			switch p := r.(type) {
			case storetypes.ErrorOutOfGas:
				err = fmt.Errorf("out of gas: %s", p.Descriptor)
				k.Logger.Error("Message execution ran out of gas",
					"msg_type", sdk.MsgTypeURL(msg),
					"descriptor", p.Descriptor)
			case storetypes.ErrorGasOverflow:
				err = fmt.Errorf("gas overflow: %s", p.Descriptor)
				k.Logger.Error("Message execution caused gas overflow",
					"msg_type", sdk.MsgTypeURL(msg),
					"descriptor", p.Descriptor)
			default:
				// Log the full error with stack trace for other panics
				k.Logger.Error("Message execution panicked",
					"msg_type", sdk.MsgTypeURL(msg),
					"panic", r,
					"stack", stack)
				err = fmt.Errorf("message execution panicked: %v", r)
			}
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

// removeOldTasks deletes terminal-state tasks (DONE, FAILED, EXPIRED) whose
// execution/expiry timestamps are older than the configured CleanUpTime.
func (k Keeper) removeOldTasks(ctx context.Context, currentTime int64) error {
	params, err := k.GetParams(ctx)
	if err != nil {
		return fmt.Errorf("failed to get params: %w", err)
	}

	// If CleanUpTime is zero, feature disabled.
	if params.CleanUpTime == 0 {
		return nil
	}

	cutoff := currentTime - params.CleanUpTime

	// Terminal statuses
	statuses := []string{
		crontasktypes.TaskStatus_DONE,
		crontasktypes.TaskStatus_FAILED,
		crontasktypes.TaskStatus_EXPIRED,
	}

	for _, status := range statuses {
		iter := k.iterateStatusTimestamp(ctx, status, false) // oldest → newest
		var deleteIDs []uint64
		statusPrefix := append(indexStatusTsPrefix, []byte(status)...)

		for ; iter.Valid(); iter.Next() {
			key := iter.Key()
			if len(key) < len(statusPrefix)+8+8 {
				continue
			}
			ts := int64(binary.BigEndian.Uint64(key[len(statusPrefix) : len(statusPrefix)+8]))
			if ts > cutoff {
				// newer tasks; stop scanning further for this status
				break
			}
			id := binary.BigEndian.Uint64(key[len(key)-8:])
			deleteIDs = append(deleteIDs, id)
		}
		iter.Close()

		for _, id := range deleteIDs {
			if err := k.RemoveTask(ctx, id); err != nil {
				k.Logger.Error("failed to remove old task", "task_id", id, "error", err)
			} else {
				k.Logger.Info("old task deleted", "task_id", id, "status", status)
			}
		}
	}

	return nil
}
