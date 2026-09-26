// Package geo is the worked example of a composite value object: a value made
// of several generated ones, written by hand.
//
// vogue generates single-field value objects. A value with more than one field
// — a point, an amount with its currency, a range — is composed from them by
// hand, and stays a value object by following the same three rules the
// generated code follows:
//
//   - its fields are unexported, so only its constructor builds a valid one;
//   - the constructor validates every part and reports every failure at once,
//     by folding each part's error into one [validation.Notification] with
//     Collect;
//   - a rule about the whole, rather than a part, is recorded on the same
//     notification with Reject.
//
// Coordinates below is the whole pattern, written to the same strict lint
// profile as the generated code next to it.
package geo

import (
	"fmt"
	"strings"

	"github.com/MathiasHilgert/vogue/validation"
)

// Coordinates is a WGS 84 point: a latitude and a longitude, each validated by
// its own generated value object.
type Coordinates struct {
	latitude  Latitude
	longitude Longitude
}

// NewCoordinates validates both parts and returns the point they describe. The
// error, when there is one, names every part that failed.
func NewCoordinates(latitude, longitude string) (Coordinates, error) {
	var (
		zero Coordinates
		n    validation.Notification
	)

	lat, latErr := NewLatitudeFromString(latitude)
	lon, lonErr := NewLongitudeFromString(longitude)

	// Collect hands back only an error that is not a validation failure,
	// which a generated constructor never returns; it is wrapped all the same
	// rather than assumed away.
	for _, err := range []error{latErr, lonErr} {
		unexpected := n.Collect(err)
		if unexpected != nil {
			return zero, fmt.Errorf("vogue: building coordinates: %w", unexpected)
		}
	}

	if n.HasErrors() {
		failed := n

		return zero, &failed
	}

	return Coordinates{latitude: lat, longitude: lon}, nil
}

// Latitude returns the latitude of the point.
func (c Coordinates) Latitude() Latitude { return c.latitude }

// Longitude returns the longitude of the point.
func (c Coordinates) Longitude() Longitude { return c.longitude }

// IsZero reports whether the receiver was never constructed.
func (c Coordinates) IsZero() bool { return c.latitude.IsZero() && c.longitude.IsZero() }

// Equal reports whether both points are the same place.
func (c Coordinates) Equal(other Coordinates) bool {
	return c.latitude.Equal(other.latitude) && c.longitude.Equal(other.longitude)
}

// String returns the point as "latitude,longitude".
func (c Coordinates) String() string { return c.latitude.String() + "," + c.longitude.String() }

// MarshalText implements encoding.TextMarshaler as "latitude,longitude".
func (c Coordinates) MarshalText() ([]byte, error) { return []byte(c.String()), nil }

// UnmarshalText implements encoding.TextUnmarshaler, re-running validation.
func (c *Coordinates) UnmarshalText(data []byte) error {
	latitude, longitude, ok := strings.Cut(string(data), ",")
	if !ok {
		var n validation.Notification

		n.Reject("coordinates", "pair", "", string(data), "coordinates must be written as latitude,longitude")

		return &n
	}

	parsed, err := NewCoordinates(latitude, longitude)
	if err != nil {
		return err
	}

	*c = parsed

	return nil
}
