package gazetteer

import (
	"strings"
)

// ResolveLocation parses a comma-separated location string right-to-left,
// extracting country and state/province qualifiers, then searches the
// gazetteer with those filters.
func (g *Gazetteer) ResolveLocation(input string, limit int) []Result {
	if input == "" || limit <= 0 {
		return nil
	}

	query, countryCode, admin1Code := parseLocationParts(input)
	if query == "" {
		return nil
	}

	// Search with the extracted query
	results := g.PrefixSearch(query, limit*3) // over-fetch for filtering

	// Apply filters
	if countryCode != "" || admin1Code != "" {
		filtered := filterResults(results, countryCode, admin1Code)
		if len(filtered) > 0 {
			results = filtered
		}
		// If filtering emptied the results, fall back to unfiltered
	}

	if len(results) > limit {
		results = results[:limit]
	}
	return results
}

// parseLocationParts splits a location string right-to-left on commas,
// identifying country and state/province qualifiers.
func parseLocationParts(input string) (query, countryCode, admin1Code string) {
	parts := strings.Split(input, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	// Remove empty parts
	var nonEmpty []string
	for _, p := range parts {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	if len(nonEmpty) == 0 {
		return "", "", ""
	}

	// Single part — just a city name
	if len(nonEmpty) == 1 {
		return nonEmpty[0], "", ""
	}

	// Walk right-to-left trying to resolve qualifiers
	remaining := nonEmpty

	// Try rightmost part as country
	rightmost := remaining[len(remaining)-1]
	rightmostClean := stripZipCodes(rightmost)

	// Check each word in the rightmost part for country match
	words := strings.Fields(rightmostClean)
	for _, w := range words {
		if cc := resolveCountry(w); cc != "" {
			countryCode = cc
		}
	}

	if countryCode != "" {
		remaining = remaining[:len(remaining)-1]
		// Any non-country words in the rightmost part might be state/admin1
		var stateWords []string
		for _, w := range words {
			if resolveCountry(w) == "" {
				stateWords = append(stateWords, w)
			}
		}
		if len(stateWords) > 0 {
			stateStr := strings.Join(stateWords, " ")
			admin1Code = resolveAdmin1(stateStr, countryCode)
		}
	}

	// If no country found, try rightmost as state (implies country)
	if countryCode == "" {
		if cc, a1 := resolveStateAsCountry(rightmostClean); cc != "" {
			countryCode = cc
			admin1Code = a1
			remaining = remaining[:len(remaining)-1]
		}
	}

	// If we have remaining parts and no admin1 yet, try the new rightmost as state
	if len(remaining) >= 2 && admin1Code == "" && countryCode != "" {
		candidate := stripZipCodes(remaining[len(remaining)-1])
		if a1 := resolveAdmin1(candidate, countryCode); a1 != "" {
			admin1Code = a1
			remaining = remaining[:len(remaining)-1]
		}
	}

	// The remaining parts form the city query.
	// Skip parts that look like street addresses.
	var cityParts []string
	for _, p := range remaining {
		if !isAddressPart(p) {
			cityParts = append(cityParts, p)
		}
	}

	if len(cityParts) == 0 {
		// All parts were addresses — try the last remaining part anyway
		if len(remaining) > 0 {
			query = remaining[len(remaining)-1]
		}
	} else {
		// Use the last non-address part as the city (most specific)
		query = cityParts[len(cityParts)-1]
	}

	return query, countryCode, admin1Code
}

func filterResults(results []Result, countryCode, admin1Code string) []Result {
	var filtered []Result
	for _, r := range results {
		if countryCode != "" && !strings.EqualFold(r.City.Country, countryCode) {
			continue
		}
		if admin1Code != "" && !strings.EqualFold(r.City.Admin1, admin1Code) {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}

// --- Country resolution ---

var countryAliases = map[string]string{
	// Common names
	"usa": "US", "united states": "US", "united states of america": "US", "america": "US",
	"uk": "GB", "united kingdom": "GB", "england": "GB", "great britain": "GB", "britain": "GB",
	"italy": "IT", "italia": "IT",
	"france": "FR",
	"germany": "DE", "deutschland": "DE",
	"spain": "ES", "espana": "ES", "españa": "ES",
	"canada": "CA",
	"japan": "JP",
	"australia": "AU",
	"new zealand": "NZ",
	"ireland": "IE",
	"netherlands": "NL", "holland": "NL",
	"switzerland": "CH",
	"brazil": "BR", "brasil": "BR",
	"mexico": "MX",
	"china": "CN",
	"india": "IN",
	"south korea": "KR", "korea": "KR",
	"belgium": "BE",
	"austria": "AT",
	"portugal": "PT",
	"sweden": "SE",
	"norway": "NO",
	"denmark": "DK",
	"finland": "FI",
	"poland": "PL",
	"czech republic": "CZ", "czechia": "CZ",
	"greece": "GR",
	"turkey": "TR",
	"israel": "IL",
	"singapore": "SG",
	"thailand": "TH",
	"indonesia": "ID",
	"philippines": "PH",
	"argentina": "AR",
	"colombia": "CO",
	"chile": "CL",
	"peru": "PE",
	// ISO codes (2-letter) — only those that don't conflict with state abbreviations
	"us": "US", "gb": "GB", "it": "IT", "fr": "FR", "de": "DE", "es": "ES",
	"jp": "JP", "au": "AU", "nz": "NZ", "ie": "IE", "nl": "NL", "ch": "CH",
	"br": "BR", "mx": "MX", "cn": "CN", "in": "IN", "kr": "KR", "be": "BE",
	"at": "AT", "pt": "PT", "se": "SE", "no": "NO", "dk": "DK", "fi": "FI",
	"pl": "PL", "cz": "CZ", "gr": "GR", "tr": "TR", "il": "IL", "sg": "SG",
	"th": "TH", "id": "ID", "ph": "PH", "ar": "AR", "cl": "CL", "pe": "PE",
}

func resolveCountry(s string) string {
	lower := strings.ToLower(strings.TrimSpace(s))
	return countryAliases[lower]
}

// --- US state resolution ---

var usStatesMap = map[string]string{
	"al": "AL", "alabama": "AL",
	"ak": "AK", "alaska": "AK",
	"az": "AZ", "arizona": "AZ",
	"ar": "AR", "arkansas": "AR",
	"ca": "CA", "california": "CA",
	"co": "CO", "colorado": "CO",
	"ct": "CT", "connecticut": "CT",
	"de": "DE", "delaware": "DE",
	"fl": "FL", "florida": "FL",
	"ga": "GA", "georgia": "GA",
	"hi": "HI", "hawaii": "HI",
	"id": "ID", "idaho": "ID",
	"il": "IL", "illinois": "IL",
	"in": "IN", "indiana": "IN",
	"ia": "IA", "iowa": "IA",
	"ks": "KS", "kansas": "KS",
	"ky": "KY", "kentucky": "KY",
	"la": "LA", "louisiana": "LA",
	"me": "ME", "maine": "ME",
	"md": "MD", "maryland": "MD",
	"ma": "MA", "massachusetts": "MA",
	"mi": "MI", "michigan": "MI",
	"mn": "MN", "minnesota": "MN",
	"ms": "MS", "mississippi": "MS",
	"mo": "MO", "missouri": "MO",
	"mt": "MT", "montana": "MT",
	"ne": "NE", "nebraska": "NE",
	"nv": "NV", "nevada": "NV",
	"nh": "NH", "new hampshire": "NH",
	"nj": "NJ", "new jersey": "NJ",
	"nm": "NM", "new mexico": "NM",
	"ny": "NY", "new york": "NY",
	"nc": "NC", "north carolina": "NC",
	"nd": "ND", "north dakota": "ND",
	"oh": "OH", "ohio": "OH",
	"ok": "OK", "oklahoma": "OK",
	"or": "OR", "oregon": "OR",
	"pa": "PA", "pennsylvania": "PA",
	"ri": "RI", "rhode island": "RI",
	"sc": "SC", "south carolina": "SC",
	"sd": "SD", "south dakota": "SD",
	"tn": "TN", "tennessee": "TN",
	"tx": "TX", "texas": "TX",
	"ut": "UT", "utah": "UT",
	"vt": "VT", "vermont": "VT",
	"va": "VA", "virginia": "VA",
	"wa": "WA", "washington": "WA",
	"wv": "WV", "west virginia": "WV",
	"wi": "WI", "wisconsin": "WI",
	"wy": "WY", "wyoming": "WY",
	"dc": "DC", "district of columbia": "DC",
}

var caProvinceNames = map[string]string{
	"ab": "01", "alberta": "01",
	"bc": "02", "british columbia": "02",
	"mb": "03", "manitoba": "03",
	"nb": "04", "new brunswick": "04",
	"nl": "05", "newfoundland": "05", "newfoundland and labrador": "05",
	"ns": "07", "nova scotia": "07",
	"on": "08", "ontario": "08",
	"pe": "09", "pei": "09", "prince edward island": "09",
	"qc": "10", "quebec": "10", "québec": "10",
	"sk": "11", "saskatchewan": "11",
	"nu": "12", "nunavut": "12",
	"nt": "13", "northwest territories": "13",
	"yt": "14", "yukon": "14",
}

// resolveAdmin1 tries to resolve a string as a state/province code
// given a known country.
func resolveAdmin1(s, countryCode string) string {
	lower := strings.ToLower(strings.TrimSpace(s))
	switch countryCode {
	case "US":
		return usStatesMap[lower]
	case "CA":
		if admin1 := caProvinceNames[lower]; admin1 != "" {
			return admin1
		}
	}
	return ""
}

// resolveStateAsCountry tries to interpret a string as a US state or
// Canadian province when no country was explicitly given.
// Returns (countryCode, admin1Code) or ("", "").
// US states are checked first (slight North American bias).
func resolveStateAsCountry(s string) (countryCode, admin1Code string) {
	lower := strings.ToLower(strings.TrimSpace(s))

	// Handle the CA ambiguity: "CA" alone could be California or Canada.
	// We DON'T resolve it here — we leave it for the caller to
	// disambiguate using the city name. Instead, only resolve
	// unambiguous state abbreviations.
	if lower == "ca" {
		// Ambiguous — don't resolve. Let the search run unfiltered.
		return "", ""
	}

	// Check US states
	if code := usStatesMap[lower]; code != "" {
		return "US", code
	}

	// Check Canadian provinces
	if admin1 := caProvinceNames[lower]; admin1 != "" {
		return "CA", admin1
	}

	return "", ""
}

// --- Address detection ---

var streetSuffixes = map[string]bool{
	"dr": true, "drive": true, "st": true, "street": true,
	"ave": true, "avenue": true, "blvd": true, "boulevard": true,
	"rd": true, "road": true, "ln": true, "lane": true,
	"way": true, "ct": true, "court": true, "pl": true, "place": true,
	"cir": true, "circle": true, "pkwy": true, "parkway": true,
	"hwy": true, "highway": true,
}

func isAddressPart(s string) bool {
	words := strings.Fields(strings.ToLower(s))
	hasNumber := false
	hasSuffix := false
	for _, w := range words {
		if len(w) > 0 && w[0] >= '0' && w[0] <= '9' {
			hasNumber = true
		}
		if streetSuffixes[w] {
			hasSuffix = true
		}
	}
	return hasNumber && hasSuffix
}

func stripZipCodes(s string) string {
	words := strings.Fields(s)
	var out []string
	for _, w := range words {
		isZip := len(w) >= 4 && len(w) <= 10
		if isZip {
			allDigitOrDash := true
			for _, r := range w {
				if (r < '0' || r > '9') && r != '-' {
					allDigitOrDash = false
					break
				}
			}
			if allDigitOrDash {
				continue
			}
		}
		out = append(out, w)
	}
	return strings.Join(out, " ")
}
