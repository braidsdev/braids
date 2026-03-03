package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate braids.yaml configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Validating braids.yaml...")
		// TODO: parse and validate config
		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
