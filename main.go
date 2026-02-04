package main

import (
	"os"

	"github.com/mj1618/swarm-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
