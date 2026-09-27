package voguetest_test

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/MathiasHilgert/vogue/validation"
	"github.com/MathiasHilgert/vogue/voguetest"
)

// Code is a hand-written value object shaped the way vogue generates one: a
// trimmed, required, two-letter code with the text and SQL codecs. It stands in
// for generated code so the suite is proven against a value object that
// honours every promise the suite checks.
type Code struct{ v string }

func NewCode(raw string) (Code, error) {
	var notification validation.Notification

	value := strings.TrimSpace(raw)
	if value == "" {
		notification.Reject("code", "required", "", value, "code is required")
	}

	if len([]rune(value)) != 2 {
		notification.Reject("code", "len", "2", value, "code must be exactly 2 characters long")
	}

	if notification.HasErrors() {
		return Code{}, &notification
	}

	return Code{v: value}, nil
}

func (c Code) String() string               { return c.v }
func (c Code) IsZero() bool                 { return c.v == "" }
func (c Code) Equal(other Code) bool        { return c.v == other.v }
func (c Code) MarshalText() ([]byte, error) { return []byte(c.v), nil }
func (c Code) Value() (driver.Value, error) {
	if c.v == "" {
		return nil, nil //nolint:nilnil // a nil driver.Value is SQL NULL.
	}

	return c.v, nil
}
func (c *Code) UnmarshalText(data []byte) error { return c.set(NewCode(string(data))) }

func (c *Code) Scan(src any) error {
	switch value := src.(type) {
	case nil:
		*c = Code{}

		return nil
	case string:
		return c.UnmarshalText([]byte(value))
	default:
		return fmt.Errorf("cannot scan %T into Code: %w", src, validation.ErrUnsupportedSource)
	}
}

func (c *Code) set(parsed Code, err error) error {
	if err != nil {
		return err
	}

	*c = parsed

	return nil
}

func TestScalar(t *testing.T) {
	t.Parallel()

	voguetest.Scalar[Code, *Code, string]{
		Field:      "code",
		New:        NewCode,
		Get:        nil,
		FromString: nil,
		ParseRule:  "",
		Examples:   []string{"AR"},
		Candidates: []string{"Tortilla", "DE"},
		Rejected: []voguetest.Rejection[string]{
			{Name: "the empty string", Input: "", Rules: []string{"required", "len"}},
			{Name: "a code too long", Input: "ARG", Rules: []string{"len"}},
		},
		Normalized: []voguetest.Normalization[string]{
			{Name: "the blanks around it", Input: "  AR ", Out: "AR"},
			{Name: "a rewrite the directive rejects", Input: " Tortilla ", Out: "Tortilla"},
		},
		RefusesFloat: false,
	}.Run(t)
}

// Seq is a hand-written database-assigned identifier.
type Seq struct{ v int64 }

func NewSeqFromInt64(raw int64) (Seq, error) {
	if raw <= 0 {
		var notification validation.Notification

		notification.Reject("seq", "positive", "", strconv.FormatInt(raw, 10), "seq must be a positive identifier")

		return Seq{}, &notification
	}

	return Seq{v: raw}, nil
}

func NewSeqFromString(raw string) (Seq, error) {
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		var notification validation.Notification

		notification.Reject("seq", "int", "", raw, "seq must be a whole number")

		return Seq{}, &notification
	}

	return NewSeqFromInt64(value)
}

func (suite Seq) String() string               { return strconv.FormatInt(suite.v, 10) }
func (suite Seq) IsZero() bool                 { return suite.v == 0 }
func (suite Seq) Equal(other Seq) bool         { return suite.v == other.v }
func (suite Seq) MarshalText() ([]byte, error) { return []byte(suite.String()), nil }
func (suite *Seq) UnmarshalText(data []byte) error {
	parsed, err := NewSeqFromString(string(data))
	if err != nil {
		return err
	}

	*suite = parsed

	return nil
}

func TestInt64ID(t *testing.T) {
	t.Parallel()

	voguetest.Int64ID[Seq, *Seq]{
		Field:      "seq",
		FromInt64:  NewSeqFromInt64,
		FromString: NewSeqFromString,
	}.Run(t)
}
