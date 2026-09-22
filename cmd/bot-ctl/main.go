package main

import (
	"fmt"
	"os"

	"github.com/bot-ctl/bot-ctl/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
}
