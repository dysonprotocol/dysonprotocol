package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/cosmos/cosmos-sdk/codec/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/gogoproto/proto"
	icahosttypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/types"
)

type AcknowledgementJSON struct {
	Result string `json:"result"`
}

func main() {
	// Read from stdin or argument
	var ackData string
	if len(os.Args) > 1 {
		ackData = os.Args[1]
	} else {
		data, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
			os.Exit(1)
		}
		ackData = string(data)
	}

	// Decode base64 acknowledgement
	ackBytes, err := base64.StdEncoding.DecodeString(ackData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding base64: %v\n", err)
		os.Exit(1)
	}

	// Unmarshal first JSON wrapper
	var ackJSON AcknowledgementJSON
	if err := json.Unmarshal(ackBytes, &ackJSON); err != nil {
		fmt.Fprintf(os.Stderr, "Error unmarshaling JSON: %v\n", err)
		os.Exit(1)
	}

	// Decode base64 result
	resultBytes, err := base64.StdEncoding.DecodeString(ackJSON.Result)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding result base64: %v\n", err)
		os.Exit(1)
	}

	// Unmarshal ICA response
	response := &icahosttypes.MsgModuleQuerySafeResponse{}
	if err := proto.Unmarshal(resultBytes, response); err != nil {
		fmt.Fprintf(os.Stderr, "Error unmarshaling protobuf: %v\n", err)
		os.Exit(1)
	}

	if len(response.Responses) == 0 {
		fmt.Fprintf(os.Stderr, "No responses in MsgModuleQuerySafeResponse\n")
		os.Exit(1)
	}

	// Try to decode as Any
	any := &types.Any{}
	if err := proto.Unmarshal(response.Responses[0], any); err == nil {
		fmt.Printf("Wrapped in Any. TypeUrl: %s\n", any.TypeUrl)
		if any.TypeUrl == "/ibc.applications.interchain_accounts.host.v1.MsgModuleQuerySafeResponse" {
			// Nested MsgModuleQuerySafeResponse
			nested := &icahosttypes.MsgModuleQuerySafeResponse{}
			if err := proto.Unmarshal(any.Value, nested); err != nil {
				fmt.Fprintf(os.Stderr, "Error decoding nested MsgModuleQuerySafeResponse: %v\n", err)
				os.Exit(1)
			}
			if len(nested.Responses) == 0 {
				fmt.Fprintf(os.Stderr, "No responses in nested MsgModuleQuerySafeResponse\n")
				os.Exit(1)
			}
			// Try to decode as QueryAllBalancesResponse
			bankResp := &banktypes.QueryAllBalancesResponse{}
			if err := proto.Unmarshal(nested.Responses[0], bankResp); err == nil {
				jsonBytes, _ := json.MarshalIndent(bankResp, "", "  ")
				fmt.Println(string(jsonBytes))
				return
			} else {
				fmt.Fprintf(os.Stderr, "Error decoding nested response as QueryAllBalancesResponse: %v\n", err)
				os.Exit(1)
			}
		} else {
			// Try to decode the Value as QueryAllBalancesResponse
			bankResp := &banktypes.QueryAllBalancesResponse{}
			if err := proto.Unmarshal(any.Value, bankResp); err == nil {
				jsonBytes, _ := json.MarshalIndent(bankResp, "", "  ")
				fmt.Println(string(jsonBytes))
				return
			} else {
				fmt.Fprintf(os.Stderr, "Error decoding Any.Value as QueryAllBalancesResponse: %v\n", err)
				os.Exit(1)
			}
		}
	} else {
		fmt.Fprintf(os.Stderr, "Not an Any: %v\n", err)
		fmt.Println(base64.StdEncoding.EncodeToString(response.Responses[0]))
		os.Exit(1)
	}
}
