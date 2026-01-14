// Package output provides formatting utilities for CLI output.
package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/user/travel-calendar/cli/internal/client"
)

// JSONOutput controls whether to output JSON format.
var JSONOutput bool

// PrintTrips prints a list of trips.
func PrintTrips(trips []client.Trip) {
	if JSONOutput {
		printJSON(trips)
		return
	}

	if len(trips) == 0 {
		fmt.Println("No trips found.")
		return
	}

	for _, trip := range trips {
		printTripSummary(trip)
		fmt.Println()
	}
}

// PrintTrip prints a single trip with details.
func PrintTrip(trip client.Trip) {
	if JSONOutput {
		printJSON(trip)
		return
	}

	fmt.Printf("Trip: %s\n", trip.Name)
	fmt.Printf("  ID:       %s\n", trip.Id)
	fmt.Printf("  Purpose:  %s\n", trip.Purpose)
	fmt.Printf("  Status:   %s\n", trip.Status)
	if trip.StartDate != nil {
		fmt.Printf("  Start:    %s\n", trip.StartDate.Format("2006-01-02"))
	}
	if trip.EndDate != nil {
		fmt.Printf("  End:      %s\n", trip.EndDate.Format("2006-01-02"))
	}
	if trip.Notes != nil && *trip.Notes != "" {
		fmt.Printf("  Notes:    %s\n", *trip.Notes)
	}

	if trip.Items != nil && len(*trip.Items) > 0 {
		fmt.Println("\nItems:")
		for _, item := range *trip.Items {
			printItemSummary(item)
		}
	}
}

func printTripSummary(trip client.Trip) {
	var dateRange string
	if trip.StartDate != nil && trip.EndDate != nil {
		dateRange = fmt.Sprintf("%s to %s", trip.StartDate.Format("2006-01-02"), trip.EndDate.Format("2006-01-02"))
	} else if trip.StartDate != nil {
		dateRange = fmt.Sprintf("from %s", trip.StartDate.Format("2006-01-02"))
	} else {
		dateRange = "(no dates)"
	}

	fmt.Printf("%s [%s]\n", trip.Name, trip.Status)
	fmt.Printf("  ID: %s\n", trip.Id)
	fmt.Printf("  Purpose: %s | Dates: %s\n", trip.Purpose, dateRange)
}

// PrintItems prints a list of items.
func PrintItems(items []client.Item) {
	if JSONOutput {
		printJSON(items)
		return
	}

	if len(items) == 0 {
		fmt.Println("No items found.")
		return
	}

	for _, item := range items {
		printItemSummary(item)
	}
}

// PrintItem prints a single item.
func PrintItem(item client.Item) {
	if JSONOutput {
		printJSON(item)
		return
	}
	printItemSummary(item)
}

func printItemSummary(item client.Item) {
	switch item.Type {
	case client.Flight:
		fmt.Printf("  [Flight] %s → %s", getString(item.From), getString(item.To))
		if item.Date != nil {
			fmt.Printf(" on %s", item.Date.Format("2006-01-02"))
		}
		if item.Time != nil {
			fmt.Printf(" at %s", *item.Time)
		}
		if item.Carrier != nil {
			fmt.Printf(" (%s", *item.Carrier)
			if item.FlightNumber != nil {
				fmt.Printf(" %s", *item.FlightNumber)
			}
			fmt.Printf(")")
		}
		fmt.Println()
	case client.Hotel:
		fmt.Printf("  [Hotel] %s", getString(item.Name))
		if item.Location != nil {
			fmt.Printf(" - %s", *item.Location)
		}
		if item.CheckIn != nil && item.CheckOut != nil {
			fmt.Printf(" (%s to %s)", item.CheckIn.Format("2006-01-02"), item.CheckOut.Format("2006-01-02"))
		}
		fmt.Println()
	case client.Train:
		fmt.Printf("  [Train] %s → %s", getString(item.From), getString(item.To))
		if item.Date != nil {
			fmt.Printf(" on %s", item.Date.Format("2006-01-02"))
		}
		if item.Time != nil {
			fmt.Printf(" at %s", *item.Time)
		}
		fmt.Println()
	case client.Drive:
		fmt.Printf("  [Drive] %s → %s", getString(item.From), getString(item.To))
		if item.Date != nil {
			fmt.Printf(" on %s", item.Date.Format("2006-01-02"))
		}
		fmt.Println()
	case client.Event:
		fmt.Printf("  [Event] %s", getString(item.Name))
		if item.Location != nil {
			fmt.Printf(" @ %s", *item.Location)
		}
		if item.Date != nil {
			fmt.Printf(" on %s", item.Date.Format("2006-01-02"))
		}
		fmt.Println()
	default:
		fmt.Printf("  [%s] ID: %s\n", item.Type, item.Id)
	}
	fmt.Printf("    ID: %s\n", item.Id)
}

