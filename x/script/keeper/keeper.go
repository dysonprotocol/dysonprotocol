package keeper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	"cosmossdk.io/core/header"
	"cosmossdk.io/core/store"
	cosmossdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"dysonprotocol.com/dysvm"
	"dysonprotocol.com/x/script"
	scriptErrors "dysonprotocol.com/x/script/errors"
	scripttypes "dysonprotocol.com/x/script/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	"github.com/gorilla/mux"
	"github.com/gorilla/rpc"
	rpcjson "github.com/gorilla/rpc/json"
)

var (
	ScriptMapPrefix = collections.NewPrefix(0)
	ParamsKey       = collections.NewPrefix(1)
)

// Ensure the keeper types implement required interfaces
var _ scripttypes.BranchKeeper = (*BranchService)(nil)

// BranchService implements the atomic execution functionality
type BranchService struct {
	sdkCtx sdk.Context
}

var rpcRe = regexp.MustCompile(`^(\/.+\.Query)([^/]+)Request$`)

// ExecuteWithGasLimit runs a function with a specific gas limit and returns gas used and any error
func (bs *BranchService) ExecuteWithGasLimit(ctx context.Context, gasLimit uint64, fn func(ctx context.Context) error) (uint64, error) {
	// Create a cached context with a gas meter with the specified limit
	sdkCtx := bs.sdkCtx.WithGasMeter(storetypes.NewGasMeter(gasLimit))

	// Create a cache-wrapped context that creates an isolated context for the execution
	cacheCtx, write := sdkCtx.CacheContext()

	// Convert the SDK context to a generic context
	fnCtx := sdk.WrapSDKContext(cacheCtx)

	// Track the gas consumed before execution
	gasConsumedBefore := sdkCtx.GasMeter().GasConsumed()

	// Execute the function
	err := fn(fnCtx)

	// Calculate the amount of gas consumed during execution
	gasUsed := cacheCtx.GasMeter().GasConsumed() - gasConsumedBefore

	// If execution was successful, write state changes back to the parent context
	if err == nil {
		write()
	}

	return gasUsed, err
}

type Keeper struct {
	// Core services
	App            *baseapp.BaseApp
	addressCodec   address.Codec
	validatorCodec address.Codec
	cdc            codec.Codec
	Schema         collections.Schema

	// Store service
	KVStoreService store.KVStoreService

	// Collections
	ScriptMap collections.Map[string, scripttypes.Script]
	params    collections.Item[scripttypes.Params]

	currentDepth int

	// Authority for governance operations
	authority string

	// Optional nameservice keeper for resolving names to addresses
	NameserviceKeeper scripttypes.NameserviceKeeper

	// Account keeper for accessing account information
	AccountKeeper scripttypes.AccountKeeper

	// Authz keeper for managing authorizations
	AuthzKeeper authzkeeper.Keeper

	// Service interfaces
	MsgRouterService   *baseapp.MsgServiceRouter
	QueryRouterService *baseapp.GRPCQueryRouter
}

// MsgRequest defines a request to dispatch a message
type MsgRequest struct {
	JsonMsg  string `json:"json_msg"`
	GasLimit uint64 `json:"gas_limit"`
}

// QueryRequest defines a request to execute a query
type QueryRequest struct {
	JsonQuery   string `protobuf:"bytes,2,opt,name=Jsonquery,proto3" json:"json_query,omitempty"`
	QueryHeight int64  `protobuf:"bytes,3,opt,name=QueryHeight,proto3" json:"query_height,omitempty"`
}

