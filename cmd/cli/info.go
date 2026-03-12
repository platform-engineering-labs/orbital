package cli

import (
	"fmt"
	"log/slog"

	"charm.land/lipgloss/v2/table"
	"github.com/platform-engineering-labs/orbital"
	"github.com/spf13/cobra"
)

var Info = &cobra.Command{
	Use:     "info [package]",
	Short:   "show installed package metadata",
	GroupID: "manage",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		if cmd.Flags().Arg(0) == "" {
			return fmt.Errorf("package argument required")
		}

		orb, err := orbital.Dynamic(cfgPath, slog.New(Logger))
		if err != nil {
			return err
		}

		manifest, err := orb.Info(cmd.Flags().Arg(0))
		if err != nil {
			return err
		}

		tbl := table.New()
		tbl.Headers("Package")

		tbl.Row("Name", manifest.Name)
		tbl.Row("Version", manifest.Version.String())
		tbl.Row("Publisher", manifest.Publisher)
		tbl.Row("Platform", manifest.Platform().String())
		tbl.Row("Summary", manifest.Summary)
		tbl.Row("Description", manifest.Description)

		fmt.Println(tbl.Render())

		tbl = table.New()
		tbl.Headers("Metadata")

		for ns, meta := range manifest.Metadata {
			for k, v := range meta {
				tbl.Row(fmt.Sprintf("%s:%s", ns, k), v)
			}
		}

		fmt.Println(tbl.Render())

		return nil
	},
}
