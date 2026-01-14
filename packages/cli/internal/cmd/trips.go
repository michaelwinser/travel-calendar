package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/user/travel-calendar/cli/internal/client"
	"github.com/user/travel-calendar/cli/internal/output"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

var tripsCmd = &cobra.Command{
	Use:   "trips",
	Short: "Manage trips",
	Long:  `Create, list, update, and delete trips.`,
}

// trips list
var tripsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List trips",
	Long:  `List all trips with optional filters.`,
	Run: func(cmd *cobra.Command, args []string) {
		upcoming, _ := cmd.Flags().GetBool("upcoming")
		past, _ := cmd.Flags().GetBool("past")
		purpose, _ := cmd.Flags().GetString("purpose")

		params := &client.ListTripsParams{}
		if upcoming {
			params.Upcoming = &upcoming
		}
		if past {
			params.Past = &past
		}
		if purpose != "" {
			p := client.TripPurpose(purpose)
			params.Purpose = &p
		}

		resp, err := getClient().ListTripsWithResponse(getContext(), params)
		if err != nil {
			output.PrintError("Failed to list trips", err)
		}
		if resp.StatusCode() != http.StatusOK {
			output.PrintError(fmt.Sprintf("API returned %d", resp.StatusCode()), nil)
		}

		output.PrintTrips(*resp.JSON200)
	},
}

// trips get <id>
var tripsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get trip details",
	Long:  `Get detailed information about a specific trip.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id, err := parseUUID(args[0])
		if err != nil {
			output.PrintError("Invalid trip ID", err)
		}

		resp, err := getClient().GetTripWithResponse(getContext(), id)
		if err != nil {
			output.PrintError("Failed to get trip", err)
		}
		if resp.StatusCode() == http.StatusNotFound {
			output.PrintError("Trip not found", nil)
		}
		if resp.StatusCode() != http.StatusOK {
			output.PrintError(fmt.Sprintf("API returned %d", resp.StatusCode()), nil)
		}

		output.PrintTrip(*resp.JSON200)
	},
}

// trips create
var tripsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new trip",
	Long:  `Create a new trip with the specified details.`,
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		purpose, _ := cmd.Flags().GetString("purpose")
		startDate, _ := cmd.Flags().GetString("start")
		endDate, _ := cmd.Flags().GetString("end")
		notes, _ := cmd.Flags().GetString("notes")

		if name == "" || purpose == "" {
			output.PrintError("--name and --purpose are required", nil)
		}

		req := client.CreateTripRequest{
			Name:    name,
			Purpose: client.TripPurpose(purpose),
		}
		if startDate != "" {
			date, err := parseDate(startDate)
			if err != nil {
				output.PrintError("Invalid start date", err)
			}
			req.StartDate = &date
		}
		if endDate != "" {
			date, err := parseDate(endDate)
			if err != nil {
				output.PrintError("Invalid end date", err)
			}
			req.EndDate = &date
		}
		if notes != "" {
			req.Notes = &notes
		}

		resp, err := getClient().CreateTripWithResponse(getContext(), req)
		if err != nil {
			output.PrintError("Failed to create trip", err)
		}
		if resp.StatusCode() != http.StatusCreated {
			output.PrintError(fmt.Sprintf("API returned %d", resp.StatusCode()), nil)
		}

		output.PrintSuccess(fmt.Sprintf("Trip created: %s", resp.JSON201.Id))
		output.PrintTrip(*resp.JSON201)
	},
}

// trips update <id>
var tripsUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a trip",
	Long:  `Update an existing trip.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id, err := parseUUID(args[0])
		if err != nil {
			output.PrintError("Invalid trip ID", err)
		}

		req := client.UpdateTripRequest{}
		hasUpdate := false

		if name, _ := cmd.Flags().GetString("name"); name != "" {
			req.Name = &name
			hasUpdate = true
		}
		if purpose, _ := cmd.Flags().GetString("purpose"); purpose != "" {
			p := client.TripPurpose(purpose)
			req.Purpose = &p
			hasUpdate = true
		}
		if status, _ := cmd.Flags().GetString("status"); status != "" {
			s := client.TripStatus(status)
			req.Status = &s
			hasUpdate = true
		}
		if startDate, _ := cmd.Flags().GetString("start"); startDate != "" {
			date, err := parseDate(startDate)
			if err != nil {
				output.PrintError("Invalid start date", err)
			}
			req.StartDate = &date
			hasUpdate = true
		}
		if endDate, _ := cmd.Flags().GetString("end"); endDate != "" {
			date, err := parseDate(endDate)
			if err != nil {
				output.PrintError("Invalid end date", err)
			}
			req.EndDate = &date
			hasUpdate = true
		}
		if notes, _ := cmd.Flags().GetString("notes"); notes != "" {
			req.Notes = &notes
			hasUpdate = true
		}

		if !hasUpdate {
			output.PrintError("No updates specified", nil)
		}

		resp, err := getClient().UpdateTripWithResponse(getContext(), id, req)
		if err != nil {
			output.PrintError("Failed to update trip", err)
		}
		if resp.StatusCode() == http.StatusNotFound {
			output.PrintError("Trip not found", nil)
		}
		if resp.StatusCode() != http.StatusOK {
			output.PrintError(fmt.Sprintf("API returned %d", resp.StatusCode()), nil)
		}

		output.PrintSuccess("Trip updated")
		output.PrintTrip(*resp.JSON200)
	},
}

