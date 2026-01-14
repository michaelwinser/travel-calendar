package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/user/travel-calendar/cli/internal/client"
	"github.com/user/travel-calendar/cli/internal/output"
)

var locationCmd = &cobra.Command{
	Use:   "location",
	Short: "Query location information",
	Long:  `Query where you will be on a specific date or date range.`,
}

// location on <date>
var locationOnCmd = &cobra.Command{
	Use:   "on <date>",
	Short: "Get location on a specific date",
	Long: `Get your location on a specific date.

Examples:
  travel location on 2025-01-30
  travel location on today
  travel location on tomorrow`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dateStr := args[0]

		// Parse special date keywords
		date, err := parseDateWithKeywords(dateStr)
		if err != nil {
			output.PrintError("Invalid date", err)
		}

		resp, err := getClient().GetLocationOnDateWithResponse(getContext(), date)
		if err != nil {
			output.PrintError("Failed to get location", err)
		}
		if resp.StatusCode() != http.StatusOK {
			output.PrintError(fmt.Sprintf("API returned %d", resp.StatusCode()), nil)
		}

		output.PrintLocationOnDate(*resp.JSON200)
	},
}

// location from <date> to <date>
var locationFromCmd = &cobra.Command{
	Use:   "from <start-date> to <end-date>",
	Short: "Get locations for a date range",
	Long: `Get your locations for a date range.

Examples:
  travel location from 2025-01-01 to 2025-01-31
  travel location from today to 2025-02-28`,
	Args: cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		fromStr := args[0]
		// args[1] should be "to"
		toStr := args[2]

		if args[1] != "to" {
			output.PrintError("Usage: travel location from <start-date> to <end-date>", nil)
		}

		from, err := parseDateWithKeywords(fromStr)
		if err != nil {
			output.PrintError("Invalid start date", err)
		}

		to, err := parseDateWithKeywords(toStr)
		if err != nil {
			output.PrintError("Invalid end date", err)
		}

		params := &client.GetLocationRangeParams{
			From: from,
			To:   to,
		}

		resp, err := getClient().GetLocationRangeWithResponse(getContext(), params)
		if err != nil {
			output.PrintError("Failed to get locations", err)
		}
		if resp.StatusCode() == http.StatusBadRequest {
			output.PrintError("Invalid date range (end must be after start)", nil)
		}
		if resp.StatusCode() != http.StatusOK {
			output.PrintError(fmt.Sprintf("API returned %d", resp.StatusCode()), nil)
		}

		output.PrintLocationRange(*resp.JSON200)
	},
}

func init() {
	rootCmd.AddCommand(locationCmd)
	locationCmd.AddCommand(locationOnCmd)
	locationCmd.AddCommand(locationFromCmd)
}
