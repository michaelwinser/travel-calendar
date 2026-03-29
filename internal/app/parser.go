package app

import (
	"strings"
	"time"
	"unicode"

	"github.com/michaelwinser/travel-calendar/gazetteer"
)

// Confidence levels for parsed fields.
const (
	ConfHigh   = "high"
	ConfMedium = "medium"
	ConfLow    = "low"
)

// ParsedResult is the internal representation returned by Parse.
type ParsedResult struct {
	Title      string
	Type       string
	StartDate  *time.Time
	EndDate    *time.Time
	Location   string
	Confidence map[string]string // field name → confidence level
	Unparsed   []string
}

// months maps lowercase month names and abbreviations to month numbers.
var months = map[string]time.Month{
	"jan": time.January, "january": time.January,
	"feb": time.February, "february": time.February,
	"mar": time.March, "march": time.March,
	"apr": time.April, "april": time.April,
	"may": time.May,
	"jun": time.June, "june": time.June,
	"jul": time.July, "july": time.July,
	"aug": time.August, "august": time.August,
	"sep": time.September, "september": time.September,
	"oct": time.October, "october": time.October,
	"nov": time.November, "november": time.November,
	"dec": time.December, "december": time.December,
}

// travelKeywords trigger type=travel.
var travelKeywords = map[string]bool{
	"flight": true, "fly": true, "flying": true, "train": true,
}

// conferenceKeywords trigger type=conference.
var conferenceKeywords = map[string]bool{
	"conference": true, "summit": true, "conf": true, "symposium": true,
}

// vacationKeywords trigger type=vacation.
var vacationKeywords = map[string]bool{
	"vacation": true, "holiday": true, "break": true,
}

// commitmentKeywords trigger type=commitment.
var commitmentKeywords = map[string]bool{
	"dentist": true, "doctor": true, "appointment": true, "meeting": true,
}

type parsedDate struct {
	t          time.Time
	hasYear    bool
	startToken int
	endToken   int // exclusive
}

