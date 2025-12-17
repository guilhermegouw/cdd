// CDD is an AI-powered coding assistant CLI.
package main

import (
	"fmt"
	"os"

	"github.com/guilhermegouw/cdd/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
