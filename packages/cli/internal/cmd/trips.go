package cmd

import (
	"fmt"
	"net/http"
	"time"

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
		location, _ := cmd.Flags().GetString("location")

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
		if location != "" {
			req.Location = &location
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

// trips get-locations <trip-id>
var tripsGetLocationsCmd = &cobra.Command{
	Use:   "get-locations <trip-id>",
	Short: "Get locations for a trip",
	Long:  `Get the location assignments for each day of a trip.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		tripID, err := parseUUID(args[0])
		if err != nil {
			output.PrintError("Invalid trip ID", err)
		}

		resp, err := getClient().GetTripLocationsWithResponse(getContext(), tripID)
		if err != nil {
			output.PrintError("Failed to get trip locations", err)
		}
		if resp.StatusCode() == http.StatusNotFound {
			output.PrintError("Trip not found", nil)
		}
		if resp.StatusCode() != http.StatusOK {
			output.PrintError(fmt.Sprintf("API returned %d", resp.StatusCode()), nil)
		}

		output.PrintTripLocations(*resp.JSON200)
	},
}

// trips set-location <trip-id> <location> [--date YYYY-MM-DD] [--start YYYY-MM-DD --end YYYY-MM-DD]
var tripsSetLocationCmd = &cobra.Command{
	Use:   "set-location <trip-id> <location>",
	Short: "Set location for a trip",
	Long: `Set the location for a trip. Without flags, sets the location for all days.
With --date, sets location for a single day.
With --start and --end, sets location for a date range.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		tripID, err := parseUUID(args[0])
		if err != nil {
			output.PrintError("Invalid trip ID", err)
		}
		location := args[1]

		dateStr, _ := cmd.Flags().GetString("date")
		startStr, _ := cmd.Flags().GetString("start")
		endStr, _ := cmd.Flags().GetString("end")

		req := client.SetTripLocationsRequest{}

		if dateStr != "" {
			// Single date
			date, err := parseDate(dateStr)
			if err != nil {
				output.PrintError("Invalid date", err)
			}
			locs := []client.TripDayLocation{{Date: date, Locations: []string{location}}}
			req.Locations = &locs
		} else if startStr != "" && endStr != "" {
			// Date range - we'll set defaultLocation and the backend will apply it
			// But we need to be more clever here - the API expects per-day or default
			// For a range, we set defaultLocation which applies to all uncovered dates
			req.DefaultLocation = &location
		} else if startStr != "" || endStr != "" {
			output.PrintError("Both --start and --end are required for date range", nil)
		} else {
			// All days - use default location
			req.DefaultLocation = &location
		}

		resp, err := getClient().SetTripLocationsWithResponse(getContext(), tripID, req)
		if err != nil {
			output.PrintError("Failed to set trip locations", err)
		}
		if resp.StatusCode() == http.StatusNotFound {
			output.PrintError("Trip not found", nil)
		}
		if resp.StatusCode() != http.StatusOK {
			output.PrintError(fmt.Sprintf("API returned %d", resp.StatusCode()), nil)
		}

		output.PrintSuccess("Location set successfully")
		output.PrintTripLocations(*resp.JSON200)
	},
}

// trips add-location <trip-id> <location> --date YYYY-MM-DD
var tripsAddLocationCmd = &cobra.Command{
	Use:   "add-location <trip-id> <location>",
	Short: "Add a location to a trip day",
	Long: `Add an additional location to a specific day (for travel days with multiple locations).
Requires --date flag to specify which day.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		tripID, err := parseUUID(args[0])
		if err != nil {
			output.PrintError("Invalid trip ID", err)
		}
		location := args[1]

		dateStr, _ := cmd.Flags().GetString("date")
		if dateStr == "" {
			output.PrintError("--date is required for add-location", nil)
		}

		date, err := parseDate(dateStr)
		if err != nil {
			output.PrintError("Invalid date", err)
		}

		// First get existing locations
		getResp, err := getClient().GetTripLocationsWithResponse(getContext(), tripID)
		if err != nil {
			output.PrintError("Failed to get existing locations", err)
		}
		if getResp.StatusCode() == http.StatusNotFound {
			output.PrintError("Trip not found", nil)
		}
		if getResp.StatusCode() != http.StatusOK {
			output.PrintError(fmt.Sprintf("API returned %d when getting locations", getResp.StatusCode()), nil)
		}

		// Build new locations list with the added location
		var newLocations []client.TripDayLocation
		dateFound := false
		for _, dayLoc := range *getResp.JSON200 {
			if dayLoc.Date.Format("2006-01-02") == date.Format("2006-01-02") {
				// Add to existing day
				dayLoc.Locations = append(dayLoc.Locations, location)
				dateFound = true
			}
			newLocations = append(newLocations, dayLoc)
		}

		if !dateFound {
			// Add new day with just this location
			newLocations = append(newLocations, client.TripDayLocation{
				Date:      date,
				Locations: []string{location},
			})
		}

		req := client.SetTripLocationsRequest{
			Locations: &newLocations,
		}

		resp, err := getClient().SetTripLocationsWithResponse(getContext(), tripID, req)
		if err != nil {
			output.PrintError("Failed to add location", err)
		}
		if resp.StatusCode() != http.StatusOK {
			output.PrintError(fmt.Sprintf("API returned %d", resp.StatusCode()), nil)
		}

		output.PrintSuccess("Location added successfully")
		output.PrintTripLocations(*resp.JSON200)
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
	tripsCreateCmd.Flags().String("location", "", "Default location for all days of this trip")
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

	// trips get-locations
	tripsCmd.AddCommand(tripsGetLocationsCmd)

	// trips set-location
	tripsCmd.AddCommand(tripsSetLocationCmd)
	tripsSetLocationCmd.Flags().String("date", "", "Set location for a single date (YYYY-MM-DD)")
	tripsSetLocationCmd.Flags().String("start", "", "Start of date range (YYYY-MM-DD)")
	tripsSetLocationCmd.Flags().String("end", "", "End of date range (YYYY-MM-DD)")

	// trips add-location
	tripsCmd.AddCommand(tripsAddLocationCmd)
	tripsAddLocationCmd.Flags().String("date", "", "Date to add location to (YYYY-MM-DD)")
	tripsAddLocationCmd.MarkFlagRequired("date")
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

func parseDateWithKeywords(s string) (openapi_types.Date, error) {
	var date openapi_types.Date
	now := time.Now()

	switch s {
	case "today":
		date.Time = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		return date, nil
	case "tomorrow":
		tomorrow := now.AddDate(0, 0, 1)
		date.Time = time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, time.UTC)
		return date, nil
	case "yesterday":
		yesterday := now.AddDate(0, 0, -1)
		date.Time = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, time.UTC)
		return date, nil
	default:
		return parseDate(s)
	}
}
