package module

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	crontaskv1 "dysonprotocol.com/api/crontask/types"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: crontaskv1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "TaskByID",
					Use:       "task-by-id --task-id <task-id>",
					Short:     "Query a task by ID",
					Long:      "Query a task by its unique identifier",
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"task_id": {
							Name:  "task-id",
							Usage: "The ID of the task to query",
						},
					},
				},
				{
					RpcMethod: "TasksByAddress",
					Use:       "tasks-by-address --creator <creator-address>",
					Short:     "Query tasks by creator address",
					Long:      "Query all tasks created by a specific address",
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"creator": {
							Name:  "creator",
							Usage: "The creator address to filter tasks by",
						},
					},
				},
				{
					RpcMethod: "TasksByStatusTimestamp",
					Use:       "tasks-by-status-timestamp --status <status>",
					Short:     "Query tasks by status and timestamp",
					Long:      "Query tasks filtered by status and ordered by timestamp",
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"status": {
							Name:  "status",
							Usage: "The status to filter tasks by",
						},
					},
				},
				{
					RpcMethod: "TasksByStatusGasPrice",
					Use:       "tasks-by-status-gas-price --status <status>",
					Short:     "Query tasks by status and gas price",
					Long:      "Query tasks filtered by status and ordered by gas price",
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"status": {
							Name:  "status",
							Usage: "The status to filter tasks by",
						},
					},
				},
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Query module parameters",
					Long:      "Query the current crontask module parameters",
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service: crontaskv1.Msg_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "CreateTask",
					Use:       "create-task --scheduled-timestamp <timestamp> --expiry-timestamp <timestamp> --task-gas-limit <limit> --task-gas-fee <fee> --msgs <messages>",
					Short:     "Create a new scheduled task",
					Long:      "Create a new task to be executed at the specified timestamp",
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"scheduled_timestamp": {
							Name:  "scheduled-timestamp",
							Usage: "Unix timestamp when the task should execute",
						},
						"expiry_timestamp": {
							Name:  "expiry-timestamp",
							Usage: "Unix timestamp after which the task will be considered expired if not executed",
						},
						"task_gas_limit": {
							Name:  "task-gas-limit",
							Usage: "Maximum gas limit for the task execution",
						},
						"task_gas_fee": {
							Name:  "task-gas-fee",
							Usage: "Gas fee for the task execution (format: <amount>dys)",
						},
						"msgs": {
							Name:  "msgs",
							Usage: "JSON-encoded messages to be executed when the task runs",
						},
					},
				},
				{
					RpcMethod: "DeleteTask",
					Use:       "delete-task --task-id <task-id>",
					Short:     "Delete a scheduled task",
					Long:      "Delete a task that you have created",
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"task_id": {
							Name:  "task-id",
							Usage: "The ID of the task to delete",
						},
					},
				},
				{
					RpcMethod: "UpdateParams",
					Use:       "update-params --authority <address> --params <json>",
					Short:     "Update crontask module parameters (governance authority only)",
					Long:      "Update the x/crontask module Params in a single message, bypassing the longer gov proposal flow. Requires the module authority account as signer.",
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"authority": {Name: "authority", Usage: "authority address (defaults to gov module account)"},
						"params":    {Name: "params", Usage: "JSON-encoded Params object"},
					},
				},
			},
		},
	}
}
