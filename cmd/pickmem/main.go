package main

import (
	"fmt"
	"os"

	"github.com/qwgao/pickmem/internal/cli"
)

func main() {
	if err := cli.NewRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "pickmem:", err)
		os.Exit(1)
	}
}
