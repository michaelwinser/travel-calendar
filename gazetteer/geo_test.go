package gazetteer

import (
	"math"
	"testing"
)

func TestHaversineKm(t *testing.T) {
	tests := []struct {
		name                       string
		lat1, lng1, lat2, lng2     float64
		wantKm                     float64
		tolerance                  float64
	}{
		{
			name: "Brussels to Paris",
			lat1: 50.85, lng1: 4.35,
			lat2: 48.86, lng2: 2.35,
			wantKm: 264, tolerance: 10,
		},
		{
			name: "JFK to EWR (nearby airports)",
			lat1: 40.64, lng1: -73.78,
			lat2: 40.69, lng2: -74.17,
			wantKm: 34, tolerance: 5,
		},
		{
			name: "Manhattan to Brooklyn (same city)",
			lat1: 40.7580, lng1: -73.9855,
			lat2: 40.6782, lng2: -73.9442,
			wantKm: 9, tolerance: 2,
		},
		{
			name: "Same point",
			lat1: 50.85, lng1: 4.35,
			lat2: 50.85, lng2: 4.35,
			wantKm: 0, tolerance: 0.01,
		},
		{
			name: "San Francisco to Tokyo (long distance)",
			lat1: 37.77, lng1: -122.42,
			lat2: 35.68, lng2: 139.69,
			wantKm: 8280, tolerance: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HaversineKm(tt.lat1, tt.lng1, tt.lat2, tt.lng2)
			if math.Abs(got-tt.wantKm) > tt.tolerance {
				t.Errorf("HaversineKm = %.1f km, want %.0f km (±%.0f)", got, tt.wantKm, tt.tolerance)
			}
		})
	}
}

func TestHasCoordinates(t *testing.T) {
	if HasCoordinates(0, 0) {
		t.Error("(0,0) should be treated as no coordinates")
	}
	if !HasCoordinates(50.85, 4.35) {
		t.Error("Brussels should have coordinates")
	}
	if !HasCoordinates(0, 4.35) {
		t.Error("(0, 4.35) should count as having coordinates")
	}
}
