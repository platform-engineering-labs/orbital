package cli

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/platform-engineering-labs/orbital"
	"github.com/spf13/cobra"
)

func init() {
	Publish.Flags().StringArray("channel", []string{}, "channel")
	Publish.Flags().String("repo", "", "repository")
	Publish.Flags().String("work-path", "", "work path")
}

var Publish = &cobra.Command{
	Use:     "publish [Opkg Files]",
	Short:   "publish packages to a repository",
	GroupID: "publishing",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		repo, _ := cmd.Flags().GetString("repo")
		workPath, _ := cmd.Flags().GetString("work-path")
		channels, _ := cmd.Flags().GetStringArray("channel")

		if cmd.Flags().NArg() == 0 {
			return fmt.Errorf("must specify at least one package to publish")
		}

		orb, err := orbital.Dynamic(cfgPath, slog.New(Logger))
		if err != nil {
			return err
		}

		published, pruned, err := orb.Publish.Publish(repo, workPath, cmd.Flags().Args(), channels)
		if err != nil {
			return err
		}

		fmt.Println(fmt.Sprintf("published: %d", len(published)))
		fmt.Println(fmt.Sprintf("pruned: %d", len(pruned)))
		fmt.Println(fmt.Sprintf("repo: %s", repo))
		fmt.Println(fmt.Sprintf("channels: %s", strings.Join(channels, ",")))

		return nil
	},
}
