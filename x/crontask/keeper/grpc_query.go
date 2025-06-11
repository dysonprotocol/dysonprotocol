package keeper

import (
	"context"
	"encoding/binary"
	"fmt"
	"sort"

	"cosmossdk.io/store/prefix"
	crontasktypes "dysonprotocol.com/x/crontask/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// isValidStatus checks if the provided status is a valid task status
func isValidStatus(status string) bool {
	validStatuses := []string{
		crontasktypes.TaskStatus_SCHEDULED,
		crontasktypes.TaskStatus_PENDING,
		crontasktypes.TaskStatus_DONE,
		crontasktypes.TaskStatus_FAILED,
		crontasktypes.TaskStatus_EXPIRED,
	}

	for _, s := range validStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// Ensure the queryServer implements the QueryServer interface
var _ crontasktypes.QueryServer = queryServer{}

// queryServer is a wrapper for Keeper that implements the QueryServer interface
type queryServer struct {
	k Keeper
}

// NewQueryServer creates a new QueryServer instance
func NewQueryServer(k Keeper) crontasktypes.QueryServer {
	return queryServer{k: k}
}

// TaskByID returns a task by its ID
func (q queryServer) TaskByID(ctx context.Context, req *crontasktypes.QueryTaskByIDRequest) (*crontasktypes.QueryTaskByIDResponse, error) {
	task, err := q.k.GetTask(ctx, req.TaskId)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("task with ID %d not found", req.TaskId))
	}
	return &crontasktypes.QueryTaskByIDResponse{Task: &task}, nil
}

