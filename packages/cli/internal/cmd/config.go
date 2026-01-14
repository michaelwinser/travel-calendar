package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/user/travel-calendar/cli/internal/client"
	"github.com/user/travel-calendar/cli/internal/output"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `Get and set configuration values like home and work locations.`,
}

// config get <key>
var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a config value",
	Long: `Get a configuration value. Available keys:
  home  - Home location (default when not traveling)
  work  - Work location (optional)`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]

		resp, err := getClient().GetBaseLocationsWithResponse(getContext())
		if err != nil {
			output.PrintError("Failed to get config", err)
		}
		if resp.StatusCode() != http.StatusOK {
			output.PrintError(fmt.Sprintf("API returned %d", resp.StatusCode()), nil)
		}

		switch key {
		case "home":
			if resp.JSON200.Home != nil {
				output.PrintConfig("home", *resp.JSON200.Home)
			} else {
				output.PrintConfig("home", "(not set)")
			}
		case "work":
			if resp.JSON200.Work != nil {
				output.PrintConfig("work", *resp.JSON200.Work)
			} else {
				output.PrintConfig("work", "(not set)")
			}
		default:
			output.PrintError(fmt.Sprintf("Unknown config key: %s. Available: home, work", key), nil)
		}
	},
}

// config set <key> <value>
var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Long: `Set a configuration value. Available keys:
  home  - Home location (default when not traveling)
  work  - Work location (optional)`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		value := args[1]

		req := client.SetBaseLocationsRequest{}
		switch key {
		case "home":
			req.Home = &value
		case "work":
			req.Work = &value
		default:
			output.PrintError(fmt.Sprintf("Unknown config key: %s. Available: home, work", key), nil)
		}

		resp, err := getClient().SetBaseLocationsWithResponse(getContext(), req)
		if err != nil {
			output.PrintError("Failed to set config", err)
		}
		if resp.StatusCode() != http.StatusOK {
			output.PrintError(fmt.Sprintf("API returned %d", resp.StatusCode()), nil)
		}

		output.PrintSuccess(fmt.Sprintf("Set %s = %s", key, value))
	},
}

// config unset <key>
var configUnsetCmd = &cobra.Command{
	Use:   "unset <key>",
	Short: "Remove a config value",
	Long: `Remove a configuration value. Available keys:
  home  - Home location
  work  - Work location`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]

		empty := ""
		req := client.SetBaseLocationsRequest{}
		switch key {
		case "home":
			req.Home = &empty
		case "work":
			req.Work = &empty
		default:
			output.PrintError(fmt.Sprintf("Unknown config key: %s. Available: home, work", key), nil)
		}

		resp, err := getClient().SetBaseLocationsWithResponse(getContext(), req)
		if err != nil {
			output.PrintError("Failed to unset config", err)
		}
		if resp.StatusCode() != http.StatusOK {
			output.PrintError(fmt.Sprintf("API returned %d", resp.StatusCode()), nil)
		}

		output.PrintSuccess(fmt.Sprintf("Unset %s", key))
	},
}

// config list
var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all config values",
	Long:  `List all configuration values.`,
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := getClient().GetBaseLocationsWithResponse(getContext())
		if err != nil {
			output.PrintError("Failed to get config", err)
		}
		if resp.StatusCode() != http.StatusOK {
			output.PrintError(fmt.Sprintf("API returned %d", resp.StatusCode()), nil)
		}

		output.PrintBaseLocations(*resp.JSON200)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configUnsetCmd)
	configCmd.AddCommand(configListCmd)
}
