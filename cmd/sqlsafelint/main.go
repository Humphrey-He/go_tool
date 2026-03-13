package main

import (
	"context"
	"os"

	"go_tool/internal/cli"
)

func main() {
	if err := cli.Run(context.Background(), os.Args[1:]); err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
