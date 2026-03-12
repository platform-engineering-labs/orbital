package cli

import (
	"fmt"
	"log/slog"

	"github.com/platform-engineering-labs/orbital"
	"github.com/spf13/cobra"
)

func init() {
	Transaction.AddCommand(TransactionList)
}

var Transaction = &cobra.Command{
	Use:     "transaction",
	Short:   "manage transactions",
	GroupID: "manage",
}

var TransactionList = &cobra.Command{
	Use:   "list",
	Short: "list transactions",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		orb, err := orbital.Dynamic(cfgPath, slog.New(Logger))
		if err != nil {
			return err
		}

		transactions, err := orb.Transaction.List()
		if err != nil {
			return err
		}

		for id, operations := range transactions {
			fmt.Println(id)

			for _, op := range operations {
				fmt.Println(fmt.Sprintf(" - %s\t%s\t%s", op.Operation, op.PkgId, op.Date))
			}
			fmt.Println()
		}

		return nil
	},
}
