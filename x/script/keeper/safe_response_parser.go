package keeper

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// convertQueryPathToType converts a query path to its corresponding type URL with the given suffix
// e.g., "/cosmos.bank.v1beta1.Query/AllBalances" + "Response" -> "/cosmos.bank.v1beta1.QueryAllBalancesResponse"
// e.g., "/cosmos.bank.v1beta1.Query/AllBalances" + "Request" -> "/cosmos.bank.v1beta1.QueryAllBalancesRequest"
func convertQueryPathToType(queryPath, suffix string) string {
	// Split by "/" to separate the service and method
	parts := strings.Split(queryPath, "/")
	if len(parts) != 3 {
		return "" // Invalid format
	}

	// parts[0] is empty, parts[1] is service, parts[2] is method
	service := parts[1]
	method := parts[2]

	// Remove "Query" from service and add it back with method + suffix
	if strings.HasSuffix(service, ".Query") {
		baseService := strings.TrimSuffix(service, ".Query")
		return "/" + baseService + ".Query" + method + suffix
	}

	return "" // Not a query service
}

// convertQueryPathToResponseType converts a query path to its corresponding response type URL
func convertQueryPathToResponseType(queryPath string) string {
	return convertQueryPathToType(queryPath, "Response")
}

// convertQueryPathToRequestType converts a query path to its corresponding request type URL
func convertQueryPathToRequestType(queryPath string) string {
	return convertQueryPathToType(queryPath, "Request")
}

