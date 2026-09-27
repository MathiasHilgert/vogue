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

	"github.com/MathiasHilgert/vogue/textjson"
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
		zero         Coordinates
		notification validation.Notification
	)

	parsedLatitude, latitudeErr := NewLatitudeFromString(latitude)
	parsedLongitude, longitudeErr := NewLongitudeFromString(longitude)

	// Collect hands back only an error that is not a validation failure,
	// which a generated constructor never returns; it is wrapped all the same
	// rather than assumed away.
	for _, err := range []error{latitudeErr, longitudeErr} {
		unexpected := notification.Collect(err)
		if unexpected != nil {
			return zero, fmt.Errorf("vogue: building coordinates: %w", unexpected)
		}
	}

	if notification.HasErrors() {
		failed := notification

		return zero, &failed
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
// zero Coordinates has no text form, like every generated value object: it
// wraps validation.ErrZeroValue.
func (coordinates Coordinates) MarshalText() ([]byte, error) {
	if coordinates.IsZero() {
		return nil, fmt.Errorf("vogue: cannot marshal the zero Coordinates: %w", validation.ErrZeroValue)
	}

	return []byte(coordinates.String()), nil
}

// MarshalJSON implements json.Marshaler: null for the zero Coordinates, the
// JSON string of its text form otherwise.
func (coordinates Coordinates) MarshalJSON() ([]byte, error) {
	if coordinates.IsZero() {
		return textjson.Null(), nil
	}

	return textjson.Quote([]byte(coordinates.String())), nil
}

// UnmarshalJSON implements json.Unmarshaler: null reads back as the zero
// Coordinates, a JSON string through UnmarshalText.
func (coordinates *Coordinates) UnmarshalJSON(data []byte) error {
	text, isNull, err := textjson.Unquote(data)
	if err != nil {
		return fmt.Errorf("vogue: cannot decode Coordinates: %w", err)
	}

	if isNull {
		var zero Coordinates

		*coordinates = zero

		return nil
	}

	return coordinates.UnmarshalText(text)
}

// UnmarshalText implements encoding.TextUnmarshaler, re-running validation.
func (coordinates *Coordinates) UnmarshalText(data []byte) error {
	latitude, longitude, ok := strings.Cut(string(data), ",")
	if !ok {
		var notification validation.Notification

		notification.Reject("coordinates", "pair", "", string(data), "coordinates must be written as latitude,longitude")

		return &notification
	}

	parsed, err := NewCoordinates(latitude, longitude)
	if err != nil {
		return err
	}

	*coordinates = parsed

	return nil
}
