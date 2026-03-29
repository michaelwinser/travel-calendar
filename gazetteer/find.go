package gazetteer

import (
	"strings"
)

// PlaceToken represents a place found within a text string.
type PlaceToken struct {
	Start   int    // token index (start, inclusive)
	End     int    // token index (end, exclusive)
	Text    string // the matched text
	City    City   // the best gazetteer match
	Score   float64
}

// FindPlaceTokens scans tokens for substrings that match known places.
// It checks single tokens and multi-token sequences (for "New York", "San Francisco", etc.).
// Returns matches sorted by position, preferring longer matches.
func (g *Gazetteer) FindPlaceTokens(tokens []string, consumed []bool) []PlaceToken {
	var found []PlaceToken
	n := len(tokens)

	for i := 0; i < n; i++ {
		if consumed[i] {
			continue
		}

		// Try progressively longer sequences starting at i
		bestMatch := PlaceToken{}
		bestLen := 0

		for length := 1; length <= 3 && i+length <= n; length++ {
			// Check if any token in the range is consumed
			allFree := true
			for j := i; j < i+length; j++ {
				if consumed[j] {
					allFree = false
					break
				}
			}
			if !allFree {
				break
			}

			// Build the candidate string
			parts := make([]string, length)
			for j := 0; j < length; j++ {
				parts[j] = tokens[i+j]
			}
			candidate := strings.Join(parts, " ")

			// Skip very short candidates that are likely not places
			if length == 1 && len(candidate) < 3 {
				continue
			}

			// Skip common words that happen to match cities
			if isCommonWord(strings.ToLower(candidate)) {
				continue
			}

			// Search gazetteer
			results := g.PrefixSearch(candidate, 1)
			if len(results) == 0 {
				continue
			}

			// Require exact match (not just prefix) for short strings
			best := results[0]
			candidateLower := strings.ToLower(candidate)
			nameMatch := strings.ToLower(best.City.Name) == candidateLower ||
				strings.ToLower(best.City.ASCIIName) == candidateLower

			if !nameMatch {
				// Check alternate names
				for _, alt := range best.City.AltNames {
					if strings.ToLower(alt) == candidateLower {
						nameMatch = true
						break
					}
				}
			}

			// For IATA codes (3 uppercase letters), accept exact match
			if !nameMatch && length == 1 && len(candidate) == 3 && isAllUpperASCII(candidate) {
				if best.City.IATA == strings.ToUpper(candidate) {
					nameMatch = true
				}
			}

			if !nameMatch {
				continue
			}

			// Prefer longer matches
			if length > bestLen {
				bestMatch = PlaceToken{
					Start: i,
					End:   i + length,
					Text:  candidate,
					City:  best.City,
					Score: best.Score,
				}
				bestLen = length
			}
		}

		if bestLen > 0 {
			found = append(found, bestMatch)
			i += bestLen - 1 // skip ahead
		}
	}

	return found
}

// isCommonWord returns true for words that shouldn't be treated as places
// even if they match a city name.
func isCommonWord(s string) bool {
	common := map[string]bool{
		// Prepositions and articles
		"the": true, "a": true, "an": true, "on": true, "in": true, "at": true,
		"to": true, "from": true, "for": true, "with": true, "and": true,
		"or": true, "but": true, "not": true, "is": true, "are": true,
		"was": true, "be": true, "been": true, "being": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
		"may": true, "might": true, "can": true, "of": true, "by": true,
		// Activity-related words
		"trip": true, "team": true, "meeting": true, "call": true,
		"flight": true, "drive": true, "train": true,
		// Month names and abbreviations (date parser should handle these)
		"jan": true, "january": true, "feb": true, "february": true,
		"mar": true, "march": true, "apr": true, "april": true,
		// "may" already covered above
		"jun": true, "june": true, "jul": true, "july": true,
		"aug": true, "august": true, "sep": true, "september": true,
		"oct": true, "october": true, "nov": true, "november": true,
		"dec": true, "december": true,
	}
	return common[s]
}

func isAllUpperASCII(s string) bool {
	for _, c := range s {
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	return true
}
