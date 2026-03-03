package main

import (
	"fmt"
	"os"

	"github.com/braidsdev/braids/cmd/braids/cli"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	if err := cli.Execute(version, commit); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
