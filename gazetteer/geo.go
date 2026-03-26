package gazetteer

import "math"

const earthRadiusKm = 6371.0

// HaversineKm returns the great-circle distance in kilometers between two points.
func HaversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

// HasCoordinates returns true if the coordinates are non-zero (i.e., set).
func HasCoordinates(lat, lng float64) bool {
	return lat != 0 || lng != 0
}

func toRad(deg float64) float64 {
	return deg * math.Pi / 180
}
