package cli

import (
	"fmt"
	"log/slog"

	"github.com/platform-engineering-labs/orbital"
	"github.com/spf13/cobra"
)

func init() {
	RepoContents.Flags().Bool("all", false, "Show packages for all platforms and channels")
	RepoContents.Flags().Bool("refresh", false, "refresh metadata")
	Repo.AddCommand(RepoContents)

	RepoInit.Flags().String("work-path", "", "work path")
	Repo.AddCommand(RepoInit)
	Repo.AddCommand(RepoList)
}

var Repo = &cobra.Command{
	Use:     "repo",
	Short:   "manage repositories",
	GroupID: "repos",
}

var RepoContents = &cobra.Command{
	Use:   "contents [Repo Name]",
	Short: "show repo contents",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		all, _ := cmd.Flags().GetBool("all")
		refresh, _ := cmd.Flags().GetBool("refresh")

		if cmd.Flags().Arg(0) == "" {
			return fmt.Errorf("repo name is required")
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

		rp, err := orb.Repo.Contents(cmd.Flags().Arg(0), all)
		if err != nil {
			return err
		}
		if rp == nil {
			return fmt.Errorf("repo: %s not found", cmd.Flags().Arg(0))
		}

		inventory := rp.Inventory()
		for _, pltfrm := range rp.Platforms() {
			fmt.Printf("%s\n", pltfrm.String())
			for _, inv := range inventory {
				if inv.Platform == pltfrm {
					if len(inv.Packages) > 0 {
						fmt.Printf("  %s\n", inv.Channel.Name)
					}
					for _, header := range inv.Packages {
						fmt.Printf("   %s\n", header.Id().String())
					}
				}
			}
		}

		return nil
	},
}

var RepoInit = &cobra.Command{
	Use:   "init [Repo Name]",
	Short: "initialize a repository",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		workPath, _ := cmd.Flags().GetString("work-path")

		if cmd.Flags().Arg(0) == "" {
			return fmt.Errorf("repo name is required")
		}

		orb, err := orbital.Dynamic(slog.New(Logger), cfgPath)
		if err != nil {
			return err
		}

		uri, err := orb.Repo.Init(cmd.Flags().Arg(0), workPath)
		if err != nil {
			return err
		}

		//TODO add method to return safe uri
		uri.Fragment = ""
		fmt.Println(fmt.Sprintf("created orbital repo: %s", uri.String()))

		return nil
	},
}

var RepoList = &cobra.Command{
	Use:   "list",
	Short: `list configured repositories`,

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		orb, err := orbital.Dynamic(slog.New(Logger), cfgPath)
		if err != nil {
			return err
		}

		for _, repo := range orb.Repo.List() {
			status := "disabled"
			if repo.Enabled {
				status = "enabled"
			}
			fmt.Println(fmt.Sprintf("%s\t%s", repo.SafeUri(), status))
		}

		return nil
	},
}
