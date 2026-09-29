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
//     by joining the error of each part with errors.Join;
//   - a rule about the whole, rather than a part, is recorded on the failure
//     type of the project, the one generated code records its failures on.
//
// Coordinates below is the whole pattern, written to the same strict lint
// profile as the generated code next to it.
package geo

import (
	"errors"
	"fmt"
	"strings"

	"github.com/MathiasHilgert/vogue/examples/validation"
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
	parsedLatitude, latitudeErr := NewLatitudeFromString(latitude)
	parsedLongitude, longitudeErr := NewLongitudeFromString(longitude)

	err := errors.Join(latitudeErr, longitudeErr)
	if err != nil {
		return Coordinates{}, fmt.Errorf("invalid coordinates: %w", err)
	}

	return Coordinates{latitude: parsedLatitude, longitude: parsedLongitude}, nil
}

// Latitude returns the latitude of the point.
func (coordinates Coordinates) Latitude() Latitude { return coordinates.latitude }

// Longitude returns the longitude of the point.
func (coordinates Coordinates) Longitude() Longitude { return coordinates.longitude }

// IsZero reports whether the receiver was never constructed.
func (coordinates Coordinates) IsZero() bool {
	return coordinates.latitude.IsZero() && coordinates.longitude.IsZero()
}

// Equal reports whether both points are the same place.
func (coordinates Coordinates) Equal(other Coordinates) bool {
	return coordinates.latitude.Equal(other.latitude) && coordinates.longitude.Equal(other.longitude)
}

// String returns the point as "latitude,longitude".
func (coordinates Coordinates) String() string {
	return coordinates.latitude.String() + "," + coordinates.longitude.String()
}

// MarshalText implements encoding.TextMarshaler as "latitude,longitude". The
// zero Coordinates has no text form, like every generated value object.
func (coordinates Coordinates) MarshalText() ([]byte, error) {
	if coordinates.IsZero() {
		return nil, fmt.Errorf("cannot marshal the zero Coordinates: %w", errors.ErrUnsupported)
	}

	return []byte(coordinates.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler, re-running validation.
func (coordinates *Coordinates) UnmarshalText(data []byte) error {
	latitude, longitude, ok := strings.Cut(string(data), ",")
	if !ok {
		var failures validation.Validation
		failures.Add("coordinates", "pair", "must be written as latitude,longitude")

		return fmt.Errorf("invalid coordinates: %w", failures.Err())
	}

	parsed, err := NewCoordinates(latitude, longitude)
	if err != nil {
		return err
	}

	*coordinates = parsed

	return nil
}
