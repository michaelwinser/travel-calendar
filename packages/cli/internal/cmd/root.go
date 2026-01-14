// Package cmd provides the CLI command implementations.
package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/user/travel-calendar/cli/internal/client"
	"github.com/user/travel-calendar/cli/internal/output"
)

var (
	apiURL     string
	jsonOutput bool
	apiClient  *client.ClientWithResponses
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "travel",
	Short: "Travel Calendar CLI",
	Long: `Travel Calendar CLI - Manage your travel trips, items, and documents.

Examples:
  travel trips list                     # List all trips
  travel trips list --upcoming          # List upcoming trips
  travel trips create --name "NYC 2025" --purpose vacation
  travel trips get <id>                 # Get trip details
  travel items add <trip-id> flight --from JFK --to LAX
  travel documents list                 # List all documents`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Set JSON output mode
		output.JSONOutput = jsonOutput

		// Create API client
		var err error
		apiClient, err = client.NewClientWithResponses(apiURL, client.WithHTTPClient(http.DefaultClient))
		if err != nil {
			return fmt.Errorf("failed to create API client: %w", err)
		}
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "http://localhost:3000", "API server URL")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	viper.BindPFlag("api_url", rootCmd.PersistentFlags().Lookup("api-url"))
}

func initConfig() {
	viper.SetEnvPrefix("TRAVEL")
	viper.AutomaticEnv()

	if url := viper.GetString("api_url"); url != "" {
		apiURL = url
	}
}

// getClient returns the API client.
func getClient() *client.ClientWithResponses {
	return apiClient
}

// getContext returns a context for API calls.
func getContext() context.Context {
	return context.Background()
}
