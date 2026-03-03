package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	appVersion string
	appCommit  string
	configFile string
)

func Execute(version, commit string) error {
	appVersion = version
	appCommit = commit
	return rootCmd.Execute()
}

var rootCmd = &cobra.Command{
	Use:   "braids",
	Short: "Config-driven API composition",
	Long:  "Braids — Declare integrations in YAML, get a unified API.",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("braids %s (%s)\n", appVersion, appCommit)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "braids.yaml", "path to braids.yaml config file")
	rootCmd.AddCommand(versionCmd)
}
