package cli

import (
	"fmt"
	"log/slog"

	"github.com/platform-engineering-labs/orbital"
	"github.com/platform-engineering-labs/orbital/platform"
	"github.com/platform-engineering-labs/orbital/platform/arch"
	"github.com/platform-engineering-labs/orbital/platform/os"
	"github.com/spf13/cobra"
)

func init() {
	Fetch.Flags().String("os", "", "override OS")
	Fetch.Flags().String("arch", "", "override architecture")
	Fetch.Flags().Bool("refresh", false, "refresh metadata")
}

var Fetch = &cobra.Command{
	Use:     "fetch [package ...]",
	Short:   "fetch a package from a repository",
	GroupID: "publishing",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")
		fos, _ := cmd.Flags().GetString("os")
		farch, _ := cmd.Flags().GetString("arch")
		refresh, _ := cmd.Flags().GetBool("refresh")

		if cmd.Flags().NArg() == 0 {
			return fmt.Errorf("package argument required")
		}

		orb, err := orbital.Dynamic(cfgPath, slog.New(Logger))
		if err != nil {
			return err
		}

		pltfrm := platform.Current()
		if fos != "" && farch != "" {
			pltfrm = &platform.Platform{OS: os.OS(fos), Arch: arch.Arch(farch)}
		}

		if refresh {
			err := orb.Refresh()
			if err != nil {
				return err
			}
		}

		err = orb.Publish.Fetch(cmd.Flags().Args(), pltfrm)
		if err != nil {
			return err
		}

		return nil
	},
}
