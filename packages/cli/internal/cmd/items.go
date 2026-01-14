package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/user/travel-calendar/cli/internal/client"
	"github.com/user/travel-calendar/cli/internal/output"
)

var itemsCmd = &cobra.Command{
	Use:   "items",
	Short: "Manage trip items",
	Long:  `Add, list, and delete trip items (flights, hotels, etc.).`,
}

// items list <trip-id>
var itemsListCmd = &cobra.Command{
	Use:   "list <trip-id>",
	Short: "List items for a trip",
	Long:  `List all items for a specific trip.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		tripID, err := parseUUID(args[0])
		if err != nil {
			output.PrintError("Invalid trip ID", err)
		}

		resp, err := getClient().ListTripItemsWithResponse(getContext(), tripID)
		if err != nil {
			output.PrintError("Failed to list items", err)
		}
		if resp.StatusCode() == http.StatusNotFound {
			output.PrintError("Trip not found", nil)
		}
		if resp.StatusCode() != http.StatusOK {
			output.PrintError(fmt.Sprintf("API returned %d", resp.StatusCode()), nil)
		}

		output.PrintItems(*resp.JSON200)
	},
}

// items add <trip-id> <type>
var itemsAddCmd = &cobra.Command{
	Use:   "add <trip-id> <type>",
	Short: "Add an item to a trip",
	Long: `Add an item to a trip. Types: flight, hotel, train, drive, event.

Examples:
  travel items add <trip-id> flight --from JFK --to LAX --date 2025-03-01 --time 10:00 --carrier Delta --flight DL123
  travel items add <trip-id> hotel --name "Hilton" --location "New York" --check-in 2025-03-01 --check-out 2025-03-05
  travel items add <trip-id> event --name "Conference" --location "Convention Center" --date 2025-03-02`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		tripID, err := parseUUID(args[0])
		if err != nil {
			output.PrintError("Invalid trip ID", err)
		}

		itemType := args[1]
		validTypes := []string{"flight", "hotel", "train", "drive", "event"}
		valid := false
		for _, t := range validTypes {
			if itemType == t {
				valid = true
				break
			}
		}
		if !valid {
			output.PrintError(fmt.Sprintf("Invalid item type '%s'. Valid types: flight, hotel, train, drive, event", itemType), nil)
		}

		req := client.CreateItemRequest{
			Type: client.ItemType(itemType),
		}

		// Common flags
		if date, _ := cmd.Flags().GetString("date"); date != "" {
			d, err := parseDate(date)
			if err != nil {
				output.PrintError("Invalid date", err)
			}
			req.Date = &d
		}
		if time, _ := cmd.Flags().GetString("time"); time != "" {
			req.Time = &time
		}
		if conf, _ := cmd.Flags().GetString("confirmation"); conf != "" {
			req.Confirmation = &conf
		}
		if notes, _ := cmd.Flags().GetString("notes"); notes != "" {
			req.Notes = &notes
		}

		// Transport flags
		if from, _ := cmd.Flags().GetString("from"); from != "" {
			req.From = &from
		}
		if to, _ := cmd.Flags().GetString("to"); to != "" {
			req.To = &to
		}
		if carrier, _ := cmd.Flags().GetString("carrier"); carrier != "" {
			req.Carrier = &carrier
		}
		if flight, _ := cmd.Flags().GetString("flight"); flight != "" {
			req.FlightNumber = &flight
		}

		// Hotel/Event flags
		if name, _ := cmd.Flags().GetString("name"); name != "" {
			req.Name = &name
		}
		if location, _ := cmd.Flags().GetString("location"); location != "" {
			req.Location = &location
		}
		if checkIn, _ := cmd.Flags().GetString("check-in"); checkIn != "" {
			d, err := parseDate(checkIn)
			if err != nil {
				output.PrintError("Invalid check-in date", err)
			}
			req.CheckIn = &d
		}
		if checkOut, _ := cmd.Flags().GetString("check-out"); checkOut != "" {
			d, err := parseDate(checkOut)
			if err != nil {
				output.PrintError("Invalid check-out date", err)
			}
			req.CheckOut = &d
		}

		resp, err := getClient().CreateTripItemWithResponse(getContext(), tripID, req)
		if err != nil {
			output.PrintError("Failed to add item", err)
		}
		if resp.StatusCode() == http.StatusNotFound {
			output.PrintError("Trip not found", nil)
		}
		if resp.StatusCode() != http.StatusCreated {
			output.PrintError(fmt.Sprintf("API returned %d", resp.StatusCode()), nil)
		}

		output.PrintSuccess(fmt.Sprintf("Item added: %s", resp.JSON201.Id))
		output.PrintItem(*resp.JSON201)
	},
}

// items delete <id>
var itemsDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an item",
	Long:  `Delete an item from a trip.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id, err := parseUUID(args[0])
		if err != nil {
			output.PrintError("Invalid item ID", err)
		}

		resp, err := getClient().DeleteItemWithResponse(getContext(), id)
		if err != nil {
			output.PrintError("Failed to delete item", err)
		}
		if resp.StatusCode() == http.StatusNotFound {
			output.PrintError("Item not found", nil)
		}
		if resp.StatusCode() != http.StatusNoContent {
			output.PrintError(fmt.Sprintf("API returned %d", resp.StatusCode()), nil)
		}

		output.PrintSuccess("Item deleted")
	},
}

func init() {
	rootCmd.AddCommand(itemsCmd)

	// items list
	itemsCmd.AddCommand(itemsListCmd)

	// items add
	itemsCmd.AddCommand(itemsAddCmd)
	// Common flags
	itemsAddCmd.Flags().String("date", "", "Date (YYYY-MM-DD)")
	itemsAddCmd.Flags().String("time", "", "Time (HH:MM)")
	itemsAddCmd.Flags().String("confirmation", "", "Confirmation number")
	itemsAddCmd.Flags().String("notes", "", "Additional notes")
	// Transport flags
	itemsAddCmd.Flags().String("from", "", "Origin (for flights, trains, drives)")
	itemsAddCmd.Flags().String("to", "", "Destination (for flights, trains, drives)")
	itemsAddCmd.Flags().String("carrier", "", "Carrier/airline (for flights, trains)")
	itemsAddCmd.Flags().String("flight", "", "Flight number (for flights)")
	// Hotel/Event flags
	itemsAddCmd.Flags().String("name", "", "Name (for hotels, events)")
	itemsAddCmd.Flags().String("location", "", "Location (for hotels, events)")
	itemsAddCmd.Flags().String("check-in", "", "Check-in date (for hotels)")
	itemsAddCmd.Flags().String("check-out", "", "Check-out date (for hotels)")

	// items delete
	itemsCmd.AddCommand(itemsDeleteCmd)
}
