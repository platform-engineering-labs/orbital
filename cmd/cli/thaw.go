package cli

import (
	"fmt"
	"log/slog"

	"github.com/platform-engineering-labs/orbital"
	"github.com/spf13/cobra"
)

var Thaw = &cobra.Command{
	Use:     "thaw [package ...]",
	Short:   "thaw package version(s)",
	GroupID: "manage",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		orb, err := orbital.Dynamic(cfgPath, slog.New(Logger))
		if err != nil {
			return err
		}

		if len(cmd.Flags().Args()) == 0 {
			return fmt.Errorf("package(s) argument required")
		}

		return orb.Thaw(cmd.Flags().Args()...)
	},
}
