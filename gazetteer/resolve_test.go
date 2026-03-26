package gazetteer

import (
	"testing"
)

func TestResolveLocation_WestportCT(t *testing.T) {
	g, err := Get()
	if err != nil {
		t.Fatal(err)
	}

	results := g.ResolveLocation("Westport, CT", 5)
	if len(results) == 0 {
		t.Fatal("no results for 'Westport, CT'")
	}
	if results[0].City.Name != "Westport" {
		t.Errorf("got %q, want Westport", results[0].City.Name)
	}
	if results[0].City.Country != "US" {
		t.Errorf("country = %q, want US", results[0].City.Country)
	}
	if results[0].City.Admin1 != "CT" {
		t.Errorf("admin1 = %q, want CT", results[0].City.Admin1)
	}
}

func TestResolveLocation_WestportCTUSA(t *testing.T) {
	g, err := Get()
	if err != nil {
		t.Fatal(err)
	}

	results := g.ResolveLocation("Westport, CT, USA", 5)
	if len(results) == 0 {
		t.Fatal("no results for 'Westport, CT, USA'")
	}
	if results[0].City.Admin1 != "CT" {
		t.Errorf("admin1 = %q, want CT", results[0].City.Admin1)
	}
}

func TestResolveLocation_FullAddress(t *testing.T) {
	g, err := Get()
	if err != nil {
		t.Fatal(err)
	}

	results := g.ResolveLocation("3 Woodland Dr, Westport, CT 06880 USA", 5)
	if len(results) == 0 {
		t.Fatal("no results for full address")
	}
	if results[0].City.Name != "Westport" {
		t.Errorf("got %q, want Westport", results[0].City.Name)
	}
}

func TestResolveLocation_MilanItaly(t *testing.T) {
	g, err := Get()
	if err != nil {
		t.Fatal(err)
	}

	results := g.ResolveLocation("Milan, Italy", 5)
	if len(results) == 0 {
		t.Fatal("no results for 'Milan, Italy'")
	}
	if results[0].City.Country != "IT" {
		t.Errorf("country = %q, want IT", results[0].City.Country)
	}
}

func TestResolveLocation_MilanIT(t *testing.T) {
	g, err := Get()
	if err != nil {
		t.Fatal(err)
	}

	results := g.ResolveLocation("Milan, IT", 5)
	if len(results) == 0 {
		t.Fatal("no results for 'Milan, IT'")
	}
	if results[0].City.Country != "IT" {
		t.Errorf("country = %q, want IT", results[0].City.Country)
	}
}

func TestResolveLocation_LondonPlain(t *testing.T) {
	g, err := Get()
	if err != nil {
		t.Fatal(err)
	}

	results := g.ResolveLocation("London", 5)
	if len(results) == 0 {
		t.Fatal("no results for 'London'")
	}
	// London GB should win by population
	if results[0].City.Country != "GB" {
		t.Errorf("country = %q, want GB", results[0].City.Country)
	}
}

func TestResolveLocation_LondonGB(t *testing.T) {
	g, err := Get()
	if err != nil {
		t.Fatal(err)
	}

	results := g.ResolveLocation("London, GB", 5)
	if len(results) == 0 {
		t.Fatal("no results")
	}
	// All results should be GB
	for _, r := range results {
		if r.City.Country != "GB" {
			t.Errorf("got country %q for %q, want GB", r.City.Country, r.City.Name)
		}
	}
}

func TestResolveLocation_TorontoCA(t *testing.T) {
	g, err := Get()
	if err != nil {
		t.Fatal(err)
	}

	// "Toronto, CA" — CA is ambiguous but Toronto is clearly Canada
	results := g.ResolveLocation("Toronto, Canada", 5)
	if len(results) == 0 {
		t.Fatal("no results")
	}
	if results[0].City.Country != "CA" {
		t.Errorf("country = %q, want CA", results[0].City.Country)
	}
}

func TestResolveLocation_PortlandOR(t *testing.T) {
	g, err := Get()
	if err != nil {
		t.Fatal(err)
	}

	results := g.ResolveLocation("Portland, OR", 5)
	if len(results) == 0 {
		t.Fatal("no results")
	}
	if results[0].City.Admin1 != "OR" {
		t.Errorf("admin1 = %q, want OR", results[0].City.Admin1)
	}
}

func TestResolveLocation_PortlandME(t *testing.T) {
	g, err := Get()
	if err != nil {
		t.Fatal(err)
	}

	results := g.ResolveLocation("Portland, ME", 5)
	if len(results) == 0 {
		t.Fatal("no results")
	}
	if results[0].City.Admin1 != "ME" {
		t.Errorf("admin1 = %q, want ME", results[0].City.Admin1)
	}
}