// trips delete <id>
var tripsDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a trip",
	Long:  `Delete a trip and all its items.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id, err := parseUUID(args[0])
		if err != nil {
			output.PrintError("Invalid trip ID", err)
		}

		resp, err := getClient().DeleteTripWithResponse(getContext(), id)
		if err != nil {
			output.PrintError("Failed to delete trip", err)
		}
		if resp.StatusCode() == http.StatusNotFound {
			output.PrintError("Trip not found", nil)
		}
		if resp.StatusCode() != http.StatusNoContent {
			output.PrintError(fmt.Sprintf("API returned %d", resp.StatusCode()), nil)
		}

		output.PrintSuccess("Trip deleted")
	},
}

// trips search <query>
var tripsSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search trips",
	Long:  `Search trips by name or notes.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]

		params := &client.SearchTripsParams{Q: query}
		resp, err := getClient().SearchTripsWithResponse(getContext(), params)
		if err != nil {
			output.PrintError("Failed to search trips", err)
		}
		if resp.StatusCode() != http.StatusOK {
			output.PrintError(fmt.Sprintf("API returned %d", resp.StatusCode()), nil)
		}

		output.PrintTrips(*resp.JSON200)
	},
}

func init() {
	rootCmd.AddCommand(tripsCmd)

	// trips list
	tripsCmd.AddCommand(tripsListCmd)
	tripsListCmd.Flags().Bool("upcoming", false, "Show only upcoming trips")
	tripsListCmd.Flags().Bool("past", false, "Show only past trips")
	tripsListCmd.Flags().String("purpose", "", "Filter by purpose (business, vacation, conference, family, other)")

	// trips get
	tripsCmd.AddCommand(tripsGetCmd)

	// trips create
	tripsCmd.AddCommand(tripsCreateCmd)
	tripsCreateCmd.Flags().String("name", "", "Trip name (required)")
	tripsCreateCmd.Flags().String("purpose", "", "Trip purpose: business, vacation, conference, family, other (required)")
	tripsCreateCmd.Flags().String("start", "", "Start date (YYYY-MM-DD)")
	tripsCreateCmd.Flags().String("end", "", "End date (YYYY-MM-DD)")
	tripsCreateCmd.Flags().String("notes", "", "Additional notes")
	tripsCreateCmd.MarkFlagRequired("name")
	tripsCreateCmd.MarkFlagRequired("purpose")

	// trips update
	tripsCmd.AddCommand(tripsUpdateCmd)
	tripsUpdateCmd.Flags().String("name", "", "New trip name")
	tripsUpdateCmd.Flags().String("purpose", "", "New trip purpose")
	tripsUpdateCmd.Flags().String("status", "", "Trip status: planning, confirmed, in_progress, completed, cancelled")
	tripsUpdateCmd.Flags().String("start", "", "New start date (YYYY-MM-DD)")
	tripsUpdateCmd.Flags().String("end", "", "New end date (YYYY-MM-DD)")
	tripsUpdateCmd.Flags().String("notes", "", "New notes")

	// trips delete
	tripsCmd.AddCommand(tripsDeleteCmd)

	// trips search
	tripsCmd.AddCommand(tripsSearchCmd)
}

// Helper functions

func parseUUID(s string) (openapi_types.UUID, error) {
	var uuid openapi_types.UUID
	err := uuid.UnmarshalText([]byte(s))
	return uuid, err
}

func parseDate(s string) (openapi_types.Date, error) {
	var date openapi_types.Date
	err := date.UnmarshalText([]byte(s))
	return date, err
}
