package cli

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/platform-engineering-labs/orbital"
	"github.com/spf13/cobra"
)

var Contents = &cobra.Command{
	Use:     "contents [package]",
	Short:   "show installed package contents",
	GroupID: "manage",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		if cmd.Flags().Arg(0) == "" {
			return fmt.Errorf("package argument required")
		}

		orb, err := orbital.Dynamic(slog.New(Logger), cfgPath)
		if err != nil {
			return err
		}

		contents, err := orb.Contents(cmd.Flags().Arg(0))
		if err != nil {
			return err
		}

		for _, f := range contents {
			fmt.Println(strings.Join(f.Columns(), "\t"))
		}

		return nil
	},
}
