// Package formatter provides LLM-friendly markdown formatters.
package formatter

import (
	"fmt"
	"strings"
)

// FormatTrips formats a list of trips as markdown.
func FormatTrips(trips []map[string]interface{}) string {
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
func FormatTrip(trip map[string]interface{}) string {
	var sb strings.Builder

	// Trip header
	sb.WriteString(fmt.Sprintf("# %s\n\n", getString(trip, "name")))

	// Trip details
	sb.WriteString("## Details\n\n")
	sb.WriteString(fmt.Sprintf("- **ID**: `%s`\n", getString(trip, "id")))
	sb.WriteString(fmt.Sprintf("- **Purpose**: %s\n", getString(trip, "purpose")))
	sb.WriteString(fmt.Sprintf("- **Status**: %s\n", getString(trip, "status")))
	if startDate := getString(trip, "startDate"); startDate != "" {
		sb.WriteString(fmt.Sprintf("- **Start Date**: %s\n", startDate))
	}
	if endDate := getString(trip, "endDate"); endDate != "" {
		sb.WriteString(fmt.Sprintf("- **End Date**: %s\n", endDate))
	}
	if notes := getString(trip, "notes"); notes != "" {
		sb.WriteString(fmt.Sprintf("\n**Notes**: %s\n", notes))
	}

	// Items
	if items, ok := trip["items"].([]interface{}); ok && len(items) > 0 {
		sb.WriteString("\n## Items\n\n")
		for _, item := range items {
			if itemMap, ok := item.(map[string]interface{}); ok {
				sb.WriteString(FormatItem(itemMap))
				sb.WriteString("\n")
			}
		}
	}

	return sb.String()
}

func formatTripSummary(trip map[string]interface{}) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("### %s\n\n", getString(trip, "name")))
	sb.WriteString(fmt.Sprintf("- **ID**: `%s`\n", getString(trip, "id")))
	sb.WriteString(fmt.Sprintf("- **Purpose**: %s\n", getString(trip, "purpose")))
	sb.WriteString(fmt.Sprintf("- **Status**: %s\n", getString(trip, "status")))

	startDate := getString(trip, "startDate")
	endDate := getString(trip, "endDate")
	if startDate != "" && endDate != "" {
		sb.WriteString(fmt.Sprintf("- **Dates**: %s to %s\n", startDate, endDate))
	} else if startDate != "" {
		sb.WriteString(fmt.Sprintf("- **Start Date**: %s\n", startDate))
	}

	return sb.String()
}

// FormatItem formats a single item as markdown.
func FormatItem(item map[string]interface{}) string {
	itemType := getString(item, "type")

	switch itemType {
	case "flight":
		return formatFlight(item)
	case "hotel":
		return formatHotel(item)
	case "train":
		return formatTrain(item)
	case "drive":
		return formatDrive(item)
	case "event":
		return formatEvent(item)
	default:
		return formatGenericItem(item)
	}
}

func formatFlight(item map[string]interface{}) string {
	var sb strings.Builder
	sb.WriteString("#### ✈️ Flight\n\n")
	sb.WriteString(fmt.Sprintf("- **Route**: %s → %s\n", getString(item, "from"), getString(item, "to")))
	if date := getString(item, "date"); date != "" {
		sb.WriteString(fmt.Sprintf("- **Date**: %s", date))
		if time := getString(item, "time"); time != "" {
			sb.WriteString(fmt.Sprintf(" at %s", time))
		}
		sb.WriteString("\n")
	}
	if carrier := getString(item, "carrier"); carrier != "" {
		sb.WriteString(fmt.Sprintf("- **Carrier**: %s", carrier))
		if flightNumber := getString(item, "flightNumber"); flightNumber != "" {
			sb.WriteString(fmt.Sprintf(" %s", flightNumber))
		}
		sb.WriteString("\n")
	}
	if conf := getString(item, "confirmation"); conf != "" {
		sb.WriteString(fmt.Sprintf("- **Confirmation**: %s\n", conf))
	}
	sb.WriteString(fmt.Sprintf("- **ID**: `%s`\n", getString(item, "id")))
	return sb.String()
}

