package main

import (
	"context"
	"os"

	"charm.land/fang/v2"
	"github.com/platform-engineering-labs/orbital/cmd/cli"
)

func main() {
	if err := fang.Execute(
		context.Background(),
		cli.Root,
		fang.WithoutVersion(),
	); err != nil {
		os.Exit(1)
	}
}