// PrintDocuments prints a list of documents.
func PrintDocuments(docs []client.Document) {
	if JSONOutput {
		printJSON(docs)
		return
	}

	if len(docs) == 0 {
		fmt.Println("No documents found.")
		return
	}

	for _, doc := range docs {
		fmt.Printf("%s [%s]\n", doc.Name, doc.Type)
		fmt.Printf("  ID: %s\n", doc.Id)
		if doc.TripId != nil {
			fmt.Printf("  Trip ID: %s\n", *doc.TripId)
		} else {
			fmt.Println("  (not associated with a trip)")
		}
		if doc.Url != nil {
			fmt.Printf("  URL: %s\n", *doc.Url)
		}
		fmt.Println()
	}
}

// PrintSuccess prints a success message.
// In JSON mode, success messages are suppressed to allow clean JSON output.
func PrintSuccess(message string) {
	if JSONOutput {
		return
	}
	fmt.Println(message)
}

// PrintError prints an error message and exits.
func PrintError(message string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s: %v\n", message, err)
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	}
	os.Exit(1)
}

func printJSON(data interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		PrintError("Failed to encode JSON", err)
	}
}

func getString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// PrintConfig prints a single config key-value pair.
func PrintConfig(key, value string) {
	if JSONOutput {
		printJSON(map[string]string{key: value})
		return
	}
	fmt.Printf("%s = %s\n", key, value)
}

// PrintBaseLocations prints all base location configuration.
func PrintBaseLocations(locs client.BaseLocations) {
	if JSONOutput {
		printJSON(locs)
		return
	}

	fmt.Println("Base Locations:")
	if locs.Home != nil {
		fmt.Printf("  home = %s\n", *locs.Home)
	} else {
		fmt.Println("  home = (not set)")
	}
	if locs.Work != nil {
		fmt.Printf("  work = %s\n", *locs.Work)
	} else {
		fmt.Println("  work = (not set)")
	}
}

// PrintTripLocations prints location assignments for a trip.
func PrintTripLocations(locs []client.TripDayLocation) {
	if JSONOutput {
		printJSON(locs)
		return
	}

	if len(locs) == 0 {
		fmt.Println("No locations set for this trip.")
		return
	}

	fmt.Println("Trip Locations:")
	for _, dayLoc := range locs {
		fmt.Printf("  %s: %s\n", dayLoc.Date.Format("2006-01-02"), strings.Join(dayLoc.Locations, ", "))
	}
}

// PrintLocationOnDate prints the location for a single date.
func PrintLocationOnDate(loc client.LocationOnDateResponse) {
	if JSONOutput {
		printJSON(loc)
		return
	}

	fmt.Printf("Location on %s:\n", loc.Date.Format("2006-01-02"))
	fmt.Printf("  %s\n", strings.Join(loc.Locations, ", "))

	sourceDesc := ""
	switch loc.Source.Type {
	case client.LocationSourceTypeHome:
		sourceDesc = "home"
	case client.LocationSourceTypeWork:
		sourceDesc = "work"
	case client.LocationSourceTypeTrip:
		if loc.Source.TripName != nil {
			sourceDesc = fmt.Sprintf("trip: %s", *loc.Source.TripName)
		} else {
			sourceDesc = "trip"
		}
	}
	fmt.Printf("  (source: %s)\n", sourceDesc)
}

// PrintLocationRange prints location segments for a date range.
func PrintLocationRange(segments []client.LocationRangeSegment) {
	if JSONOutput {
		printJSON(segments)
		return
	}

	if len(segments) == 0 {
		fmt.Println("No location data for this range.")
		return
	}

	fmt.Println("Location Timeline:")
	for _, seg := range segments {
		dateRange := ""
		if seg.StartDate.Format("2006-01-02") == seg.EndDate.Format("2006-01-02") {
			dateRange = seg.StartDate.Format("2006-01-02")
		} else {
			dateRange = fmt.Sprintf("%s to %s", seg.StartDate.Format("2006-01-02"), seg.EndDate.Format("2006-01-02"))
		}

		sourceDesc := ""
		switch seg.Source.Type {
		case client.LocationSourceTypeHome:
			sourceDesc = "home"
		case client.LocationSourceTypeWork:
			sourceDesc = "work"
		case client.LocationSourceTypeTrip:
			if seg.Source.TripName != nil {
				sourceDesc = fmt.Sprintf("trip: %s", *seg.Source.TripName)
			} else {
				sourceDesc = "trip"
			}
		}

		fmt.Printf("  %s: %s (%s)\n", dateRange, strings.Join(seg.Locations, ", "), sourceDesc)
	}
}
