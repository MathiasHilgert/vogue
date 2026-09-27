//go:generate go run github.com/MathiasHilgert/vogue/cmd/vogue -sql=false

package geo

// Latitude is a WGS 84 latitude in decimal degrees.
//vogue:decimal Latitude min=-90 max=90 scale=6 example=-34.603722

// Longitude is a WGS 84 longitude in decimal degrees.
//vogue:decimal Longitude min=-180 max=180 scale=6 example=-58.381592
