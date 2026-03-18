package cli

import (
	"github.com/spf13/cobra"
)

func init() {
	Root.AddGroup(&cobra.Group{
		ID:    "manage",
		Title: "Manage Current Tree",
	})
	Root.AddGroup(&cobra.Group{
		ID:    "publishing",
		Title: "Opkg publishing/fetching/channeling",
	})
	Root.AddGroup(&cobra.Group{
		ID:    "repos",
		Title: "Repository Management",
	})
	Root.AddGroup(&cobra.Group{
		ID:    "treepkg",
		Title: "Trees and Opkgs",
	})

	Root.AddCommand(Cache)
	Root.AddCommand(Channel)
	Root.AddCommand(Contents)
	Root.AddCommand(Fetch)
	Root.AddCommand(Freeze)
	Root.AddCommand(Info)
	Root.AddCommand(Install)
	Root.AddCommand(List)
	Root.AddCommand(Opkg)
	Root.AddCommand(Publish)
	Root.AddCommand(Pki)
	Root.AddCommand(Plan)
	Root.AddCommand(Refresh)
	Root.AddCommand(Remove)
	Root.AddCommand(Repo)
	Root.AddCommand(Setup)
	Root.AddCommand(Status)
	Root.AddCommand(Thaw)
	Root.AddCommand(Transaction)
	Root.AddCommand(Tree)
	Root.AddCommand(Update)
	Root.AddCommand(Version)
	Root.AddCommand(Yank)

	Root.PersistentFlags().StringP("config", "c", "", "config file path")
}

var Root = &cobra.Command{
	Use:   "ops",
	Short: "ops - the orbital package system",
}
