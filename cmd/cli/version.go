package cli

import (
	"fmt"

	"github.com/platform-engineering-labs/orbital/runtime"
	"github.com/spf13/cobra"
)

var Version = &cobra.Command{
	Use:   "version",
	Short: "display version information",

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("version: ", runtime.BuildVersion())
		fmt.Println("commit: ", runtime.BuildCommitRev())
	},
}
