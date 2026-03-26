package gazetteer

import (
	"strings"
	"testing"
)

func TestGet(t *testing.T) {
	g, err := Get()
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if g == nil {
		t.Fatal("Get() returned nil")
	}
	if len(g.cities) == 0 {
		t.Fatal("no cities loaded")
	}
	t.Logf("Loaded %d cities, %d name index entries", len(g.cities), len(g.nameIndex))
}

func TestPrefixSearch_Brussels(t *testing.T) {
	g, err := Get()
	if err != nil {
		t.Fatal(err)
	}

	results := g.PrefixSearch("brussel", 10)
	if len(results) == 0 {
		t.Fatal("no results for 'brussel'")
	}

	// Brussels should be the top result for a more specific prefix
	if results[0].City.Name != "Brussels" {
		t.Errorf("top result = %q, want Brussels", results[0].City.Name)
	}
	if results[0].City.Country != "BE" {
		t.Errorf("country = %q, want BE", results[0].City.Country)
	}
	if results[0].City.Timezone != "Europe/Brussels" {
		t.Errorf("timezone = %q, want Europe/Brussels", results[0].City.Timezone)
	}

	// "bru" should include Brussels somewhere in results (via BRU alternate name)
	results2 := g.PrefixSearch("bru", 20)
	found := false
	for _, r := range results2 {
		if r.City.Name == "Brussels" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Brussels not found in 'bru' results")
	}
}

func TestPrefixSearch_BRU_AirportCode(t *testing.T) {
	g, err := Get()
	if err != nil {
		t.Fatal(err)
	}

	// "BRU" is an alternate name for Brussels in the GeoNames data
	results := g.PrefixSearch("bru", 10)
	found := false
	for _, r := range results {
		if r.City.Name == "Brussels" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Brussels not found when searching 'bru' (should match via BRU alternate name)")
	}
}

func TestPrefixSearch_CaseInsensitive(t *testing.T) {
	g, err := Get()
	if err != nil {
		t.Fatal(err)
	}

	r1 := g.PrefixSearch("tokyo", 5)
	r2 := g.PrefixSearch("TOKYO", 5)
	r3 := g.PrefixSearch("Tokyo", 5)

	if len(r1) == 0 || len(r2) == 0 || len(r3) == 0 {
		t.Fatal("one of the searches returned no results")
	}

	if r1[0].City.Name != r2[0].City.Name || r2[0].City.Name != r3[0].City.Name {
		t.Errorf("case mismatch: %q vs %q vs %q", r1[0].City.Name, r2[0].City.Name, r3[0].City.Name)
	}
}

func TestPrefixSearch_EmptyQuery(t *testing.T) {
	g, err := Get()
	if err != nil {
		t.Fatal(err)
	}

	results := g.PrefixSearch("", 10)
	if len(results) != 0 {
		t.Errorf("expected no results for empty query, got %d", len(results))
	}
}

func TestPrefixSearch_PopulationRanking(t *testing.T) {
	g, err := Get()
	if err != nil {
		t.Fatal(err)
	}

	// Use a longer prefix to avoid exact-match boosts interfering
	results := g.PrefixSearch("sant", 10)
	if len(results) < 2 {
		t.Fatal("not enough results for 'sant'")
	}

	// Among prefix-only matches, results should be sorted by population descending
	for i := 1; i < len(results); i++ {
		if results[i].City.Population > results[i-1].City.Population {
			t.Errorf("result %d (pop %d) > result %d (pop %d) — not sorted by population",
				i, results[i].City.Population, i-1, results[i-1].City.Population)
		}
	}
}

func TestExactSearch(t *testing.T) {
	g, err := Get()
	if err != nil {
		t.Fatal(err)
	}

	c := g.ExactSearch("Brussels")
	if c == nil {
		t.Fatal("ExactSearch('Brussels') returned nil")
	}
	if c.Country != "BE" {
		t.Errorf("country = %q, want BE", c.Country)
	}

	c2 := g.ExactSearch("xyznonexistent")
	if c2 != nil {
		t.Errorf("ExactSearch should return nil for nonexistent, got %+v", c2)
	}
}

func TestPrefixSearch_AirportIATA(t *testing.T) {
	g, err := Get()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		query string
		want  string
	}{
		{"ewr", "Newark Liberty International Airport"},
		{"cdg", "Charles de Gaulle International Airport"},
		{"jfk", "John F Kennedy International Airport"},
		{"sfo", "San Francisco International Airport"},
	}

	for _, tt := range tests {
		results := g.PrefixSearch(tt.query, 5)
		if len(results) == 0 {
			t.Errorf("no results for %q", tt.query)
			continue
		}
		found := false
		for _, r := range results {
			if r.City.Name == tt.want {
				found = true
				if !r.City.IsAirport {
					t.Errorf("%q: result is not marked as airport", tt.query)
				}
				if r.City.IATA != strings.ToUpper(tt.query) {
					t.Errorf("%q: IATA = %q, want %q", tt.query, r.City.IATA, strings.ToUpper(tt.query))
				}
				break
			}
		}
		if !found {
			t.Errorf("%q: expected %q in results, got %v", tt.query, tt.want, results[0].City.Name)
		}
	}
}

func TestPrefixSearch_CommonAbbreviations(t *testing.T) {
	g, err := Get()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		query string
		want  string
	}{
		{"nyc", "New York City"},
		{"sf", "San Francisco"},
		{"la", "Los Angeles"},
	}

	for _, tt := range tests {
		results := g.PrefixSearch(tt.query, 5)
		if len(results) == 0 {
			t.Errorf("no results for %q", tt.query)
			continue
		}
		found := false
		for _, r := range results {
			if r.City.Name == tt.want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%q: expected %q in results, got %q", tt.query, tt.want, results[0].City.Name)
		}
	}
}

func TestExactSearch_NYC(t *testing.T) {
	g, err := Get()
	if err != nil {
		t.Fatal(err)
	}

	c := g.ExactSearch("NYC")
	if c == nil {
		t.Fatal("ExactSearch('NYC') returned nil")
	}
	if c.Name != "New York City" {
		t.Errorf("got %q, want New York City", c.Name)
	}
}

func TestPrefixSearch_Coordinates(t *testing.T) {
	g, err := Get()
	if err != nil {
		t.Fatal(err)
	}

	results := g.PrefixSearch("brussels", 1)
	if len(results) == 0 {
		t.Fatal("no results")
	}

	c := results[0].City
	// Brussels is roughly at 50.85N, 4.35E
	if c.Latitude < 50.0 || c.Latitude > 51.0 {
		t.Errorf("latitude = %f, expected ~50.85", c.Latitude)
	}
	if c.Longitude < 4.0 || c.Longitude > 5.0 {
		t.Errorf("longitude = %f, expected ~4.35", c.Longitude)
	}
}
