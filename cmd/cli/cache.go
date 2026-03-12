package cli

import (
	"log/slog"

	"github.com/platform-engineering-labs/orbital"
	"github.com/spf13/cobra"
)

func init() {
	Cache.AddCommand(CacheClean)
	Cache.AddCommand(CacheClear)
}

var Cache = &cobra.Command{
	Use:     "cache",
	Short:   "manage cache",
	GroupID: "manage",
}

var CacheClean = &cobra.Command{
	Use:   "clean",
	Short: "clean Opkg files",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		orb, err := orbital.Dynamic(slog.New(Logger), cfgPath)
		if err != nil {
			return err
		}

		return orb.Cache.Clean()
	},
}

var CacheClear = &cobra.Command{
	Use:   "clear",
	Short: "clear Opkg files and metadata files",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		orb, err := orbital.Dynamic(slog.New(Logger), cfgPath)
		if err != nil {
			return err
		}

		return orb.Cache.Clear()
	},
}
