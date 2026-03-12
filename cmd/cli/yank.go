package cli

import (
	"fmt"
	"log/slog"

	"github.com/platform-engineering-labs/orbital"
	"github.com/spf13/cobra"
)

func init() {
	Yank.Flags().String("repo", "", "repository")
	Yank.Flags().String("work-path", "", "work path")
}

var Yank = &cobra.Command{
	Use:     "yank [Opkg Id]",
	Short:   "yank a package from a repository",
	GroupID: "publishing",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		repo, _ := cmd.Flags().GetString("repo")
		workPath, _ := cmd.Flags().GetString("work-path")

		if cmd.Flags().NArg() == 0 {
			return fmt.Errorf("must specify at least one package to publish")
		}

		orb, err := orbital.Dynamic(slog.New(Logger), cfgPath)
		if err != nil {
			return err
		}

		err = orb.Publish.Yank(repo, cmd.Flags().Arg(0), workPath)
		if err != nil {
			return err
		}

		return nil
	},
}