// decodeMessage decodes bytes using the given type URL and returns a JSON map
func (k *Keeper) decodeMessage(msgBytes []byte, typeURL string) (map[string]interface{}, error) {
	// Create an empty message of the given type
	msg, err := sdk.GetMsgFromTypeURL(k.cdc, typeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get message type for %s: %w", typeURL, err)
	}

	// Unmarshal the bytes into the message
	if err := k.cdc.Unmarshal(msgBytes, msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Convert to JSON and then to map
	jsonBytes, err := k.cdc.MarshalInterfaceJSON(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal to JSON: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	return result, nil
}

// Constants for message types
const (
	MsgModuleQuerySafeType         = "/ibc.applications.interchain_accounts.host.v1.MsgModuleQuerySafe"
	MsgModuleQuerySafeResponseType = "/ibc.applications.interchain_accounts.host.v1.MsgModuleQuerySafeResponse"
)

// checkMessageType safely checks if a message map has the specified @type
func checkMessageType(msgMap map[string]interface{}, expectedType string) bool {
	msgType, ok := msgMap["@type"].(string)
	return ok && msgType == expectedType
}

// copySlice creates a copy of a slice to avoid modifying the original
func copyInterfaceSlice(src []interface{}) []interface{} {
	result := make([]interface{}, len(src))
	copy(result, src)
	return result
}

// copyMapSlice creates a copy of a map slice to avoid modifying the original
func copyMapSlice(src []map[string]interface{}) []map[string]interface{} {
	result := make([]map[string]interface{}, len(src))
	copy(result, src)
	return result
}

// ProcessQueryResponses enriches ack messages by decoding query responses.
// It matches MsgModuleQuerySafeResponse entries with their corresponding MsgModuleQuerySafe requests
// and decodes the base64 responses using the appropriate response types.
func (k *Keeper) ProcessQueryResponses(ackMsgsJson []map[string]interface{}, packetMessages []interface{}) []map[string]interface{} {
	result := copyMapSlice(ackMsgsJson)

	// Process each acknowledgement message
	for ackIdx, ackMsg := range result {
		// Check if this is a MsgModuleQuerySafeResponse
		if !checkMessageType(ackMsg, MsgModuleQuerySafeResponseType) {
			continue
		}

		// Get the corresponding packet message at the same index
		if ackIdx >= len(packetMessages) {
			continue
		}

		packetMsg, ok := packetMessages[ackIdx].(map[string]interface{})
		if !ok || !checkMessageType(packetMsg, MsgModuleQuerySafeType) {
			continue
		}

		// Extract requests and responses
		requests, reqOk := packetMsg["requests"].([]interface{})
		responses, respOk := ackMsg["responses"].([]interface{})
		if !reqOk || !respOk {
			continue
		}

		// Process each request-response pair
		decodedResponses := make([]interface{}, len(responses))
		for i, response := range responses {
			decodedResponses[i] = k.processQueryResponse(response, i, requests)
		}

		// Replace the responses array with decoded responses
		result[ackIdx]["responses"] = decodedResponses
	}

	return result
}

// ProcessPacketMessages enriches packet messages by decoding query request data.
// It finds MsgModuleQuerySafe messages and decodes the base64 request data using the appropriate request types.
func (k *Keeper) ProcessPacketMessages(packetMessages []interface{}) []interface{} {
	result := copyInterfaceSlice(packetMessages)

	// Process each packet message
	for msgIdx, msg := range result {
		msgMap, ok := msg.(map[string]interface{})
		if !ok || !checkMessageType(msgMap, MsgModuleQuerySafeType) {
			continue
		}

		// Extract requests from message
		requests, ok := msgMap["requests"].([]interface{})
		if !ok {
			continue
		}

		// Process each request
		decodedRequests := make([]interface{}, len(requests))
		for i, request := range requests {
			decodedRequests[i] = k.processQueryRequest(request)
		}

		// Replace the requests array with decoded requests
		msgMap["requests"] = decodedRequests
		result[msgIdx] = msgMap
	}

	return result
}

// decodeBase64Data safely decodes base64 string to bytes
func decodeBase64Data(dataStr string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(dataStr)
}

// processQueryRequest handles the decoding of a single query request
func (k *Keeper) processQueryRequest(request interface{}) interface{} {
	requestMap, ok := request.(map[string]interface{})
	if !ok {
		return request // Keep original if not a map
	}

	// Get the query path and data
	queryPath, ok := requestMap["path"].(string)
	if !ok {
		return request // Keep original if no path
	}

	dataStr, ok := requestMap["data"].(string)
	if !ok {
		return request // Keep original if no data
	}

	// Decode the base64 request data
	requestBytes, err := decodeBase64Data(dataStr)
	if err != nil {
		return request // Keep original if decode fails
	}

	// Convert query path to request type URL
	requestTypeURL := convertQueryPathToRequestType(queryPath)
	if requestTypeURL == "" {
		return request // Keep original if conversion fails
	}

	// Decode the request
	if decodedRequest, err := k.decodeMessage(requestBytes, requestTypeURL); err != nil {
		return request // Keep original if decode fails
	} else {
		// Create a new request map with decoded data
		enrichedRequest := make(map[string]interface{})
		for k, v := range requestMap {
			enrichedRequest[k] = v
		}
		enrichedRequest["decoded_data"] = decodedRequest
		return enrichedRequest
	}
}

// processQueryResponse handles the decoding of a single query response
func (k *Keeper) processQueryResponse(response interface{}, requestIndex int, requests []interface{}) interface{} {
	responseStr, ok := response.(string)
	if !ok {
		return response // Keep original if not a string
	}

	// Decode the base64 response
	responseBytes, err := decodeBase64Data(responseStr)
	if err != nil {
		return response // Keep original if decode fails
	}

	// Get the corresponding request to determine response type
	if requestIndex >= len(requests) {
		return response // Keep original if no matching request
	}

	request, ok := requests[requestIndex].(map[string]interface{})
	if !ok {
		return response // Keep original if request is malformed
	}

	queryPath, ok := request["path"].(string)
	if !ok {
		return response // Keep original if no path
	}

	// Convert query path to response type URL
	responseTypeURL := convertQueryPathToResponseType(queryPath)
	if responseTypeURL == "" {
		return response // Keep original if conversion fails
	}

	// Decode the response
	if decodedResponse, err := k.decodeMessage(responseBytes, responseTypeURL); err != nil {
		return response // Keep original if decode fails
	} else {
		return decodedResponse // Use decoded response
	}
}
