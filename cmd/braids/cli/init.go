package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new braids project",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Initializing braids project...")
		// TODO: scaffold braids.yaml
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
