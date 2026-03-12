package cli

import (
	"fmt"
	"log/slog"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/platform-engineering-labs/orbital"
	"github.com/spf13/cobra"
)

var Status = &cobra.Command{
	Use:     "status [package]",
	Short:   "show status of a given package",
	GroupID: "manage",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		if cmd.Flags().Arg(0) == "" {
			return fmt.Errorf("package not specified")
		}

		orb, err := orbital.Dynamic(slog.New(Logger), cfgPath)
		if err != nil {
			return err
		}

		_, packages, err := orb.Status(cmd.Flags().Arg(0))
		if err != nil {
			return err
		}

		if len(packages) == 0 {
			return nil
		}

		tbl := table.New()
		tbl.Headers("Name", "Version", "Summary", "Status")

		for _, pkg := range packages {
			status := ""

			if pkg.Frozen {
				status = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Render("*")
			} else if pkg.Priority == -1 {
				status = lipgloss.NewStyle().Foreground(lipgloss.Color("202")).Render("*")
			}

			tbl.Row(pkg.Name, pkg.Version.String(), pkg.Summary, status)
		}

		fmt.Println(tbl.Render())

		return nil
	},
}
