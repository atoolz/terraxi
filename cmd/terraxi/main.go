package main

import (
	"fmt"
	"os"

	"github.com/ahlert/terraxi/cmd/terraxi/commands"
)

var version = "dev"

func main() {
	root := commands.NewRootCmd(version)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
