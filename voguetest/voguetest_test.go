package voguetest_test

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/MathiasHilgert/vogue/schema"
	"github.com/MathiasHilgert/vogue/textjson"
	"github.com/MathiasHilgert/vogue/validation"
	"github.com/MathiasHilgert/vogue/voguetest"
)

// Code is a hand-written value object shaped the way vogue generates one: a
// trimmed, required, two-letter code with the text, JSON and SQL codecs. It
// stands in for generated code so the suite is proven against a value object
// that honours every promise the suite checks.
type Code struct{ value string }

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

	return Code{value: value}, nil
}

func (code Code) String() string        { return code.value }
func (code Code) IsZero() bool          { return code.value == "" }
func (code Code) Equal(other Code) bool { return code.value == other.value }

func (code Code) MarshalText() ([]byte, error) {
	if code.IsZero() {
		return nil, fmt.Errorf("cannot marshal the zero Code: %w", validation.ErrZeroValue)
	}

	return []byte(code.value), nil
}

func (code *Code) UnmarshalText(data []byte) error {
	parsed, err := NewCode(string(data))
	if err != nil {
		return err
	}

	*code = parsed

	return nil
}

func (code Code) MarshalJSON() ([]byte, error) {
	if code.IsZero() {
		return textjson.Null(), nil
	}

	return textjson.Quote([]byte(code.value)), nil
}

func (code *Code) UnmarshalJSON(data []byte) error {
	text, isNull, err := textjson.Unquote(data)
	if err != nil {
		return err
	}

	if isNull {
		*code = Code{}

		return nil
	}

	return code.UnmarshalText(text)
}

// JSONSchema describes Code the way -schema would: two characters, required.
func (Code) JSONSchema() schema.Schema {
	return schema.Schema{
		Type:             schema.String,
		Format:           "",
		Pattern:          "",
		Enum:             nil,
		MinLength:        schema.Length{Set: true, Value: 2},
		MaxLength:        schema.Length{Set: true, Value: 2},
		Minimum:          schema.Number{Set: false, Value: 0},
		Maximum:          schema.Number{Set: false, Value: 0},
		ExclusiveMinimum: schema.Number{Set: false, Value: 0},
	}
}

func (code Code) Value() (driver.Value, error) {
	if code.IsZero() {
		return nil, nil //nolint:nilnil // a nil driver.Value is SQL NULL.
	}

	return code.value, nil
}

func (code *Code) Scan(source any) error {
	switch value := source.(type) {
	case nil:
		*code = Code{}

		return nil
	case string:
		return code.UnmarshalText([]byte(value))
	default:
		return fmt.Errorf("cannot scan %T into Code: %w", source, validation.ErrUnsupportedSource)
	}
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
			{Name: "rejects the empty string", Input: "", Rules: []string{"required", "len"}, Described: true},
			{Name: "rejects a code too long", Input: "ARG", Rules: []string{"len"}, Described: true},
		},
		Normalized: []voguetest.Normalization[string]{
			{Name: "the blanks around it", Input: "  AR ", Out: "AR"},
			{Name: "a rewrite the directive rejects", Input: " Tortilla ", Out: "Tortilla"},
		},
		RefusesFloat: false,
	}.Run(t)
}

// Sequence is a hand-written database-assigned identifier.
type Sequence struct{ value int64 }

func NewSequenceFromInt64(raw int64) (Sequence, error) {
	if raw <= 0 {
		var notification validation.Notification

		notification.Reject("sequence", "positive", "", strconv.FormatInt(raw, 10), "sequence must be a positive identifier")

		return Sequence{}, &notification
	}

	return Sequence{value: raw}, nil
}

func NewSequenceFromString(raw string) (Sequence, error) {
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		var notification validation.Notification

		notification.Reject("sequence", "int", "", raw, "sequence must be a whole number")

		return Sequence{}, &notification
	}

	return NewSequenceFromInt64(value)
}

func (sequence Sequence) String() string            { return strconv.FormatInt(sequence.value, 10) }
func (sequence Sequence) IsZero() bool              { return sequence.value == 0 }
func (sequence Sequence) Equal(other Sequence) bool { return sequence.value == other.value }

func (sequence Sequence) MarshalText() ([]byte, error) {
	if sequence.IsZero() {
		return nil, fmt.Errorf("cannot marshal the zero Sequence: %w", validation.ErrZeroValue)
	}

	return []byte(sequence.String()), nil
}

func (sequence *Sequence) UnmarshalText(data []byte) error {
	parsed, err := NewSequenceFromString(string(data))
	if err != nil {
		return err
	}

	*sequence = parsed

	return nil
}

func (sequence Sequence) MarshalJSON() ([]byte, error) {
	if sequence.IsZero() {
		return textjson.Null(), nil
	}

	return textjson.Quote([]byte(sequence.String())), nil
}

func (sequence *Sequence) UnmarshalJSON(data []byte) error {
	text, isNull, err := textjson.Unquote(data)
	if err != nil {
		return err
	}

	if isNull {
		*sequence = Sequence{}

		return nil
	}

	return sequence.UnmarshalText(text)
}

func TestInt64ID(t *testing.T) {
	t.Parallel()

	voguetest.Int64ID[Sequence, *Sequence]{
		Field:      "sequence",
		FromInt64:  NewSequenceFromInt64,
		FromString: NewSequenceFromString,
	}.Run(t)
}
