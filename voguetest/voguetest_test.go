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
	var n validation.Notification

	v := strings.TrimSpace(raw)
	if v == "" {
		n.Reject("code", "required", "", v, "code is required")
	}

	if len([]rune(v)) != 2 {
		n.Reject("code", "len", "2", v, "code must be exactly 2 characters long")
	}

	if n.HasErrors() {
		return Code{}, &n
	}

	return Code{v: v}, nil
}

func (c Code) String() string                   { return c.v }
func (c Code) IsZero() bool                     { return c.v == "" }
func (c Code) Equal(other Code) bool            { return c.v == other.v }
func (c Code) MarshalText() ([]byte, error)     { return []byte(c.v), nil }
func (c Code) Value() (driver.Value, error)     { return c.v, nil }
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
			{Name: "the empty string", In: "", Rules: []string{"required", "len"}},
			{Name: "a code too long", In: "ARG", Rules: []string{"len"}},
		},
		Normalized: []voguetest.Normalization[string]{
			{Name: "the blanks around it", In: "  AR ", Out: "AR"},
			{Name: "a rewrite the directive rejects", In: " Tortilla ", Out: "Tortilla"},
		},
		RefusesFloat: false,
	}.Run(t)
}

// Seq is a hand-written database-assigned identifier.
type Seq struct{ v int64 }

func NewSeqFromInt64(raw int64) (Seq, error) {
	if raw <= 0 {
		var n validation.Notification

		n.Reject("seq", "positive", "", strconv.FormatInt(raw, 10), "seq must be a positive identifier")

		return Seq{}, &n
	}

	return Seq{v: raw}, nil
}

func NewSeqFromString(raw string) (Seq, error) {
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		var n validation.Notification

		n.Reject("seq", "int", "", raw, "seq must be a whole number")

		return Seq{}, &n
	}

	return NewSeqFromInt64(v)
}

func (s Seq) String() string               { return strconv.FormatInt(s.v, 10) }
func (s Seq) IsZero() bool                 { return s.v == 0 }
func (s Seq) Equal(other Seq) bool         { return s.v == other.v }
func (s Seq) MarshalText() ([]byte, error) { return []byte(s.String()), nil }
func (s *Seq) UnmarshalText(data []byte) error {
	parsed, err := NewSeqFromString(string(data))
	if err != nil {
		return err
	}

	*s = parsed

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
