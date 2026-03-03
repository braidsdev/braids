package cli

import (
	"fmt"

	"github.com/braidsdev/braids/internal/config"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate braids.yaml configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configFile)
		if err != nil {
			return err
		}

		if err := config.Validate(cfg); err != nil {
			return err
		}

		fmt.Printf("%s is valid\n", configFile)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
