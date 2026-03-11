package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/braidsdev/braids/internal/extension"
	"github.com/spf13/cobra"
)

var extensionsCmd = &cobra.Command{
	Use:   "extensions",
	Short: "Manage protocol extensions",
}

var extensionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available and installed extensions",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := extension.NewManager()

		installed, err := mgr.ListInstalled()
		if err != nil {
			return err
		}
		installedMap := make(map[string]extension.InstalledExtension, len(installed))
		for _, ext := range installed {
			installedMap[ext.Protocol] = ext
		}

		// List all known extension protocols
		protocols := make([]string, 0, len(extension.ExtensionProtocols))
		for proto := range extension.ExtensionProtocols {
			protocols = append(protocols, proto)
		}
		sort.Strings(protocols)

		fmt.Println("\nEXTENSIONS:")
		for _, proto := range protocols {
			if ext, ok := installedMap[proto]; ok {
				fmt.Printf("  %-12s v%-10s installed %s\n", proto, ext.Version, ext.InstalledAt.Format("2006-01-02"))
			} else {
				fmt.Printf("  %-12s %-11s not installed\n", proto, "")
			}
		}
		fmt.Printf("\n%d extensions available, %d installed\n", len(protocols), len(installed))
		fmt.Println("\nInstall with: braids extensions install <protocol>")
		return nil
	},
}

var extensionsInstallCmd = &cobra.Command{
	Use:   "install <protocol>",
	Short: "Download and install an extension",
	Long: `Download and install an extension binary for a protocol.

Supported protocols: ` + strings.Join(extensionProtocolList(), ", "),
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		protocol := args[0]
		if !extension.IsExtensionProtocol(protocol) {
			return fmt.Errorf("unknown extension protocol %q\nAvailable: %s",
				protocol, strings.Join(extensionProtocolList(), ", "))
		}

		fmt.Printf("Installing extension %q...\n", protocol)
		mgr := extension.NewManager()
		if err := mgr.Download(protocol, false); err != nil {
			return err
		}
		fmt.Printf("Extension %q installed successfully.\n", protocol)
		return nil
	},
}

var extensionsUpdateCmd = &cobra.Command{
	Use:   "update [protocol]",
	Short: "Update installed extensions to the latest version",
	Long: `Re-download the latest version of an installed extension.
If no protocol is specified, updates all installed extensions.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := extension.NewManager()

		var protocols []string
		if len(args) > 0 {
			protocol := args[0]
			if !extension.IsExtensionProtocol(protocol) {
				return fmt.Errorf("unknown extension protocol %q", protocol)
			}
			protocols = []string{protocol}
		} else {
			// Update all installed
			installed, err := mgr.ListInstalled()
			if err != nil {
				return err
			}
			if len(installed) == 0 {
				fmt.Println("No extensions installed.")
				return nil
			}
			for _, ext := range installed {
				protocols = append(protocols, ext.Protocol)
			}
		}

		for _, proto := range protocols {
			fmt.Printf("Updating extension %q...\n", proto)
			if err := mgr.Download(proto, true); err != nil {
				fmt.Printf("  %s: %v\n", proto, err)
				continue
			}
			fmt.Printf("  %s: updated\n", proto)
		}
		return nil
	},
}

func extensionProtocolList() []string {
	protocols := make([]string, 0, len(extension.ExtensionProtocols))
	for proto := range extension.ExtensionProtocols {
		protocols = append(protocols, proto)
	}
	sort.Strings(protocols)
	return protocols
}

func init() {
	extensionsCmd.AddCommand(extensionsListCmd)
	extensionsCmd.AddCommand(extensionsInstallCmd)
	extensionsCmd.AddCommand(extensionsUpdateCmd)
	rootCmd.AddCommand(extensionsCmd)
}
