package main

import (
	"context"
	"os"

	"charm.land/fang/v2"
	"github.com/platform-engineering-labs/orbital/cmd/cli"
	"github.com/platform-engineering-labs/pelx/theme"
)

func main() {
	if err := fang.Execute(
		context.Background(),
		cli.Root,
		fang.WithoutVersion(),
		fang.WithColorSchemeFunc(theme.FangTheme),
	); err != nil {
		os.Exit(1)
	}
}