func formatHotel(item map[string]interface{}) string {
	var sb strings.Builder
	sb.WriteString("#### 🏨 Hotel\n\n")
	sb.WriteString(fmt.Sprintf("- **Name**: %s\n", getString(item, "name")))
	if location := getString(item, "location"); location != "" {
		sb.WriteString(fmt.Sprintf("- **Location**: %s\n", location))
	}
	checkIn := getString(item, "checkIn")
	checkOut := getString(item, "checkOut")
	if checkIn != "" && checkOut != "" {
		sb.WriteString(fmt.Sprintf("- **Dates**: %s to %s\n", checkIn, checkOut))
	}
	if conf := getString(item, "confirmation"); conf != "" {
		sb.WriteString(fmt.Sprintf("- **Confirmation**: %s\n", conf))
	}
	sb.WriteString(fmt.Sprintf("- **ID**: `%s`\n", getString(item, "id")))
	return sb.String()
}

func formatTrain(item map[string]interface{}) string {
	var sb strings.Builder
	sb.WriteString("#### 🚆 Train\n\n")
	sb.WriteString(fmt.Sprintf("- **Route**: %s → %s\n", getString(item, "from"), getString(item, "to")))
	if date := getString(item, "date"); date != "" {
		sb.WriteString(fmt.Sprintf("- **Date**: %s", date))
		if time := getString(item, "time"); time != "" {
			sb.WriteString(fmt.Sprintf(" at %s", time))
		}
		sb.WriteString("\n")
	}
	if carrier := getString(item, "carrier"); carrier != "" {
		sb.WriteString(fmt.Sprintf("- **Operator**: %s\n", carrier))
	}
	if conf := getString(item, "confirmation"); conf != "" {
		sb.WriteString(fmt.Sprintf("- **Confirmation**: %s\n", conf))
	}
	sb.WriteString(fmt.Sprintf("- **ID**: `%s`\n", getString(item, "id")))
	return sb.String()
}

func formatDrive(item map[string]interface{}) string {
	var sb strings.Builder
	sb.WriteString("#### 🚗 Drive\n\n")
	sb.WriteString(fmt.Sprintf("- **Route**: %s → %s\n", getString(item, "from"), getString(item, "to")))
	if date := getString(item, "date"); date != "" {
		sb.WriteString(fmt.Sprintf("- **Date**: %s", date))
		if time := getString(item, "time"); time != "" {
			sb.WriteString(fmt.Sprintf(" at %s", time))
		}
		sb.WriteString("\n")
	}
	sb.WriteString(fmt.Sprintf("- **ID**: `%s`\n", getString(item, "id")))
	return sb.String()
}

func formatEvent(item map[string]interface{}) string {
	var sb strings.Builder
	sb.WriteString("#### 📅 Event\n\n")
	sb.WriteString(fmt.Sprintf("- **Name**: %s\n", getString(item, "name")))
	if location := getString(item, "location"); location != "" {
		sb.WriteString(fmt.Sprintf("- **Location**: %s\n", location))
	}
	if date := getString(item, "date"); date != "" {
		sb.WriteString(fmt.Sprintf("- **Date**: %s", date))
		if time := getString(item, "time"); time != "" {
			sb.WriteString(fmt.Sprintf(" at %s", time))
		}
		sb.WriteString("\n")
	}
	if conf := getString(item, "confirmation"); conf != "" {
		sb.WriteString(fmt.Sprintf("- **Confirmation**: %s\n", conf))
	}
	sb.WriteString(fmt.Sprintf("- **ID**: `%s`\n", getString(item, "id")))
	return sb.String()
}

func formatGenericItem(item map[string]interface{}) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("#### %s\n\n", getString(item, "type")))
	sb.WriteString(fmt.Sprintf("- **ID**: `%s`\n", getString(item, "id")))
	return sb.String()
}

// FormatDocuments formats a list of documents as markdown.
func FormatDocuments(documents []map[string]interface{}) string {
	if len(documents) == 0 {
		return "No documents found."
	}

	var sb strings.Builder
	sb.WriteString("# Documents\n\n")
	for _, doc := range documents {
		sb.WriteString(fmt.Sprintf("### %s\n\n", getString(doc, "name")))
		sb.WriteString(fmt.Sprintf("- **ID**: `%s`\n", getString(doc, "id")))
		sb.WriteString(fmt.Sprintf("- **Type**: %s\n", getString(doc, "type")))
		if tripID := getString(doc, "tripId"); tripID != "" {
			sb.WriteString(fmt.Sprintf("- **Trip ID**: `%s`\n", tripID))
		} else {
			sb.WriteString("- **Trip**: (unassociated)\n")
		}
		if url := getString(doc, "url"); url != "" {
			sb.WriteString(fmt.Sprintf("- **URL**: %s\n", url))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
