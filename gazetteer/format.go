package gazetteer

import "strings"

// FormatLocation produces a human-friendly location string.
// "Westport", admin1="CT", country="US" → "Westport, Connecticut"
// "Brussels", admin1="", country="BE" → "Brussels, Belgium"
// "London", admin1="", country="GB" → "London" (well-known, no country needed)
func FormatLocation(name, admin1, country string) string {
	parts := []string{name}

	// Add state/province name for US and Canada
	if admin1 != "" {
		if country == "US" {
			if full, ok := usStateNames[strings.ToUpper(admin1)]; ok {
				parts = append(parts, full)
			} else {
				parts = append(parts, admin1)
			}
		} else if country == "CA" {
			if full, ok := caProvinceDisplay[strings.ToUpper(admin1)]; ok {
				parts = append(parts, full)
			} else {
				parts = append(parts, admin1)
			}
		}
	}

	// Add country name (skip for well-known unambiguous cities)
	if country != "" && !isWellKnownCity(strings.ToLower(name)) {
		if full, ok := countryNames[strings.ToUpper(country)]; ok {
			parts = append(parts, full)
		} else {
			parts = append(parts, country)
		}
	}

	return strings.Join(parts, ", ")
}

var usStateNames = map[string]string{
	"AL": "Alabama", "AK": "Alaska", "AZ": "Arizona", "AR": "Arkansas",
	"CA": "California", "CO": "Colorado", "CT": "Connecticut", "DE": "Delaware",
	"FL": "Florida", "GA": "Georgia", "HI": "Hawaii", "ID": "Idaho",
	"IL": "Illinois", "IN": "Indiana", "IA": "Iowa", "KS": "Kansas",
	"KY": "Kentucky", "LA": "Louisiana", "ME": "Maine", "MD": "Maryland",
	"MA": "Massachusetts", "MI": "Michigan", "MN": "Minnesota", "MS": "Mississippi",
	"MO": "Missouri", "MT": "Montana", "NE": "Nebraska", "NV": "Nevada",
	"NH": "New Hampshire", "NJ": "New Jersey", "NM": "New Mexico", "NY": "New York",
	"NC": "North Carolina", "ND": "North Dakota", "OH": "Ohio", "OK": "Oklahoma",
	"OR": "Oregon", "PA": "Pennsylvania", "RI": "Rhode Island", "SC": "South Carolina",
	"SD": "South Dakota", "TN": "Tennessee", "TX": "Texas", "UT": "Utah",
	"VT": "Vermont", "VA": "Virginia", "WA": "Washington", "WV": "West Virginia",
	"WI": "Wisconsin", "WY": "Wyoming", "DC": "District of Columbia",
}

var caProvinceDisplay = map[string]string{
	"01": "Alberta", "02": "British Columbia", "03": "Manitoba",
	"04": "New Brunswick", "05": "Newfoundland", "07": "Nova Scotia",
	"08": "Ontario", "09": "Prince Edward Island", "10": "Quebec",
	"11": "Saskatchewan", "12": "Nunavut", "13": "Northwest Territories",
	"14": "Yukon",
	// Also handle standard abbreviations
	"AB": "Alberta", "BC": "British Columbia", "MB": "Manitoba",
	"NB": "New Brunswick", "NL": "Newfoundland", "NS": "Nova Scotia",
	"ON": "Ontario", "PE": "Prince Edward Island", "QC": "Quebec",
	"SK": "Saskatchewan", "NU": "Nunavut", "NT": "Northwest Territories",
	"YT": "Yukon",
}

var countryNames = map[string]string{
	"US": "United States", "GB": "United Kingdom", "CA": "Canada",
	"FR": "France", "DE": "Germany", "IT": "Italy", "ES": "Spain",
	"NL": "Netherlands", "BE": "Belgium", "CH": "Switzerland",
	"AT": "Austria", "SE": "Sweden", "NO": "Norway", "DK": "Denmark",
	"FI": "Finland", "IE": "Ireland", "PT": "Portugal", "PL": "Poland",
	"CZ": "Czech Republic", "GR": "Greece", "TR": "Turkey",
	"JP": "Japan", "CN": "China", "KR": "South Korea", "IN": "India",
	"AU": "Australia", "NZ": "New Zealand", "BR": "Brazil", "MX": "Mexico",
	"AR": "Argentina", "CL": "Chile", "CO": "Colombia",
	"IL": "Israel", "SG": "Singapore", "TH": "Thailand",
	"RU": "Russia", "UA": "Ukraine",
}

// Well-known cities that don't need country qualification
func isWellKnownCity(name string) bool {
	known := map[string]bool{
		"london": true, "paris": true, "tokyo": true, "berlin": true,
		"rome": true, "amsterdam": true, "brussels": true, "vienna": true,
		"sydney": true, "dublin": true, "singapore": true,
	}
	return known[name]
}
