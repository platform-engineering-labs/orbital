package cli

import (
	"log/slog"

	"github.com/platform-engineering-labs/orbital"
	"github.com/spf13/cobra"
)

var Update = &cobra.Command{
	Use:     "update [package ...]",
	Short:   "update package(s)",
	GroupID: "manage",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		orb, err := orbital.Dynamic(cfgPath, slog.New(Logger))
		if err != nil {
			return err
		}

		return orb.Update(cmd.Flags().Args()...)
	},
}
