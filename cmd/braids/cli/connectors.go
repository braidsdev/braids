package cli

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

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

var connectorsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all built-in connectors",
	RunE: func(cmd *cobra.Command, args []string) error {
		connectors, err := connector.ListBuiltinConnectors()
		if err != nil {
			return err
		}

		// Group by category
		byCategory := make(map[string][]connector.ConnectorInfo)
		for _, c := range connectors {
			cat := c.Category
			if cat == "" {
				cat = "other"
			}
			byCategory[cat] = append(byCategory[cat], c)
		}

		// Sort categories
		categories := make([]string, 0, len(byCategory))
		for cat := range byCategory {
			categories = append(categories, cat)
		}
		sort.Strings(categories)

		for _, cat := range categories {
			fmt.Printf("\n%s:\n", strings.ToUpper(cat))
			// Sort connectors within category
			items := byCategory[cat]
			sort.Slice(items, func(i, j int) bool {
				return items[i].Name < items[j].Name
			})
			for _, c := range items {
				fmt.Printf("  %-16s %s\n", c.Name, c.Description)
			}
		}
		fmt.Printf("\n%d connectors available\n", len(connectors))
		return nil
	},
}

func init() {
	connectorsCmd.AddCommand(connectorsUpdateCmd)
	connectorsCmd.AddCommand(connectorsListCmd)
	rootCmd.AddCommand(connectorsCmd)
}