// NewKeeper creates a new script keeper.
func NewKeeper(
	app *baseapp.BaseApp,
	kvStoreService store.KVStoreService,
	cdc codec.Codec,
	accKeeper scripttypes.AccountKeeper,
	nameserviceKeeper scripttypes.NameserviceKeeper,
	authzKeeper authzkeeper.Keeper,
	addressCodec address.Codec,
	validatorCodec address.Codec,
	msgServiceRouter *baseapp.MsgServiceRouter,
	queryServiceRouter *baseapp.GRPCQueryRouter,
	authority string,
) Keeper {
	sb := collections.NewSchemaBuilder(kvStoreService)

	k := Keeper{
		App:                app,
		cdc:                cdc,
		addressCodec:       addressCodec,
		validatorCodec:     validatorCodec,
		KVStoreService:     kvStoreService,
		NameserviceKeeper:  nameserviceKeeper,
		AccountKeeper:      accKeeper,
		AuthzKeeper:        authzKeeper,
		ScriptMap:          collections.NewMap(sb, ScriptMapPrefix, "script_map", collections.StringKey, codec.CollValue[scripttypes.Script](cdc)),
		params:             collections.NewItem(sb, ParamsKey, "params", codec.CollValue[scripttypes.Params](cdc)),
		MsgRouterService:   msgServiceRouter,
		QueryRouterService: queryServiceRouter,
		authority:          authority,
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema
	return k
}

type ExecScriptContext struct {
	Msg                    *scripttypes.MsgExec
	Script                 *scripttypes.Script
	AttachedMessageResults []sdk.Msg
}

type ExecScriptResponse struct {
	Result string
}

func (k Keeper) execScript(ctx sdk.Context, scriptCtx *ExecScriptContext) (*ExecScriptResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	k.currentDepth += 1

	fmt.Printf("currentDepth: %v\n", k.currentDepth)

	executorAddr, err := k.addressCodec.StringToBytes(scriptCtx.Msg.ExecutorAddress)
	if err != nil {
		return nil, cosmossdkerrors.Wrapf(err, "error getting executor address")
	}

	attachedMsgs, err := script.GetMsgExecMessages(scriptCtx.Msg)

	if err != nil {
		return nil, cosmossdkerrors.Wrapf(err, "error getting attached messages")
	}

	if len(scriptCtx.AttachedMessageResults) > 0 {
		// If the pre-populated AttachedMessageResults are provided, use them
		if len(scriptCtx.AttachedMessageResults) != len(attachedMsgs) {
			return nil, cosmossdkerrors.Wrapf(scriptErrors.ErrInvalid, "pre-populated AttachedMessageResults length (%d) does not match attachedMsgs length (%d)", len(scriptCtx.AttachedMessageResults), len(attachedMsgs))
		}
		k.Logger(sdkCtx).Info("Skipping message dispatch, using pre-populated AttachedMessageResults")
	} else {
		// Dispatch messages attached to the script
		results := make([]sdk.Msg, len(attachedMsgs))
		for i, attachedMsg := range attachedMsgs {
			r, err := k.DispatchMessage(sdkCtx, executorAddr, attachedMsg)
			if err != nil {
				return nil, cosmossdkerrors.Wrapf(err, "error dispatching attached message index [%d]", i)
			}
			results[i] = r
		}
		scriptCtx.AttachedMessageResults = results
	}

	fmt.Println("Starting RPC server")
	port, srv, err := k.NewRPCServer(ctx, scriptCtx.Script.Address, k.App)

	if err != nil {
		return nil, err
	}

	fmt.Println("Started RPC server on port", port)

	now := time.Now()
	defer func() {
		fmt.Println(fmt.Sprintf("Elapsed time %s", time.Since(now)))
		k.currentDepth -= 1

		if err := srv.Shutdown(context.Background()); err != nil {
			fmt.Printf("shutdown error")
			panic(err) // failure/timeout shutting down the server gracefully
		}
		fmt.Println("server stopped")
	}()

	msgJSON, err := k.cdc.MarshalInterfaceJSON(scriptCtx.Msg)
	if err != nil {
		return nil, err
	}
	scriptJSON, err := k.cdc.MarshalInterfaceJSON(scriptCtx.Script)
	if err != nil {
		return nil, cosmossdkerrors.Wrap(err, "failed to marshal script")
	}

	attachedMsgResultsJSON := "["
	for i, m := range scriptCtx.AttachedMessageResults {
		msgJSON, err := k.cdc.MarshalInterfaceJSON(m)
		if err != nil {
			return nil, err
		}
		attachedMsgResultsJSON += string(msgJSON)
		if i < len(scriptCtx.AttachedMessageResults)-1 {
			attachedMsgResultsJSON += ","
		}
	}
	attachedMsgResultsJSON += "]"

	headerInfo := header.Info{
		Height:  sdkCtx.BlockHeight(),
		Time:    sdkCtx.BlockTime(),
		ChainID: sdkCtx.ChainID(),
		AppHash: sdkCtx.BlockHeader().AppHash,
		Hash:    sdkCtx.BlockHeader().LastBlockId.Hash,
	}

	headerInfoJSON, err := json.Marshal(headerInfo)
	if err != nil {
		return nil, err
	}

	out, runErr := dysvm.Exec(string(msgJSON),
		string(scriptJSON),
		attachedMsgResultsJSON,
		string(headerInfoJSON),
		port)

	// Consume gas for script execution
	sdkCtx.GasMeter().ConsumeGas(1, "execScript")

	temp := strings.Split(string(out), "\n")
	response := string(temp[len(temp)-1])
	if response == "exit status 1" {
		response = string(temp[len(temp)-2])
	}

	fmt.Printf("Output: %s\n", out)
	fmt.Printf("Response: %s\n", response)
	fmt.Printf("Error: %v\n", runErr)

	if runErr != nil {
		// if the response is json, we can wrap the error in a json error
		if strings.HasPrefix(response, "{") {
			return nil, cosmossdkerrors.Wrap(scriptErrors.ErrScriptExecution, response)
		}
		return nil, runErr
	}

	return &ExecScriptResponse{Result: response}, nil
}

func ConvertRPCPath(in string) string {
	if m := rpcRe.FindStringSubmatch(in); m != nil {
		return m[1] + "/" + m[2]
	}
	return in
}

func GetResponseTypeURL(reqTypeURL string) string {
	// replace Request with Response
	return strings.Replace(reqTypeURL, "Request", "Response", 1)

}

func (k Keeper) HandleJSONAnyQuery(ctx context.Context, req *QueryRequest) (string, error) {
	// Parse the type URL from the JSON
	var anyMsg map[string]interface{}
	err := json.Unmarshal([]byte(req.JsonQuery), &anyMsg)
	if err != nil {
		return "", cosmossdkerrors.Wrapf(err, "failed to parse JSON")
	}

	typeURL, ok := anyMsg["@type"].(string)
	if !ok {
		return "", fmt.Errorf("JSON doesn't contain @type field")
	}

	reqMsg, err := k.cdc.InterfaceRegistry().Resolve(typeURL)
	if err != nil {
		return "", cosmossdkerrors.Wrapf(err, "failed to resolve request type")
	}
	fmt.Println("reqMsg", fmt.Sprintf("%T", reqMsg))

	respMsg, err := k.cdc.InterfaceRegistry().Resolve(GetResponseTypeURL(typeURL))
	if err != nil {
		return "", cosmossdkerrors.Wrapf(err, "failed to resolve response type")
	}
	fmt.Println("respMsg", fmt.Sprintf("%T", respMsg))

	// First try to unmarshal into a specific interface
	var msg sdk.Msg
	err = k.cdc.UnmarshalInterfaceJSON([]byte(req.JsonQuery), &msg)
	if err != nil {
		// If that fails, try a generic approach using the registry
		return "", cosmossdkerrors.Wrapf(err, "failed to unmarshal request")
	}

	// Convert to binary protobuf
	binaryData, err := k.cdc.Marshal(msg)
	if err != nil {
		return "", cosmossdkerrors.Wrapf(err, "failed to marshal to protobuf")
	}

	// Unwrap the SDK context
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Handle historical queries if QueryHeight is specified
	if req.QueryHeight != 0 {

		if req.QueryHeight < 0 {
			return "", cosmossdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "query height must be greater than 0")
		}

		// Get module parameters to determine the maximum allowed historical query height
		params := k.GetParams(ctx)
		maxRelativeHistoricalBlocks := params.MaxRelativeHistoricalBlocks

		relativeHeight := sdkCtx.BlockHeight() - req.QueryHeight
		if relativeHeight > maxRelativeHistoricalBlocks {
			return "", cosmossdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "Max relative historical query height is %d blocks in the past", maxRelativeHistoricalBlocks)
		}

		sdkCtx, err = k.App.CreateQueryContextWithCheckHeader(req.QueryHeight, false, false)
		if err != nil {
			return "", err
		}
	}

	path := ConvertRPCPath(typeURL)

	handler := k.QueryRouterService.Route(path)
	if handler == nil {
		return "", fmt.Errorf("no handler found for query message: %s", typeURL)
	}

	abciReqQuery := abci.RequestQuery{
		Data: binaryData,
	}

	// For historical queries, use the historical context directly
	// For current queries, use a cached context to avoid state changes
	var queryCtx sdk.Context
	if req.QueryHeight != 0 {
		// Historical query - use the historical context directly
		queryCtx = sdkCtx
	} else {
		// Current query - use cached context to avoid state changes
		queryCtx, _ = sdkCtx.CacheContext()
	}

	respQuery, err := handler(queryCtx, &abciReqQuery)
	if err != nil {
		return "", cosmossdkerrors.Wrapf(err, "failed to execute query; message %v", req.JsonQuery)
	}

	// unmarshal the respQuery.Value into the respMsg
	err = k.cdc.Unmarshal(respQuery.Value, respMsg)
	if err != nil {
		return "", cosmossdkerrors.Wrapf(err, "failed to unmarshal response")
	}

	// marshal the respMsg into a json string
	respJSON, err := k.cdc.MarshalInterfaceJSON(respMsg)
	if err != nil {
		return "", cosmossdkerrors.Wrapf(err, "failed to marshal response")
	}

	// The cached context is automatically discarded as we don't call write()
	return string(respJSON), nil
}

