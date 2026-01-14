package mcp

import (
	"fmt"
	"strings"

	"github.com/user/travel-calendar/backend/internal/api"
)

// FormatTrips formats a list of trips as markdown.
func FormatTrips(trips []api.Trip) string {
	if len(trips) == 0 {
		return "No trips found."
	}

	var sb strings.Builder
	for i, trip := range trips {
		if i > 0 {
			sb.WriteString("\n---\n\n")
		}
		sb.WriteString(formatTripSummary(trip))
	}
	return sb.String()
}

// FormatTrip formats a single trip with its items as markdown.
func FormatTrip(trip api.Trip) string {
	var sb strings.Builder

	// Trip header
	sb.WriteString(fmt.Sprintf("# %s\n\n", trip.Name))

	// Trip details
	sb.WriteString("## Details\n\n")
	sb.WriteString(fmt.Sprintf("- **ID**: `%s`\n", trip.Id))
	sb.WriteString(fmt.Sprintf("- **Purpose**: %s\n", trip.Purpose))
	sb.WriteString(fmt.Sprintf("- **Status**: %s\n", trip.Status))
	if trip.StartDate != nil {
		sb.WriteString(fmt.Sprintf("- **Start Date**: %s\n", trip.StartDate.Format("2006-01-02")))
	}
	if trip.EndDate != nil {
		sb.WriteString(fmt.Sprintf("- **End Date**: %s\n", trip.EndDate.Format("2006-01-02")))
	}
	if trip.Notes != nil && *trip.Notes != "" {
		sb.WriteString(fmt.Sprintf("\n**Notes**: %s\n", *trip.Notes))
	}

	// Items
	if trip.Items != nil && len(*trip.Items) > 0 {
		sb.WriteString("\n## Items\n\n")
		for _, item := range *trip.Items {
			sb.WriteString(FormatItem(item))
			sb.WriteString("\n")
		}
	}

	// Locations
	if trip.Locations != nil && len(*trip.Locations) > 0 {
		sb.WriteString("\n## Locations\n\n")
		for _, loc := range *trip.Locations {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", loc.Date.Format("2006-01-02"), strings.Join(loc.Locations, ", ")))
		}
	}

	return sb.String()
}

func formatTripSummary(trip api.Trip) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("### %s\n\n", trip.Name))
	sb.WriteString(fmt.Sprintf("- **ID**: `%s`\n", trip.Id))
	sb.WriteString(fmt.Sprintf("- **Purpose**: %s\n", trip.Purpose))
	sb.WriteString(fmt.Sprintf("- **Status**: %s\n", trip.Status))

	if trip.StartDate != nil && trip.EndDate != nil {
		sb.WriteString(fmt.Sprintf("- **Dates**: %s to %s\n",
			trip.StartDate.Format("2006-01-02"),
			trip.EndDate.Format("2006-01-02")))
	} else if trip.StartDate != nil {
		sb.WriteString(fmt.Sprintf("- **Start Date**: %s\n", trip.StartDate.Format("2006-01-02")))
	}

	return sb.String()
}

// FormatItem formats a single item as markdown.
func FormatItem(item api.Item) string {
	switch item.Type {
	case api.Flight:
		return formatFlight(item)
	case api.Hotel:
		return formatHotel(item)
	case api.Train:
		return formatTrain(item)
	case api.Drive:
		return formatDrive(item)
	case api.Event:
		return formatEvent(item)
	default:
		return formatGenericItem(item)
	}
}

func formatFlight(item api.Item) string {
	var sb strings.Builder
	sb.WriteString("#### Flight\n\n")
	if item.From != nil && item.To != nil {
		sb.WriteString(fmt.Sprintf("- **Route**: %s -> %s\n", *item.From, *item.To))
	}
	if item.Date != nil {
		sb.WriteString(fmt.Sprintf("- **Date**: %s", item.Date.Format("2006-01-02")))
		if item.Time != nil {
			sb.WriteString(fmt.Sprintf(" at %s", *item.Time))
		}
		sb.WriteString("\n")
	}
	if item.Carrier != nil {
		sb.WriteString(fmt.Sprintf("- **Carrier**: %s", *item.Carrier))
		if item.FlightNumber != nil {
			sb.WriteString(fmt.Sprintf(" %s", *item.FlightNumber))
		}
		sb.WriteString("\n")
	}
	if item.Confirmation != nil {
		sb.WriteString(fmt.Sprintf("- **Confirmation**: %s\n", *item.Confirmation))
	}
	sb.WriteString(fmt.Sprintf("- **ID**: `%s`\n", item.Id))
	return sb.String()
}

