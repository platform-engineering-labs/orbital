package cli

import (
	"fmt"
	"log/slog"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/platform-engineering-labs/orbital"
	"github.com/platform-engineering-labs/orbital/platform"
	"github.com/platform-engineering-labs/orbital/platform/arch"
	"github.com/platform-engineering-labs/orbital/platform/os"
	"github.com/spf13/cobra"
)

func init() {
	Tree.AddCommand(TreeDestroy)

	TreeInit.Flags().BoolP("force", "f", false, "force overwrite of existing tree")
	TreeInit.Flags().BoolP("arch", "a", false, "architecture")
	TreeInit.Flags().BoolP("os", "o", false, "operating system")
	Tree.AddCommand(TreeInit)

	Tree.AddCommand(TreeList)

	Tree.AddCommand(TreeSwitch)
}

var Tree = &cobra.Command{
	Use:     "tree",
	Short:   "manage ops trees",
	GroupID: "treepkg",
}

var TreeDestroy = &cobra.Command{
	Use:   "destroy [Tree Name]",
	Short: "destroy ops tree",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		name := cmd.Flags().Arg(0)
		if name == "" {
			return fmt.Errorf("must specify a name for the ops tree")
		}

		orb, err := orbital.Dynamic(slog.New(Logger), cfgPath)
		if err != nil {
			return err
		}

		tree, err := orb.Tree.Get(name)
		if err != nil {
			return err
		}

		fmt.Println(fmt.Sprintf("this will delete the ops tree %s: %s", tree.Name, tree.Path))

		confirm := false
		err = huh.NewConfirm().
			Title("Are you sure?").
			Affirmative("Yes!").
			Negative("No.").
			Value(&confirm).
			Run()
		if err != nil {
			return err
		}

		if confirm {
			entry, err := orb.Tree.Destroy(name)
			if err != nil {
				return err
			}

			fmt.Println(fmt.Sprintf("destroyed ops tree: %s - %s", entry.Name, entry.Path))
		}

		return nil
	},
}

var TreeInit = &cobra.Command{
	Use:   "init [Tree Name]",
	Short: "create a new ops tree",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")
		force, _ := cmd.Flags().GetBool("force")

		a, _ := cmd.Flags().GetString("arch")
		o, _ := cmd.Flags().GetString("os")

		var pltfrm *platform.Platform
		if o != "" && a != "" {
			pltfrm = &platform.Platform{
				OS:   os.OS(o),
				Arch: arch.Arch(a),
			}
		} else {
			pltfrm = platform.Current()
		}

		if !pltfrm.Supported() {
			return fmt.Errorf("platform not supported: %s", pltfrm.String())
		}

		name := cmd.Flags().Arg(0)
		if name == "" {
			return fmt.Errorf("must specify a name for the ops tree")
		}

		orb, err := orbital.Dynamic(slog.New(Logger), cfgPath)
		if err != nil {
			return err
		}

		entry, err := orb.Tree.Init(name, pltfrm, force)
		if err != nil {
			return err
		}

		fmt.Println(fmt.Sprintf("created orbital tree: %s - %s", entry.Name, entry.Path))

		return nil
	},
}

var TreeList = &cobra.Command{
	Use:   "list",
	Short: "list ops trees",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		orb, err := orbital.Dynamic(slog.New(Logger), cfgPath)
		if err != nil {
			return err
		}

		trees, err := orb.Tree.List()
		if err != nil {
			return fmt.Errorf("error: %s", err)
		}

		tbl := table.New()
		tbl.Headers("Name", "Path", "Current")

		for _, tree := range trees {
			current := ""
			if tree.Current {
				current = " *"
			}

			tbl.Row(tree.Name, tree.Path, current)
		}

		fmt.Println(tbl.Render())

		return nil
	},
}

var TreeSwitch = &cobra.Command{
	Use:   "switch [Tree Name]",
	Short: "switch current ops tree",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		name := cmd.Flags().Arg(0)
		if name == "" {
			return fmt.Errorf("must specify a name for the ops tree")
		}

		orb, err := orbital.Dynamic(slog.New(Logger), cfgPath)
		if err != nil {
			return err
		}

		_, err = orb.Tree.Get(name)
		if err != nil {
			return fmt.Errorf("error: %s", err)
		}

		err = orb.Tree.Switch(name)
		if err != nil {
			return fmt.Errorf("error: %s", err)
		}

		return nil
	},
}
