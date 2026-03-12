package cli

import (
	"fmt"
	"log/slog"

	"github.com/platform-engineering-labs/orbital"
	"github.com/spf13/cobra"
)

var Remove = &cobra.Command{
	Use:     "remove [package ...]",
	Short:   "remove packages",
	GroupID: "manage",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		if cmd.Flags().NArg() == 0 {
			return fmt.Errorf("must specify at least one package to remove")
		}

		orb, err := orbital.Dynamic(slog.New(Logger), cfgPath)
		if err != nil {
			return err
		}

		return orb.Remove(cmd.Flags().Args()...)
	},
}
