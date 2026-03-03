package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the braids gateway",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Starting braids gateway...")
		// TODO: load config, start gateway
		return nil
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
