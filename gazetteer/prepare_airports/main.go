// Command prepare_airports converts OpenFlights airports.dat into a CSV for embedding.
//
// Usage:
//
//	go run ./gazetteer/prepare_airports /path/to/airports.dat > gazetteer/airports.csv
package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <airports.dat>\n", os.Args[0])
		os.Exit(1)
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "open: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.LazyQuotes = true

	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	w.Write([]string{"iata", "name", "city", "country", "lat", "lng", "timezone"})

	for {
		record, err := r.Read()
		if err != nil {
			break
		}
		if len(record) < 13 {
			continue
		}

		iata := strings.TrimSpace(record[4])
		if iata == "" || iata == "\\N" || len(iata) != 3 {
			continue
		}

		name := record[1]
		city := record[2]
		country := record[3]
		lat := record[6]
		lng := record[7]
		timezone := record[11]

		if timezone == "\\N" {
			timezone = ""
		}

		w.Write([]string{iata, name, city, country, lat, lng, timezone})
	}
}
