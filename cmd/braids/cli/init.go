package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const starterConfig = `version: "1"

connectors:
  stripe:
    type: stripe
    config:
      api_key: ${STRIPE_API_KEY}

  shopify:
    type: shopify
    config:
      store: ${SHOPIFY_STORE}
      access_token: ${SHOPIFY_ACCESS_TOKEN}

schemas:
  customer:
    merge_on: email
    conflict_resolution: prefer_latest
    fields:
      id:
        type: string
      email:
        type: string
      name:
        type: string
      created_at:
        type: datetime

endpoints:
  /customers:
    schema: customer
    sources:
      - connector: stripe
        resource: customers
        mapping:
          id: "'stripe_' + id"
          email: email
          name: name
          created_at: created
      - connector: shopify
        resource: customers
        mapping:
          id: "'shopify_' + id"
          email: email
          name: first_name + ' ' + last_name
          created_at: created_at

server:
  port: 8080
  hot_reload: true
`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new braids project",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := os.Stat(configFile); err == nil {
			return fmt.Errorf("%s already exists", configFile)
		}

		if err := os.WriteFile(configFile, []byte(starterConfig), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", configFile, err)
		}

		fmt.Printf("Created %s\n", configFile)
		fmt.Println("Next steps:")
		fmt.Println("  1. Set environment variables: STRIPE_API_KEY, SHOPIFY_STORE, SHOPIFY_ACCESS_TOKEN")
		fmt.Println("  2. Run: braids validate")
		fmt.Println("  3. Run: braids serve")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