func formatHotel(item api.Item) string {
	var sb strings.Builder
	sb.WriteString("#### Hotel\n\n")
	if item.Name != nil {
		sb.WriteString(fmt.Sprintf("- **Name**: %s\n", *item.Name))
	}
	if item.Location != nil {
		sb.WriteString(fmt.Sprintf("- **Location**: %s\n", *item.Location))
	}
	if item.CheckIn != nil && item.CheckOut != nil {
		sb.WriteString(fmt.Sprintf("- **Dates**: %s to %s\n",
			item.CheckIn.Format("2006-01-02"),
			item.CheckOut.Format("2006-01-02")))
	}
	if item.Confirmation != nil {
		sb.WriteString(fmt.Sprintf("- **Confirmation**: %s\n", *item.Confirmation))
	}
	sb.WriteString(fmt.Sprintf("- **ID**: `%s`\n", item.Id))
	return sb.String()
}

func formatTrain(item api.Item) string {
	var sb strings.Builder
	sb.WriteString("#### Train\n\n")
	if item.From != nil && item.To != nil {
		sb.WriteString(fmt.Sprintf("- **Route**: %s -> %s\n", *item.From, *item.To))
	}
	if item.Date != nil {
		sb.WriteString(fmt.Sprintf("- **Date**: %s", item.Date.Format("2006-01-02")))
		if item.Time != nil {
			sb.WriteString(fmt.Sprintf(" at %s", *item.Time))
		}
		sb.WriteString("\n")
	}
	if item.Carrier != nil {
		sb.WriteString(fmt.Sprintf("- **Operator**: %s\n", *item.Carrier))
	}
	if item.Confirmation != nil {
		sb.WriteString(fmt.Sprintf("- **Confirmation**: %s\n", *item.Confirmation))
	}
	sb.WriteString(fmt.Sprintf("- **ID**: `%s`\n", item.Id))
	return sb.String()
}

func formatDrive(item api.Item) string {
	var sb strings.Builder
	sb.WriteString("#### Drive\n\n")
	if item.From != nil && item.To != nil {
		sb.WriteString(fmt.Sprintf("- **Route**: %s -> %s\n", *item.From, *item.To))
	}
	if item.Date != nil {
		sb.WriteString(fmt.Sprintf("- **Date**: %s", item.Date.Format("2006-01-02")))
		if item.Time != nil {
			sb.WriteString(fmt.Sprintf(" at %s", *item.Time))
		}
		sb.WriteString("\n")
	}
	sb.WriteString(fmt.Sprintf("- **ID**: `%s`\n", item.Id))
	return sb.String()
}

func formatEvent(item api.Item) string {
	var sb strings.Builder
	sb.WriteString("#### Event\n\n")
	if item.Name != nil {
		sb.WriteString(fmt.Sprintf("- **Name**: %s\n", *item.Name))
	}
	if item.Location != nil {
		sb.WriteString(fmt.Sprintf("- **Location**: %s\n", *item.Location))
	}
	if item.Date != nil {
		sb.WriteString(fmt.Sprintf("- **Date**: %s", item.Date.Format("2006-01-02")))
		if item.Time != nil {
			sb.WriteString(fmt.Sprintf(" at %s", *item.Time))
		}
		sb.WriteString("\n")
	}
	if item.Confirmation != nil {
		sb.WriteString(fmt.Sprintf("- **Confirmation**: %s\n", *item.Confirmation))
	}
	sb.WriteString(fmt.Sprintf("- **ID**: `%s`\n", item.Id))
	return sb.String()
}

func formatGenericItem(item api.Item) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("#### %s\n\n", item.Type))
	sb.WriteString(fmt.Sprintf("- **ID**: `%s`\n", item.Id))
	return sb.String()
}

