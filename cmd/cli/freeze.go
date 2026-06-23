package cli

import (
	"fmt"
	"log/slog"

	"github.com/platform-engineering-labs/orbital"
	"github.com/spf13/cobra"
)

var Freeze = &cobra.Command{
	Use:     "freeze [package ...]",
	Short:   "freeze package version(s)",
	GroupID: "manage",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		if cmd.Flags().NArg() == 0 {
			return fmt.Errorf("package(s) argument required")
		}

		orb, err := orbital.New(slog.New(Logger), orbital.WithConfig(cfgPath), orbital.WithSudo())
		if err != nil {
			return err
		}

		return orb.Freeze(cmd.Flags().Args()...)
	},
}