func (k Keeper) HandleJSONAnyMsg(ctx context.Context, scriptAddress sdk.AccAddress, req *MsgRequest) (respJSONStr string, gasused uint64, err error) {
	var msg sdk.Msg
	err = k.cdc.UnmarshalInterfaceJSON([]byte(req.JsonMsg), &msg)
	if err != nil {
		err = cosmossdkerrors.Wrapf(err, "failed to UnmarshalInterfaceJSON request: %s", req.JsonMsg)
		return
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// IMPORTANT: Check for nil or zero gas limit
	gasLimit := req.GasLimit
	if gasLimit == 0 {
		gasLimit = sdkCtx.GasMeter().Limit() - sdkCtx.GasMeter().GasConsumed()
	}

	// Create a gas meter safely
	gasMeter := storetypes.NewGasMeter(gasLimit)

	// Create a cached context with the gas meter
	cacheCtx := sdkCtx.WithGasMeter(gasMeter)
	cacheCtx, write := cacheCtx.CacheContext()

	respJSONStr, err = k.dispatchJSONMsg(cacheCtx, scriptAddress, req.JsonMsg)

	// Get gas consumed from the new meter
	gasused = cacheCtx.GasMeter().GasConsumed()

	// Only write if successful
	if err == nil {
		write()
	}

	return respJSONStr, gasused, err
}

// DispatchMessage dispatches a message for execution and returns the result
func (k Keeper) DispatchMessage(sdkCtx sdk.Context, executor sdk.AccAddress, msg sdk.Msg) (sdk.Msg, error) {
	err := validateMsg(msg)
	if err != nil {
		return nil, err
	}

	// Use the MsgServiceRouter to route and handle the message
	handler := k.MsgRouterService.Handler(msg)
	if handler == nil {
		return nil, fmt.Errorf("no message handler found for %s", sdk.MsgTypeURL(msg))
	}

	// Get the response and convert back to sdk.Msg
	resp, err := handler(sdkCtx, msg)
	fmt.Println("DispatchMessage handler", msg, resp)
	if err != nil {
		return nil, cosmossdkerrors.Wrapf(err, "failed to dispatch message")
	}

	// emit all events in the response
	for _, event := range resp.Events {
		sdkCtx.EventManager().EmitEvent(sdk.Event{
			Type:       event.Type,
			Attributes: event.Attributes,
		})
	}

	// Get signers using the codec
	signers, _, err := k.cdc.GetMsgV1Signers(msg)
	if err != nil {
		return nil, err
	}

	// Verify signer count
	if len(signers) != 1 {
		return nil, cosmossdkerrors.Wrap(scriptErrors.ErrUnauthorized, "incorrect number of signers")
	}

	// Verify executor
	if !bytes.Equal(signers[0], executor) {
		return nil, cosmossdkerrors.Wrap(scriptErrors.ErrUnauthorized, "the first signer must be the message creator")
	}

	// Extract message from response
	var respMsg sdk.Msg
	if resp != nil && len(resp.MsgResponses) > 0 {
		err = k.cdc.UnpackAny(resp.MsgResponses[0], &respMsg)
		if err != nil {
			return nil, cosmossdkerrors.Wrapf(err, "failed to unpack response message")
		}
	}

	return respMsg, nil
}

func validateMsg(msg sdk.Msg) error {
	m, ok := msg.(sdk.HasValidateBasic)
	if !ok {
		return nil
	}

	if err := m.ValidateBasic(); err != nil {
		return err
	}

	return nil
}

type RpcService struct {
	k             *Keeper
	ctx           context.Context
	ScriptAddress sdk.AccAddress
	App           *baseapp.BaseApp
}

func (k Keeper) NewRPCServer(ctx context.Context, address string, app *baseapp.BaseApp) (string, *http.Server, error) {
	s := rpc.NewServer()
	s.RegisterCodec(rpcjson.NewCodec(), "application/json")
	s.RegisterCodec(rpcjson.NewCodec(), "application/json;charset=UTF-8")
	rpcservice := new(RpcService)
	rpcservice.k = &k
	rpcservice.ctx = ctx
	rpcservice.App = app
	addr, err := k.addressCodec.StringToBytes(address)

	if err != nil {
		return "", nil, err
	}
	rpcservice.ScriptAddress = addr

	s.RegisterService(rpcservice, "")
	r := mux.NewRouter()
	r.Handle("/rpc", s)
	srv := &http.Server{Handler: r}

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}

	//fmt.Println("Using port:", listener.Addr().(*net.TCPAddr).Port)

	go func() {
		//fmt.Println("start ListenAndServe")
		srv.Serve(listener)
		//fmt.Println("end ListenAndServe")
	}()
	//fmt.Println("running dysvm")
	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	return port, srv, nil
}