// Parse parses freeform text into a proposed activity.
// today is the reference date for year inference.
func Parse(text string, today time.Time) ParsedResult {
	tokens := tokenize(text)
	n := len(tokens)
	consumed := make([]bool, n)
	conf := map[string]string{}

	// --- Pass 0: Find place tokens via gazetteer ---
	var placeTokens []gazetteer.PlaceToken
	gaz, gazErr := gazetteer.Get()
	if gazErr == nil {
		placeTokens = gaz.FindPlaceTokens(tokens, consumed)
	}
	// Also check for "Home" (common user-defined place, not in gazetteer)
	for i, tok := range tokens {
		if !consumed[i] && strings.ToLower(tok) == "home" {
			// Check it's not already found by gazetteer
			alreadyFound := false
			for _, pt := range placeTokens {
				if pt.Start == i {
					alreadyFound = true
					break
				}
			}
			if !alreadyFound {
				placeTokens = append(placeTokens, gazetteer.PlaceToken{
					Start: i, End: i + 1, Text: tok,
				})
			}
		}
	}
	// Sort place tokens by position
	for i := 1; i < len(placeTokens); i++ {
		for j := i; j > 0 && placeTokens[j].Start < placeTokens[j-1].Start; j-- {
			placeTokens[j], placeTokens[j-1] = placeTokens[j-1], placeTokens[j]
		}
	}

	// --- Pass 1: Find dates ---
	var dates []parsedDate

	for i := 0; i < n; i++ {
		if consumed[i] {
			continue
		}
		lower := strings.ToLower(tokens[i])
		month, isMonth := months[lower]
		if !isMonth {
			continue
		}
		// Need at least a day number after the month
		if i+1 >= n {
			continue
		}
		day := parseDay(tokens[i+1])
		if day == 0 {
			continue
		}

		// Check for year
		year := 0
		endIdx := i + 2
		if i+2 < n {
			year = parseYear(tokens[i+2])
			if year > 0 {
				endIdx = i + 3
			}
		}

		// Apply year inference if no explicit year
		hasYear := year > 0
		if !hasYear {
			year = inferYear(month, day, today)
		}

		d := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		dates = append(dates, parsedDate{t: d, hasYear: hasYear, startToken: i, endToken: endIdx})

		// Check for same-month range: "March 12-16", "March 12 - 16", or "March 12 to 16"
		rangeEndIdx := endIdx
		rangeConnector := ""
		rangeConnectorIdx := -1

		if rangeEndIdx < n {
			tok := strings.ToLower(tokens[rangeEndIdx])
			if tok == "-" || tok == "to" || tok == "through" || tok == "–" || tok == "—" {
				rangeConnector = tok
				rangeConnectorIdx = rangeEndIdx
				rangeEndIdx++
			}
		}

		if rangeConnector != "" && rangeEndIdx < n {
			endDay := parseDay(tokens[rangeEndIdx])
			if endDay > 0 {
				// Check that the next token is NOT a month (that would be a separate date)
				isStandaloneDay := true
				if _, isMonth := months[strings.ToLower(tokens[rangeEndIdx])]; isMonth {
					isStandaloneDay = false
				}
				if isStandaloneDay {
					endYear := year
					if !hasYear {
						endYear = inferYear(month, endDay, today)
					}
					d2 := time.Date(endYear, month, endDay, 0, 0, 0, 0, time.UTC)
					dates = append(dates, parsedDate{t: d2, hasYear: hasYear, startToken: rangeConnectorIdx, endToken: rangeEndIdx + 1})
					consumed[rangeConnectorIdx] = true
					consumed[rangeEndIdx] = true
					endIdx = rangeEndIdx + 1
				}
			}
		}

		// Mark tokens as consumed
		for j := i; j < endIdx; j++ {
			consumed[j] = true
		}
		i = endIdx - 1
	}

	// --- Pass 1b: Find numeric dates (M/D, MM/DD, M/D/YYYY) ---
	for i := 0; i < n; i++ {
		if consumed[i] {
			continue
		}
		m, d, y, ok := parseNumericDate(tokens[i])
		if !ok {
			continue
		}
		hasYear := y > 0
		if !hasYear {
			y = inferYear(time.Month(m), d, today)
		}
		dt := time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
		dates = append(dates, parsedDate{t: dt, hasYear: hasYear, startToken: i, endToken: i + 1})
		consumed[i] = true

		// Consume preceding "on"
		if i > 0 && strings.ToLower(tokens[i-1]) == "on" && !consumed[i-1] {
			consumed[i-1] = true
		}
	}

	// --- Pass 1c: Find relative dates (tomorrow, Tuesday, next Friday) ---
	if len(dates) == 0 {
		for i := 0; i < n; i++ {
			if consumed[i] {
				continue
			}
			lower := strings.ToLower(tokens[i])

			var resolved time.Time
			resolvedEnd := i + 1
			found := false

			switch lower {
			case "today":
				resolved = today
				found = true
			case "tomorrow":
				resolved = today.AddDate(0, 0, 1)
				found = true
			case "yesterday":
				resolved = today.AddDate(0, 0, -1)
				found = true
			default:
				// Check for day of week
				if dow, isDow := dayOfWeek[lower]; isDow {
					// "next Friday" or "this Thursday"
					hasPrefix := false
					if i > 0 {
						prev := strings.ToLower(tokens[i-1])
						if prev == "next" || prev == "this" || prev == "on" {
							hasPrefix = true
							resolvedEnd = i + 1
							// Consume the prefix
							consumed[i-1] = true
						}
					}
					_ = hasPrefix
					resolved = nextDayOfWeek(today, dow)
					found = true
				}
			}

			if found {
				// Consume preceding "on" if present
				if i > 0 && strings.ToLower(tokens[i-1]) == "on" && !consumed[i-1] {
					consumed[i-1] = true
				}

				dates = append(dates, parsedDate{t: resolved, hasYear: true, startToken: i, endToken: resolvedEnd})
				for j := i; j < resolvedEnd; j++ {
					consumed[j] = true
				}
			}
		}
	}

	// --- Pass 2: Find connectors and assign dates to start/end ---
	var startDate, endDate *time.Time
	startHasYear, endHasYear := false, false

	if len(dates) >= 2 {
		// Check for range connectors between the two dates
		startDate = &dates[0].t
		endDate = &dates[1].t
		startHasYear = dates[0].hasYear
		endHasYear = dates[1].hasYear

		// Consume connectors between the two dates (-, to, through)
		for i := dates[0].endToken; i < dates[1].startToken; i++ {
			lower := strings.ToLower(tokens[i])
			if lower == "-" || lower == "to" || lower == "through" || lower == "–" || lower == "—" {
				consumed[i] = true
			}
		}
	} else if len(dates) == 1 {
		startDate = &dates[0].t
		endDate = &dates[0].t
		startHasYear = dates[0].hasYear
		endHasYear = dates[0].hasYear
	}

	// Consume "on" before a date
	for _, d := range dates {
		if d.startToken > 0 && strings.ToLower(tokens[d.startToken-1]) == "on" && !consumed[d.startToken-1] {
			consumed[d.startToken-1] = true
		}
	}

	// --- Pass 3: Location extraction ---
	location := ""
	locationConf := ConfLow
	routeFrom := ""

	// Strategy:
	// 1. Route detection (2 place tokens with "to") — most specific
	// 2. Keyword extraction ("at"/"in") — gives full phrases like "the Google NYC Office"
	// 3. Single place token from gazetteer — last resort

	// Step 1: Route from place tokens
	if len(placeTokens) >= 2 {
		p1 := placeTokens[0]
		p2 := placeTokens[1]
		isRoute := false
		for i := p1.End; i < p2.Start; i++ {
			if strings.ToLower(tokens[i]) == "to" {
				isRoute = true
				consumed[i] = true
				break
			}
		}
		if isRoute {
			for i := 0; i < p1.Start; i++ {
				if strings.ToLower(tokens[i]) == "from" && !consumed[i] {
					consumed[i] = true
					break
				}
			}
			routeFrom = p1.Text
			location = p1.Text + " → " + p2.Text
			locationConf = ConfHigh
			for i := p1.Start; i < p1.End; i++ { consumed[i] = true }
			for i := p2.Start; i < p2.End; i++ { consumed[i] = true }
		}
	}

	// Step 1b: Route from keyword extraction (fallback)
	if location == "" {
		rf, rt, rc := extractRoute(tokens, consumed, dates)
		if rf != "" && rt != "" {
			routeFrom = rf
			location = rf + " → " + rt
			locationConf = ConfHigh
			for _, idx := range rc {
				consumed[idx] = true
			}
		}
	}

	// Step 2: Keyword "in"/"at" extraction — finds the LAST "in"/"at" for full phrases
	if location == "" {
		bestLocStart := -1
		for i := 0; i < n-1; i++ {
			if consumed[i] {
				continue
			}
			lower := strings.ToLower(tokens[i])
			if lower == "in" || lower == "at" {
				bestLocStart = i
			}
		}
		if bestLocStart >= 0 {
			locParts := []string{}
			consumed[bestLocStart] = true
			for j := bestLocStart + 1; j < n; j++ {
				if consumed[j] {
					break
				}
				jLower := strings.ToLower(tokens[j])
				if jLower == "from" || jLower == "to" || jLower == "on" {
					break
				}
				if _, isMonth := months[jLower]; isMonth {
					break
				}
				locParts = append(locParts, tokens[j])
				consumed[j] = true
			}
			if len(locParts) > 0 {
				location = strings.Join(locParts, " ")
				locationConf = ConfHigh
			}
		}
	}

	// Step 3: Single place token from gazetteer (no keyword needed)
	if location == "" && len(placeTokens) >= 1 {
		p := placeTokens[0]
		location = p.Text
		locationConf = ConfHigh
		for i := p.Start; i < p.End; i++ { consumed[i] = true }
		if p.Start > 0 {
			prev := strings.ToLower(tokens[p.Start-1])
			if (prev == "in" || prev == "at" || prev == "to") && !consumed[p.Start-1] {
				consumed[p.Start-1] = true
			}
		}
	}

	// Consume "from" before first date if not part of a route
	if routeFrom == "" && len(dates) >= 1 {
		for i := 0; i < dates[0].startToken; i++ {
			if strings.ToLower(tokens[i]) == "from" && !consumed[i] {
				consumed[i] = true
				break
			}
		}
	}

	// Consume stray "from" and "to" that are date connectors (not route parts)
	if routeFrom == "" && len(dates) >= 1 {
		for i := 0; i < n; i++ {
			if consumed[i] {
				continue
			}
			lower := strings.ToLower(tokens[i])
			if lower == "from" || lower == "to" {
				// Check if this is adjacent to a date token
				isDateConnector := false
				for _, d := range dates {
					if i == d.startToken-1 || i == d.endToken {
						isDateConnector = true
						break
					}
				}
				if isDateConnector {
					consumed[i] = true
				}
			}
		}
	}

	// --- Pass 4: Title and unparsed ---
	var titleParts []string
	var unparsed []string

	for i := 0; i < n; i++ {
		if consumed[i] {
			continue
		}
		titleParts = append(titleParts, tokens[i])
		// Check if this token looks like it should be meaningful but wasn't parsed
		lower := strings.ToLower(tokens[i])
		if isLikelyUnparsedDateWord(lower) {
			unparsed = append(unparsed, tokens[i])
		}
	}
	title := strings.Join(titleParts, " ")

	// --- Pass 5: Type inference ---
	actType := inferType(title, location, routeFrom != "")
	typeConf := ConfLow
	if actType != TypeStay {
		typeConf = ConfMedium
	}

	// --- Build confidence ---
	if title != "" {
		conf["title"] = ConfHigh
	} else {
		conf["title"] = ConfLow
	}
	conf["type"] = typeConf
	if startDate != nil {
		if startHasYear {
			conf["startDate"] = ConfHigh
		} else {
			conf["startDate"] = ConfMedium
		}
	}
	if endDate != nil {
		if endHasYear {
			conf["endDate"] = ConfHigh
		} else if startDate != nil && endDate.Equal(*startDate) && len(dates) == 1 {
			conf["endDate"] = ConfMedium // defaulted to startDate
		} else {
			conf["endDate"] = ConfMedium
		}
	}
	if location != "" {
		conf["location"] = locationConf
	}

	return ParsedResult{
		Title:      title,
		Type:       actType,
		StartDate:  startDate,
		EndDate:    endDate,
		Location:   location,
		Confidence: conf,
		Unparsed:   unparsed,
	}
}

