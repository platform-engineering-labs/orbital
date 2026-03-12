package cli

import (
	"fmt"
	"log/slog"

	"github.com/platform-engineering-labs/orbital"
	"github.com/spf13/cobra"
)

func init() {
	Install.Flags().Bool("refresh", false, "refresh metadata")
}

var Install = &cobra.Command{
	Use:     "install [package ...]",
	Short:   "install packages",
	GroupID: "manage",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")
		refresh, _ := cmd.Flags().GetBool("refresh")

		if cmd.Flags().NArg() == 0 {
			return fmt.Errorf("package argument required")
		}

		orb, err := orbital.Dynamic(slog.New(Logger), cfgPath)
		if err != nil {
			return err
		}

		if refresh {
			err := orb.Refresh()
			if err != nil {
				return err
			}
		}

		return orb.Install(cmd.Flags().Args()...)
	},
}