func (k Keeper) RunWeb(ctx context.Context, address string, httpreq string) (string, error) {
	now := time.Now()
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	cacheCtx, _ := sdkCtx.CacheContext()

	// Resolve the address parameter using the nameservice keeper
	resolvedAddress, err := k.NameserviceKeeper.ResolveNameOrAddress(cacheCtx, address)
	if err != nil {
		return "", cosmossdkerrors.Wrapf(err, "failed to resolve address or name: %s", address)
	}

	script, err := k.ScriptMap.Get(cacheCtx, resolvedAddress)
	if cosmossdkerrors.IsOf(err, collections.ErrNotFound) {
		return "", status.Errorf(codes.NotFound, "script with address %s doesn't exist", resolvedAddress)
	}
	if err != nil {
		return "", cosmossdkerrors.Wrap(err, "failed to get script")
	}

	scriptJSON, err := k.cdc.MarshalInterfaceJSON(&script)
	if err != nil {
		return "", err
	}
	headerInfo := header.Info{
		Height:  cacheCtx.BlockHeight(),
		Time:    cacheCtx.BlockTime(),
		ChainID: cacheCtx.ChainID(),
		AppHash: cacheCtx.BlockHeader().AppHash,
		Hash:    cacheCtx.BlockHeader().LastBlockId.Hash,
	}

	headerInfoJSON, err := json.Marshal(headerInfo)
	if err != nil {
		return "", err
	}

	fmt.Println("Starting RPC server")
	port, srv, err := k.NewRPCServer(cacheCtx, script.Address, k.App)

	if err != nil {
		return "", err
	}
	defer func() {
		fmt.Println(fmt.Sprintf("Elapsed time %s", time.Since(now)))
		k.currentDepth -= 1

		if err := srv.Shutdown(context.Background()); err != nil {
			fmt.Printf("shutdown error")
			panic(err) // failure/timeout shutting down the server gracefully
		}
		fmt.Println("server stopped")
	}()

	fmt.Println("Started RPC server on port", port)

	out, err := dysvm.Wsgi(port, string(scriptJSON), string(headerInfoJSON), httpreq)

	if err != nil {
		return "", cosmossdkerrors.Wrapf(err, "error running script: %s", string(out))
	}

	return out, nil
}