// extractRoute looks for "from X to Y" where X and Y are not dates.
func extractRoute(tokens []string, consumed []bool, dates []parsedDate) (from, to string, consumedIdxs []int) {
	n := len(tokens)
	for i := 0; i < n; i++ {
		if consumed[i] || strings.ToLower(tokens[i]) != "from" {
			continue
		}
		// Check if the token after "from" is a date (month name)
		if i+1 < n {
			if _, isMonth := months[strings.ToLower(tokens[i+1])]; isMonth {
				continue // date usage, not route
			}
		}
		// Find matching "to"
		fromParts := []string{}
		fromIdxs := []int{i}
		j := i + 1
		for j < n && !consumed[j] && strings.ToLower(tokens[j]) != "to" {
			fromParts = append(fromParts, tokens[j])
			fromIdxs = append(fromIdxs, j)
			j++
		}
		if j < n && strings.ToLower(tokens[j]) == "to" {
			toIdx := j
			toParts := []string{}
			toIdxs := []int{toIdx}
			k := j + 1
			for k < n && !consumed[k] {
				lower := strings.ToLower(tokens[k])
				if lower == "on" || lower == "in" || lower == "at" {
					break
				}
				if _, isMonth := months[lower]; isMonth {
					break
				}
				toParts = append(toParts, tokens[k])
				toIdxs = append(toIdxs, k)
				k++
			}
			if len(fromParts) > 0 && len(toParts) > 0 {
				allIdxs := append(fromIdxs, toIdxs...)
				return strings.Join(fromParts, " "), strings.Join(toParts, " "), allIdxs
			}
		}
	}
	return "", "", nil
}

