package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/braidsdev/braids/internal/gateway"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the braids gateway",
	RunE: func(cmd *cobra.Command, args []string) error {
		gw, err := gateway.New(configFile)
		if err != nil {
			return fmt.Errorf("initializing gateway: %w", err)
		}

		// Graceful shutdown on SIGINT/SIGTERM
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

		go func() {
			<-stop
			log.Println("Shutting down...")
			gw.Shutdown(context.Background())
		}()

		return gw.Start()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
