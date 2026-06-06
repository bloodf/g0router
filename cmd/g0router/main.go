package main

import (
	"fmt"
	"os"

	"github.com/bloodf/g0router/internal/cli"
)

var (
	version   = "0.1.0-dev"
	buildDate = ""
)

func main() {
	if err := cli.NewRootCommandWithBuildDate(version, buildDate).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