func inferYear(month time.Month, day int, today time.Time) int {
	candidate := time.Date(today.Year(), month, day, 0, 0, 0, 0, time.UTC)
	if candidate.Before(today) {
		return today.Year() + 1
	}
	return today.Year()
}

func inferType(title, location string, isRoute bool) string {
	if isRoute {
		return TypeTravel
	}
	lower := strings.ToLower(title + " " + location)
	words := strings.Fields(lower)
	for _, w := range words {
		if travelKeywords[w] {
			return TypeTravel
		}
		if conferenceKeywords[w] {
			return TypeConference
		}
		if vacationKeywords[w] {
			return TypeVacation
		}
		if commitmentKeywords[w] {
			return TypeCommitment
		}
		// 3 uppercase letters could be airport code
		if len(w) == 3 && isAllUpper(w) {
			return TypeTravel
		}
	}
	// Also check original case for airport codes
	for _, w := range strings.Fields(title + " " + location) {
		if len(w) == 3 && isAllUpper(w) {
			return TypeTravel
		}
	}
	return TypeStay
}

func isAlpha(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && r != '\'' {
			return false
		}
	}
	return len(s) > 0
}

func isAllUpper(s string) bool {
	for _, r := range s {
		if !unicode.IsUpper(r) {
			return false
		}
	}
	return true
}