// FormatDocuments formats a list of documents as markdown.
func FormatDocuments(documents []api.Document) string {
	if len(documents) == 0 {
		return "No documents found."
	}

	var sb strings.Builder
	sb.WriteString("# Documents\n\n")
	for _, doc := range documents {
		sb.WriteString(fmt.Sprintf("### %s\n\n", doc.Name))
		sb.WriteString(fmt.Sprintf("- **ID**: `%s`\n", doc.Id))
		sb.WriteString(fmt.Sprintf("- **Type**: %s\n", doc.Type))
		if doc.TripId != nil {
			sb.WriteString(fmt.Sprintf("- **Trip ID**: `%s`\n", *doc.TripId))
		} else {
			sb.WriteString("- **Trip**: (unassociated)\n")
		}
		if doc.Url != nil {
			sb.WriteString(fmt.Sprintf("- **URL**: %s\n", *doc.Url))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// FormatLocationOnDate formats a single date location response as markdown.
func FormatLocationOnDate(loc api.LocationOnDateResponse) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## Location on %s\n\n", loc.Date.Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("**Location(s)**: %s\n\n", strings.Join(loc.Locations, ", ")))

	switch loc.Source.Type {
	case api.LocationSourceTypeHome:
		sb.WriteString("*You'll be at home.*\n")
	case api.LocationSourceTypeWork:
		sb.WriteString("*You'll be at work.*\n")
	case api.LocationSourceTypeTrip:
		if loc.Source.TripName != nil {
			sb.WriteString(fmt.Sprintf("*Source: trip \"%s\"*\n", *loc.Source.TripName))
		} else {
			sb.WriteString("*Source: a scheduled trip*\n")
		}
	}

	return sb.String()
}

// FormatLocationRange formats a location range timeline as markdown.
func FormatLocationRange(segments []api.LocationRangeSegment) string {
	if len(segments) == 0 {
		return "No location data for this range."
	}

	var sb strings.Builder
	sb.WriteString("## Location Timeline\n\n")

	for _, seg := range segments {
		startDate := seg.StartDate.Format("2006-01-02")
		endDate := seg.EndDate.Format("2006-01-02")

		// Format date range
		var dateRange string
		if startDate == endDate {
			dateRange = startDate
		} else {
			dateRange = fmt.Sprintf("%s to %s", startDate, endDate)
		}

		// Format source
		sourceDesc := ""
		switch seg.Source.Type {
		case api.LocationSourceTypeHome:
			sourceDesc = "home"
		case api.LocationSourceTypeWork:
			sourceDesc = "work"
		case api.LocationSourceTypeTrip:
			if seg.Source.TripName != nil {
				sourceDesc = fmt.Sprintf("trip: %s", *seg.Source.TripName)
			} else {
				sourceDesc = "trip"
			}
		}

		sb.WriteString(fmt.Sprintf("- **%s**: %s", dateRange, strings.Join(seg.Locations, ", ")))
		if sourceDesc != "" {
			sb.WriteString(fmt.Sprintf(" (%s)", sourceDesc))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// FormatTripLocations formats trip day locations as markdown.
func FormatTripLocations(locations []api.TripDayLocation) string {
	if len(locations) == 0 {
		return "No locations set for this trip."
	}

	var sb strings.Builder
	sb.WriteString("## Trip Locations\n\n")

	for _, dayLoc := range locations {
		sb.WriteString(fmt.Sprintf("- **%s**: %s\n",
			dayLoc.Date.Format("2006-01-02"),
			strings.Join(dayLoc.Locations, ", ")))
	}

	return sb.String()
}

// FormatBaseLocations formats base location configuration as markdown.
func FormatBaseLocations(locs api.BaseLocations) string {
	var sb strings.Builder
	sb.WriteString("## Base Locations\n\n")

	if locs.Home != nil && *locs.Home != "" {
		sb.WriteString(fmt.Sprintf("- **Home**: %s\n", *locs.Home))
	} else {
		sb.WriteString("- **Home**: (not set)\n")
	}

	if locs.Work != nil && *locs.Work != "" {
		sb.WriteString(fmt.Sprintf("- **Work**: %s\n", *locs.Work))
	} else {
		sb.WriteString("- **Work**: (not set)\n")
	}

	return sb.String()
}
