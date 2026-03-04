package cli

import (
	"fmt"
	"path/filepath"

	"github.com/braidsdev/braids/internal/config"
	"github.com/braidsdev/braids/internal/connector"
	"github.com/spf13/cobra"
)

var connectorsCmd = &cobra.Command{
	Use:   "connectors",
	Short: "Manage connectors",
}

var connectorsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Download or refresh cached OpenAPI specs for all connectors",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configFile)
		if err != nil {
			return err
		}

		configDir := filepath.Dir(configFile)

		for name, ref := range cfg.Connectors {
			def, err := connector.LoadDefWithoutSpec(ref.Type, configDir, ref.Path)
			if err != nil {
				fmt.Printf("  %s: error loading definition: %v\n", name, err)
				continue
			}
			if def.OpenAPIURL == "" {
				continue
			}
			fmt.Printf("  %s: downloading %s ...\n", name, def.OpenAPIURL)
			if err := connector.RefreshCachedSpec(name, def.OpenAPIURL); err != nil {
				fmt.Printf("  %s: %v\n", name, err)
				continue
			}
			fmt.Printf("  %s: cached\n", name)
		}
		return nil
	},
}

func init() {
	connectorsCmd.AddCommand(connectorsUpdateCmd)
	rootCmd.AddCommand(connectorsCmd)
}
