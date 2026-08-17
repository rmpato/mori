// Command mori is a quiet place to remember your days.
package main

import (
	"context"
	"os"

	"github.com/rmpato/mori/internal/cli"
)

func main() {
	if err := cli.Execute(context.Background()); err != nil {
		os.Exit(1)
	}
}
