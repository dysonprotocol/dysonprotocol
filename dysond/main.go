package main

import (
	"fmt"
	"os"

	"dysonprotocol.com"
	cmd "dysonprotocol.com/dysond/cmd"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func main() {
	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount("dys", "dyspub")                     // account addresses
	cfg.SetBech32PrefixForValidator("dysvaloper", "dysvaloperpub")     // validator operator addresses
	cfg.SetBech32PrefixForConsensusNode("dysvalcons", "dysvalconspub") // consensus addresses

	rootCmd := cmd.NewRootCmd()
	// TODO: set the default node env prefix
	if err := svrcmd.Execute(rootCmd, "DYSON", dysonprotocol.DefaultNodeHome); err != nil {
		fmt.Fprintln(rootCmd.OutOrStderr(), err)
		os.Exit(1)
	}
}