// GetCodec returns the codec used by the keeper
// This is useful for testing to access the interface registry
func (k Keeper) GetCodec() codec.Codec {
	return k.cdc
}

func (k Keeper) dispatchJSONMsg(ctx sdk.Context, scriptAddress sdk.AccAddress, jsonMsg string) (string, error) {
	var msg sdk.Msg
	err := k.cdc.UnmarshalInterfaceJSON([]byte(jsonMsg), &msg)
	if err != nil {
		return "", cosmossdkerrors.Wrapf(err, "failed to UnmarshalInterfaceJSON message")
	}

	resp, err := k.DispatchMessage(ctx, scriptAddress, msg)
	if err != nil {
		return "", cosmossdkerrors.Wrapf(err, "failed to DispatchMessage")
	}

	bz, err := k.cdc.MarshalInterfaceJSON(resp)
	if err != nil {
		return "", cosmossdkerrors.Wrapf(err, "failed to MarshalInterfaceJSON response")
	}

	return string(bz), nil
}

// GetParams returns the current module parameters
func (k Keeper) GetParams(ctx context.Context) (params scripttypes.Params) {
	params, err := k.params.Get(ctx)
	if err != nil {
		// If params don't exist, return defaults
		return scripttypes.DefaultParams()
	}
	return params
}

// SetParams sets the module parameters
func (k Keeper) SetParams(ctx context.Context, params scripttypes.Params) error {
	if err := params.Validate(); err != nil {
		return err
	}
	return k.params.Set(ctx, params)
}

// Logger returns a module-specific logger
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/script")
}
