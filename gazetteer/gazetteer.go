// Package gazetteer provides an embedded city database for location autocomplete
// and geocoding. Data is sourced from GeoNames cities15000.
package gazetteer

import (
	"encoding/csv"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"

	_ "embed"
)

//go:embed cities.csv
var citiesCSV string

//go:embed airports.csv
var airportsCSV string

// City represents a city from the gazetteer.
type City struct {
	Name       string
	ASCIIName  string
	AltNames   []string // pipe-separated in CSV
	Country    string   // ISO 3166-1 alpha-2
	Latitude   float64
	Longitude  float64
	Population int
	Timezone   string // IANA
	IsAirport  bool   // true if this entry came from airport data
	IATA       string // airport IATA code (e.g. "EWR"), empty for cities
}

// nameEntry maps a lowercased name to its city index.
type nameEntry struct {
	lower    string
	cityIdx  int
}

// Gazetteer holds the loaded city data with a sorted name index for prefix search.
type Gazetteer struct {
	cities    []City
	nameIndex []nameEntry // sorted by lower
}

var (
	instance *Gazetteer
	once     sync.Once
	loadErr  error
)

// Get returns the lazily-loaded global gazetteer instance.
func Get() (*Gazetteer, error) {
	once.Do(func() {
		instance, loadErr = load(citiesCSV, airportsCSV)
	})
	return instance, loadErr
}

func load(citiesData, airportsData string) (*Gazetteer, error) {
	var cities []City
	var index []nameEntry

	// Load cities
	r := csv.NewReader(strings.NewReader(citiesData))
	if _, err := r.Read(); err != nil {
		return nil, err
	}

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(record) < 8 {
			continue
		}

		lat, _ := strconv.ParseFloat(record[4], 64)
		lng, _ := strconv.ParseFloat(record[5], 64)
		pop, _ := strconv.Atoi(record[6])

		var altNames []string
		if record[2] != "" {
			altNames = strings.Split(record[2], "|")
		}

		city := City{
			Name:       record[0],
			ASCIIName:  record[1],
			AltNames:   altNames,
			Country:    record[3],
			Latitude:   lat,
			Longitude:  lng,
			Population: pop,
			Timezone:   record[7],
		}

		idx := len(cities)
		cities = append(cities, city)

		addName := func(name string) {
			lower := strings.ToLower(name)
			index = append(index, nameEntry{lower: lower, cityIdx: idx})
		}

		addName(city.Name)
		if strings.ToLower(city.ASCIIName) != strings.ToLower(city.Name) {
			addName(city.ASCIIName)
		}
		for _, alt := range altNames {
			addName(alt)
		}
	}

	// Load airports
	ar := csv.NewReader(strings.NewReader(airportsData))
	if _, err := ar.Read(); err != nil {
		return nil, err
	}

	for {
		record, err := ar.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(record) < 7 {
			continue
		}

		iata := record[0]
		name := record[1]
		airportCity := record[2]
		country := record[3]
		lat, _ := strconv.ParseFloat(record[4], 64)
		lng, _ := strconv.ParseFloat(record[5], 64)
		tz := record[6]

		airport := City{
			Name:      name,
			ASCIIName: name,
			AltNames:  []string{iata, airportCity},
			Country:   country,
			Latitude:  lat,
			Longitude: lng,
			Timezone:  tz,
			IsAirport: true,
			IATA:      iata,
			// Airports get a moderate population to rank below major cities
			// but above small towns
			Population: 50000,
		}

		idx := len(cities)
		cities = append(cities, airport)

		// Index by IATA code (primary), airport name, and city name
		index = append(index, nameEntry{lower: strings.ToLower(iata), cityIdx: idx})
		index = append(index, nameEntry{lower: strings.ToLower(name), cityIdx: idx})
		if strings.ToLower(airportCity) != strings.ToLower(name) {
			index = append(index, nameEntry{lower: strings.ToLower(airportCity) + " airport", cityIdx: idx})
		}
	}

	sort.Slice(index, func(i, j int) bool {
		return index[i].lower < index[j].lower
	})

	return &Gazetteer{cities: cities, nameIndex: index}, nil
}

// Result is a single search result.
type Result struct {
	City  City
	Score float64 // 0-1, higher is better
}

// PrefixSearch finds cities whose name, ASCII name, or alternate names start with the query.
// Results are deduplicated and sorted by population (descending).
func (g *Gazetteer) PrefixSearch(query string, limit int) []Result {
	if query == "" || limit <= 0 {
		return nil
	}

	lower := strings.ToLower(query)

	// Binary search for first entry >= lower
	start := sort.Search(len(g.nameIndex), func(i int) bool {
		return g.nameIndex[i].lower >= lower
	})

	// Scan forward while prefix matches
	seen := map[int]bool{}
	var matched []int

	for i := start; i < len(g.nameIndex); i++ {
		entry := g.nameIndex[i]
		if !strings.HasPrefix(entry.lower, lower) {
			break
		}
		if seen[entry.cityIdx] {
			continue
		}
		seen[entry.cityIdx] = true
		matched = append(matched, entry.cityIdx)
	}

	// Sort by population descending
	sort.Slice(matched, func(i, j int) bool {
		return g.cities[matched[i]].Population > g.cities[matched[j]].Population
	})

	// Build results
	if len(matched) > limit {
		matched = matched[:limit]
	}

	// Find max population for scoring
	maxPop := 1
	if len(matched) > 0 {
		maxPop = g.cities[matched[0]].Population
		if maxPop == 0 {
			maxPop = 1
		}
	}

	results := make([]Result, len(matched))
	for i, idx := range matched {
		c := g.cities[idx]
		score := 0.8 * float64(c.Population) / float64(maxPop)
		// Boost exact matches
		if strings.ToLower(c.Name) == lower || strings.ToLower(c.ASCIIName) == lower {
			score = 1.0
		}
		results[i] = Result{City: c, Score: score}
	}

	return results
}

// ExactSearch finds a city by exact name match (case-insensitive).
func (g *Gazetteer) ExactSearch(name string) *City {
	lower := strings.ToLower(name)

	start := sort.Search(len(g.nameIndex), func(i int) bool {
		return g.nameIndex[i].lower >= lower
	})

	for i := start; i < len(g.nameIndex); i++ {
		entry := g.nameIndex[i]
		if entry.lower != lower {
			break
		}
		c := g.cities[entry.cityIdx]
		return &c
	}
	return nil
}