var dayOfWeek = map[string]time.Weekday{
	"sunday": time.Sunday, "sun": time.Sunday,
	"monday": time.Monday, "mon": time.Monday,
	"tuesday": time.Tuesday, "tue": time.Tuesday, "tues": time.Tuesday,
	"wednesday": time.Wednesday, "wed": time.Wednesday,
	"thursday": time.Thursday, "thu": time.Thursday, "thurs": time.Thursday,
	"friday": time.Friday, "fri": time.Friday,
	"saturday": time.Saturday, "sat": time.Saturday,
}

// nextDayOfWeek returns the next occurrence of the given weekday after today.
// If today IS that weekday, returns next week's.
func nextDayOfWeek(today time.Time, target time.Weekday) time.Time {
	daysAhead := int(target) - int(today.Weekday())
	if daysAhead <= 0 {
		daysAhead += 7
	}
	return today.AddDate(0, 0, daysAhead)
}

func isLikelyUnparsedDateWord(s string) bool {
	words := map[string]bool{
		"tomorrow": true, "yesterday": true, "today": true,
		"next": true, "last": true, "this": true,
		"monday": true, "tuesday": true, "wednesday": true,
		"thursday": true, "friday": true, "saturday": true, "sunday": true,
	}
	return words[s]
}

func tokenize(text string) []string {
	// Split on whitespace, but keep dashes as separate tokens
	// "Jan 22 - Feb 3" → ["Jan", "22", "-", "Feb", "3"]
	raw := strings.Fields(text)
	var result []string
	for _, t := range raw {
		// Strip trailing commas/periods
		t = strings.TrimRight(t, ",.")
		if t == "" {
			continue
		}
		// Split embedded dashes: "22-Feb" → ["22", "-", "Feb"]
		// But keep hyphenated words together: "Alpha-Omega" stays as one token
		if strings.Contains(t, "-") && len(t) > 1 {
			parts := strings.Split(t, "-")
			// Check if this looks like a hyphenated word (all parts are alpha)
			allAlpha := true
			for _, p := range parts {
				if p == "" || !isAlpha(p) {
					allAlpha = false
					break
				}
			}
			if allAlpha && len(parts) == 2 {
				// Keep as a single hyphenated word
				result = append(result, t)
			} else {
				for i, p := range parts {
					if p != "" {
						result = append(result, p)
					}
					if i < len(parts)-1 {
						result = append(result, "-")
					}
				}
			}
		} else {
			result = append(result, t)
		}
	}
	return result
}

func parseDay(s string) int {
	// Strip ordinal suffixes: "22nd" → "22"
	s = strings.TrimRight(s, "stndrdth")
	if len(s) == 0 || len(s) > 2 {
		return 0
	}
	day := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		day = day*10 + int(c-'0')
	}
	if day < 1 || day > 31 {
		return 0
	}
	return day
}

// parseNumericDate parses "4/12", "04/12", "4/12/2026" into month, day, year.
// Returns (month, day, year, ok). Year is 0 if not specified.
func parseNumericDate(s string) (int, int, int, bool) {
	parts := strings.Split(s, "/")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, 0, 0, false
	}

	m := parseSmallInt(parts[0])
	d := parseSmallInt(parts[1])
	if m < 1 || m > 12 || d < 1 || d > 31 {
		return 0, 0, 0, false
	}

	y := 0
	if len(parts) == 3 {
		y = parseYear(parts[2])
		if y == 0 {
			// Try 2-digit year
			yy := parseSmallInt(parts[2])
			if yy >= 0 && yy <= 99 {
				y = 2000 + yy
			}
		}
	}

	return m, d, y, true
}

func parseSmallInt(s string) int {
	if len(s) == 0 || len(s) > 4 {
		return -1
	}
	val := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return -1
		}
		val = val*10 + int(c-'0')
	}
	return val
}

func parseYear(s string) int {
	if len(s) != 4 {
		return 0
	}
	year := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		year = year*10 + int(c-'0')
	}
	if year < 2000 || year > 2100 {
		return 0
	}
	return year
}