// TasksByAddress returns all tasks created by a specific address
func (q queryServer) TasksByAddress(ctx context.Context, req *crontasktypes.QueryTasksByAddressRequest) (*crontasktypes.QueryTasksResponse, error) {
	store := prefix.NewStore(q.k.kvStore(ctx), append(indexAddrPrefix, []byte(req.Creator)...))

	var tasks []*crontasktypes.Task
	pageRes, err := query.Paginate(store, req.Pagination, func(key, _ []byte) error {
		id := binary.BigEndian.Uint64(key[len(key)-8:])
		task, err := q.k.GetTask(ctx, id)
		if err != nil {
			return err
		}
		tasks = append(tasks, &task)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &crontasktypes.QueryTasksResponse{Tasks: tasks, Pagination: pageRes}, nil
}

// TasksByStatusTimestamp returns tasks filtered by status and ordered by timestamp
func (q queryServer) TasksByStatusTimestamp(ctx context.Context, req *crontasktypes.QueryTasksByStatusTimestampRequest) (*crontasktypes.QueryTasksResponse, error) {
	if req.Pagination == nil {
		req.Pagination = &query.PageRequest{}
	}
	// Map ascending flag to PageRequest.Reverse (false=asc, true=desc)
	req.Pagination.Reverse = !req.Ascending

	prefixBz := append(indexStatusTsPrefix, []byte(req.Status)...)
	store := prefix.NewStore(q.k.kvStore(ctx), prefixBz)

	var tasks []*crontasktypes.Task
	pageRes, err := query.Paginate(store, req.Pagination, func(key, _ []byte) error {
		id := binary.BigEndian.Uint64(key[len(key)-8:])
		task, err := q.k.GetTask(ctx, id)
		if err != nil {
			return nil
		}
		tasks = append(tasks, &task)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &crontasktypes.QueryTasksResponse{Tasks: tasks, Pagination: pageRes}, nil
}

// TasksByStatusGasPrice returns tasks filtered by status and ordered by gas price
func (q queryServer) TasksByStatusGasPrice(ctx context.Context, req *crontasktypes.QueryTasksByStatusGasPriceRequest) (*crontasktypes.QueryTasksResponse, error) {
	if req.Pagination == nil {
		req.Pagination = &query.PageRequest{}
	}
	req.Pagination.Reverse = !req.Ascending

	prefixBz := append(indexStatusGasPrefix, []byte(req.Status)...)
	store := prefix.NewStore(q.k.kvStore(ctx), prefixBz)

	var tasks []*crontasktypes.Task
	pageRes, err := query.Paginate(store, req.Pagination, func(key, _ []byte) error {
		id := binary.BigEndian.Uint64(key[len(key)-8:])
		task, err := q.k.GetTask(ctx, id)
		if err != nil {
			return nil
		}
		tasks = append(tasks, &task)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &crontasktypes.QueryTasksResponse{Tasks: tasks, Pagination: pageRes}, nil
}

// TasksAll returns all tasks ordered by task ID (ascending)
func (q queryServer) TasksAll(ctx context.Context, req *crontasktypes.QueryAllTasksRequest) (*crontasktypes.QueryTasksResponse, error) {
	if req.Pagination == nil {
		req.Pagination = &query.PageRequest{Limit: 100}
	}

	if len(req.Pagination.Key) != 0 {
		return nil, status.Error(codes.InvalidArgument, "pagination by key is not supported for this query")
	}

	if req.Pagination.Limit == 0 {
		req.Pagination.Limit = 100
	}

	// Iterate all tasks (primary map is keyed by uint64 taskID)
	iter, err := q.k.Tasks.Iterate(ctx, nil)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	defer iter.Close()

	// Skip offset
	skipped := uint64(0)
	for iter.Valid() && skipped < req.Pagination.Offset {
		iter.Next()
		skipped++
	}

	var tasks []*crontasktypes.Task
	collected := uint64(0)
	for iter.Valid() && collected < req.Pagination.Limit {
		id, err := iter.Key()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		task, err := q.k.GetTask(ctx, id)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		tasks = append(tasks, &task)
		collected++
		iter.Next()
	}

	pageRes := &query.PageResponse{Total: uint64(len(tasks)) + req.Pagination.Offset}
	return &crontasktypes.QueryTasksResponse{Tasks: tasks, Pagination: pageRes}, nil
}

// TasksScheduled returns tasks with SCHEDULED status ordered by timestamp asc (from now)
func (q queryServer) TasksScheduled(ctx context.Context, req *crontasktypes.QueryScheduledTasksRequest) (*crontasktypes.QueryTasksResponse, error) {
	// reuse TasksByStatusTimestamp with status SCHEDULED
	inner := &crontasktypes.QueryTasksByStatusTimestampRequest{
		Status:     crontasktypes.TaskStatus_SCHEDULED,
		Ascending:  req.Ascending,
		Pagination: req.Pagination,
	}
	return q.TasksByStatusTimestamp(ctx, inner)
}

// TasksPending returns tasks with PENDING status ordered by gas price (desc default)
func (q queryServer) TasksPending(ctx context.Context, req *crontasktypes.QueryPendingTasksRequest) (*crontasktypes.QueryTasksResponse, error) {
	inner := &crontasktypes.QueryTasksByStatusGasPriceRequest{
		Status:     crontasktypes.TaskStatus_PENDING,
		Ascending:  req.Ascending,
		Pagination: req.Pagination,
	}
	return q.TasksByStatusGasPrice(ctx, inner)
}

// TasksDone returns tasks with DONE status ordered by execution timestamp (desc default)
func (q queryServer) TasksDone(ctx context.Context, req *crontasktypes.QueryDoneTasksRequest) (*crontasktypes.QueryTasksResponse, error) {
	// For now, iterate over tasks with DONE status via ByStatusTimestamp index
	// then sort in memory by ExecutionTimestamp.

	inner := &crontasktypes.QueryTasksByStatusTimestampRequest{
		Status:     crontasktypes.TaskStatus_DONE,
		Ascending:  req.Ascending,                // uses scheduled timestamp; we'll reorder by exec ts below
		Pagination: &query.PageRequest{Limit: 0}, // we will apply pagination after sorting
	}

	baseRes, err := q.TasksByStatusTimestamp(ctx, inner)
	if err != nil {
		return nil, err
	}

	// sort slice by ExecutionTimestamp depending on ascending flag
	tasks := baseRes.Tasks
	sort.Slice(tasks, func(i, j int) bool {
		if req.Ascending {
			return tasks[i].ExecutionTimestamp < tasks[j].ExecutionTimestamp
		}
		return tasks[i].ExecutionTimestamp > tasks[j].ExecutionTimestamp
	})

	// Apply offset/limit similar to earlier
	var offset uint64
	var limit uint64 = 100
	if req.Pagination != nil {
		offset = req.Pagination.GetOffset()
		if l := req.Pagination.GetLimit(); l != 0 {
			limit = l
		}
	}
	var sliced []*crontasktypes.Task
	if uint64(offset) >= uint64(len(tasks)) {
		sliced = []*crontasktypes.Task{}
	} else {
		end := uint64(offset) + limit
		if end > uint64(len(tasks)) {
			end = uint64(len(tasks))
		}
		sliced = tasks[offset:end]
	}

	pageRes := &query.PageResponse{Total: uint64(len(tasks))}
	return &crontasktypes.QueryTasksResponse{Tasks: sliced, Pagination: pageRes}, nil
}

// Params returns the module parameters
func (q queryServer) Params(ctx context.Context, req *crontasktypes.QueryParamsRequest) (*crontasktypes.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	params, err := q.k.GetParams(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &crontasktypes.QueryParamsResponse{Params: &params}, nil
}

// QueryParams was the old name - keeping it for compatibility but making it call the new method
func (k Keeper) QueryParams(ctx context.Context, req *crontasktypes.QueryParamsRequest) (*crontasktypes.QueryParamsResponse, error) {
	// Create a queryServer and delegate to it
	q := queryServer{k: k}
	return q.Params(ctx, req)
}
