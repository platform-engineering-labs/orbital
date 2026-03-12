package cli

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/platform-engineering-labs/orbital"
	"github.com/spf13/cobra"
)

var Plan = &cobra.Command{
	Use:     "plan [action] [package ...]",
	Short:   "show the operations a package request will generate",
	GroupID: "manage",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		if cmd.Flags().NArg() < 2 {
			return fmt.Errorf("must specify an action and at least one package to plan")
		}

		orb, err := orbital.Dynamic(cfgPath, slog.New(Logger))
		if err != nil {
			return err
		}
		
		operations, err := orb.Plan(cmd.Flags().Arg(0), cmd.Flags().Args()[1:]...)
		if err != nil {
			return err
		}

		for _, op := range operations {
			fmt.Println(strings.ToUpper(string(op.Operation)), "\t", op.Package.Id().String())
		}

		return nil
	},
}
