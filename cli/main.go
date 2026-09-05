package main

import (
	"fmt"
	"os"

	"github.com/rosa-gus/coleoptera_identifier/cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
