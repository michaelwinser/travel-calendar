// Command prepare converts GeoNames cities15000.txt into a compact CSV for embedding.
//
// Usage:
//
//	go run ./gazetteer/prepare /path/to/cities15000.txt > gazetteer/cities.csv
package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <cities15000.txt>\n", os.Args[0])
		os.Exit(1)
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "open: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	// Header
	w.Write([]string{"name", "ascii_name", "alt_names", "country", "lat", "lng", "population", "timezone"})

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), "\t")
		if len(fields) < 19 {
			continue
		}

		name := fields[1]
		asciiName := fields[2]
		altNames := fields[3]
		lat := fields[4]
		lng := fields[5]
		country := fields[8]
		popStr := fields[14]
		timezone := fields[17]

		// Skip entries with no population
		pop, _ := strconv.Atoi(popStr)
		if pop == 0 {
			continue
		}

		// Compress alternate names: keep only ASCII-safe names, limit to 10, deduplicate
		alts := compactAlts(name, asciiName, altNames)

		w.Write([]string{name, asciiName, alts, country, lat, lng, popStr, timezone})
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "scan: %v\n", err)
		os.Exit(1)
	}
}

func compactAlts(name, asciiName, raw string) string {
	if raw == "" {
		return ""
	}

	seen := map[string]bool{
		strings.ToLower(name):      true,
		strings.ToLower(asciiName): true,
	}

	var kept []string
	for _, alt := range strings.Split(raw, ",") {
		alt = strings.TrimSpace(alt)
		if alt == "" {
			continue
		}
		lower := strings.ToLower(alt)
		if seen[lower] {
			continue
		}
		// Skip non-ASCII-friendly names (CJK, Cyrillic, etc.) to save space
		if !isASCIIFriendly(alt) {
			continue
		}
		seen[lower] = true
		kept = append(kept, alt)
		if len(kept) >= 10 {
			break
		}
	}
	return strings.Join(kept, "|")
}

func isASCIIFriendly(s string) bool {
	for _, r := range s {
		if r > 0x024F { // Beyond Latin Extended-B
			return false
		}
	}
	return true
}
