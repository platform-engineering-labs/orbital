package cli

import (
	"fmt"
	"log/slog"
	osg "os"
	"path/filepath"

	"github.com/platform-engineering-labs/orbital"
	"github.com/platform-engineering-labs/orbital/schema/names"
	"github.com/platform-engineering-labs/orbital/x/os"
	"github.com/spf13/cobra"
)

var Setup = &cobra.Command{
	Use:     "setup",
	Short:   "setup ops for publishing",
	Long:    `setup ops for publishing utilizing the following ENV vars: \n OPS_PKG_CRT OPS_PKG_KEY OPS_TREE_CONF`,
	GroupID: "publishing",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		orb, err := orbital.Dynamic(slog.New(Logger), cfgPath)
		if err != nil {
			return err
		}

		if !os.LookupEnv(
			"OPS_PKG_CRT",
			"OPS_PKG_KEY",
			"OPS_TREE_CONF",
		) {
			return fmt.Errorf("the following env vars must be set: OPS_PKG_CRT, OPS_PKG_KEY, OPS_TREE_CONF")
		}

		err = orb.Pki.KeyPairImport("env", "OPS_PKG_CRT", "OPS_PKG_KEY")
		if err != nil {
			return err
		}

		treeConf, _ := osg.LookupEnv("OPS_TREE_CONF")

		confPath := filepath.Join(orb.Tree.Current().Path, names.TreeDataDir, names.TreeConfigFile)
		return osg.WriteFile(confPath, []byte(treeConf), 0644)
	},
}
