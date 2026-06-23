package cli

import (
	"log/slog"

	"github.com/platform-engineering-labs/orbital"
	"github.com/spf13/cobra"
)

var Refresh = &cobra.Command{
	Use:     "refresh",
	Short:   "refresh repository metadata",
	GroupID: "manage",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		orb, err := orbital.New(slog.New(Logger), orbital.WithConfig(cfgPath), orbital.WithSudo())
		if err != nil {
			return err
		}

		return orb.Refresh()
	},
}
