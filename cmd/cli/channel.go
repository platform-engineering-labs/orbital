package cli

import (
	"fmt"
	"log/slog"

	"github.com/platform-engineering-labs/orbital"
	"github.com/platform-engineering-labs/orbital/ops"
	"github.com/spf13/cobra"
)

func init() {
	Channel.Flags().StringArray("channel", []string{}, "channel")
	Channel.Flags().String("repo", "", "repository")
}

var Channel = &cobra.Command{
	Use:     "channel [Opkg Id]",
	Short:   "add a package to a channel in a repository",
	GroupID: "publishing",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		repo, _ := cmd.Flags().GetString("repo")
		channels, _ := cmd.Flags().GetStringArray("channel")

		if repo == "" {
			return fmt.Errorf("repo flag is required")
		}

		if len(channels) == 0 {
			return fmt.Errorf("channel flag is required")
		}

		orb, err := orbital.Dynamic(cfgPath, slog.New(Logger))
		if err != nil {
			return err
		}

		if cmd.Flags().Arg(0) == "" {
			return fmt.Errorf("pkg id arg is required")
		}

		id := &ops.Id{}
		err = id.FromString(cmd.Flags().Arg(0))

		err = orb.Publish.Channel(repo, channels, id)
		if err != nil {
			return err
		}

		fmt.Println("done.")

		return nil
	},
}
