package voguetest

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MathiasHilgert/vogue/validation"
)

// Rejection is an input the rules of a directive declare invalid, with every
// rule expected to reject it.
type Rejection[Raw any] struct {
	// Name names the subtest.
	Name string
	// Raw is handed to the constructor.
	Input Raw
	// Rules are the rules that must each report a failure.
	Rules []string
}

// Normalization is a rewrite a normalizer declares: Raw is handed to the
// constructor, which must hold the same value as a value object built from
// Out.
type Normalization[Raw any] struct {
	// Name names the subtest.
	Name string
	// Raw is the raw input.
	Input Raw
	// Out is what the normalizers are declared to turn it into.
	Out Raw
}

// Scalar is the suite of a string, int or decimal value object.
//
// Its sample — the accepted value the round trips are proven with — is chosen
// when the test runs, not when it is generated: it is the first of Examples,
// then of Candidates, that the constructor accepts and that is not the zero
// value. A rule declares its examples for itself alone, so a value one rule
// accepts may be rejected by another rule of the same directive; asking the
// constructor is the only way to know, and the suite skips what needs a
// sample, with a message saying so, when none is accepted.
type Scalar[Object ValueObject[Object], Reference Pointer[Object], Raw any] struct {
	// Field is the name the value object reports its failures under.
	Field string
	// New is the constructor the table drives. For a decimal it is
	// New<Name>FromString, because no Go literal spells a decimal.Decimal.
	New func(Raw) (Object, error)
	// Get reads the value back as New took it. When it is set, an accepted
	// example must be held unchanged, which only holds when no normalizer of
	// the directive rewrites it.
	Get func(Object) Raw
	// FromString is New<Name>FromString, for the kinds whose New takes
	// something other than text, and ParseRule the rule it reports an
	// unreadable representation under. Both are empty otherwise.
	FromString func(string) (Object, error)
	ParseRule  string
	// Examples are the values the directive itself declares valid, with
	// `example=`. Every one of them must be accepted.
	Examples []Raw
	// Candidates are the values the rules of the directive declare valid, each
	// for itself. The first the constructor accepts becomes the sample.
	Candidates []Raw
	// Rejected are the inputs the rules declare invalid.
	Rejected []Rejection[Raw]
	// Normalized are the rewrites the normalizers declare.
	Normalized []Normalization[Raw]
	// RefusesFloat marks a kind whose Scan must refuse a binary float with
	// [validation.ErrLossySource].
	RefusesFloat bool
}

// Run runs the suite.
func (suite Scalar[Object, Reference, Raw]) Run(t *testing.T) {
	t.Helper()

	sample, found := suite.sample()

	t.Run("accepts every example the directive declares", func(t *testing.T) {
		t.Parallel()

		for _, input := range suite.Examples {
			got, err := suite.New(input)
			require.NoError(t, err, "the directive declares %v valid", input)
			assert.False(t, got.IsZero(), "a constructed value object is never the zero value, even for %v", input)

			if suite.Get != nil {
				assert.Equal(t, input, suite.Get(got))
			}
		}
	})

	for _, row := range suite.Rejected {
		t.Run(row.Name, func(t *testing.T) {
			t.Parallel()

			got, err := suite.New(row.Input)

			require.ErrorIs(t, err, validation.ErrInvalid)
			assert.True(t, got.IsZero(), "a rejected input must not produce a usable value object")

			for _, rule := range row.Rules {
				assert.ErrorIs(t, err, validation.FieldError{Field: suite.Field, Rule: rule},
					"the %q rule was expected to reject %v", rule, row.Input)
			}
		})
	}

	for _, row := range suite.Normalized {
		t.Run("normalizes "+row.Name, func(t *testing.T) {
			t.Parallel()
			suite.normalizes(t, row)
		})
	}

	if !found {
		t.Run("round trips", func(t *testing.T) {
			t.Skip("voguetest: no example declared for " + suite.Field +
				" is accepted by every rule of its directive; declare one with example=<value>")
		})

		return
	}

	t.Run("round trips", func(t *testing.T) {
		t.Parallel()
		roundTrips[Object, Reference](t, sample)
		describes(t, sample)
	})

	if suite.FromString != nil {
		t.Run("reads its textual representation", func(t *testing.T) {
			t.Parallel()
			suite.fromString(t, sample)
		})
	}

	if hasScan[Object, Reference]() {
		t.Run("scans", func(t *testing.T) {
			t.Parallel()
			suite.scans(t)
		})
	}

	t.Run("separates the zero value", func(t *testing.T) {
		t.Parallel()
		separatesZero[Object, Reference](t, sample)
	})
}

// sample returns the first declared value the constructor accepts.
func (suite Scalar[Object, Reference, Raw]) sample() (Object, bool) {
	for _, input := range append(append([]Raw(nil), suite.Examples...), suite.Candidates...) {
		got, err := suite.New(input)
		if err == nil {
			return got, true
		}
	}

	var zero Object

	return zero, false
}

// normalizes proves one declared rewrite. When the rest of the directive
// rejects the rewritten value, the rewrite cannot be observed through the
// constructor, and the row is skipped — but only after proving that it is the
// rewritten value, and not the rewrite, that the directive rejects.
func (suite Scalar[Object, Reference, Raw]) normalizes(t *testing.T, row Normalization[Raw]) {
	t.Helper()

	want, wantErr := suite.New(row.Out)
	got, err := suite.New(row.Input)

	if wantErr != nil {
		require.Error(t, err, "%v normalizes to %v, which the directive rejects", row.Input, row.Out)
		t.Skipf("the directive rejects the normalized value %v, so the rewrite is not observable", row.Out)
	}

	require.NoError(t, err)
	assert.True(t, want.Equal(got), "%v was expected to normalize to %v, got %s", row.Input, row.Out, got.String())
}

// fromString proves the textual constructor reads what String writes and
// reports what it cannot read under its own rule.
func (suite Scalar[Object, Reference, Raw]) fromString(t *testing.T, sample Object) {
	t.Helper()

	got, err := suite.FromString(sample.String())
	require.NoError(t, err)
	assert.True(t, sample.Equal(got))

	bad, err := suite.FromString("not-a-number")
	require.ErrorIs(t, err, validation.FieldError{Field: suite.Field, Rule: suite.ParseRule})
	assert.True(t, bad.IsZero())
}

// scans covers the sources Scan accepts and refuses, when there is a Scan.
func (suite Scalar[Object, Reference, Raw]) scans(t *testing.T) {
	t.Helper()

	var got Object

	scan, _ := scanner[Object, Reference](&got)

	require.NoError(t, scan.Scan(nil))
	assert.True(t, got.IsZero(), "a NULL column must produce the zero value")

	err := scan.Scan(struct{}{})
	require.ErrorIs(t, err, validation.ErrUnsupportedSource)

	if len(suite.Rejected) > 0 {
		err = scan.Scan(any(suite.Rejected[0].Input))
		require.ErrorIs(t, err, validation.ErrInvalid, "a stored value the rules reject must be refused")
	}

	if suite.RefusesFloat {
		err = scan.Scan(1.5)
		require.ErrorIs(t, err, validation.ErrLossySource)
	}
}

// separatesZero proves the zero value is told apart from a constructed one at
// every boundary: it is IsZero, has no text form, is null in JSON and NULL in
// SQL, and null and NULL read back as it; a constructed value is none of
// those and survives JSON unchanged.
func separatesZero[Object ValueObject[Object], Reference Pointer[Object]](t *testing.T, constructed Object) {
	t.Helper()

	var zero Object

	assert.True(t, zero.IsZero())
	assert.False(t, constructed.IsZero())
	assert.False(t, zero.Equal(constructed))

	_, err := zero.MarshalText()
	require.ErrorIs(t, err, validation.ErrZeroValue, "the zero value must have no text form")

	separatesZeroInJSON[Object, Reference](t, constructed)
	separatesZeroInSQL[Object, Reference](t, constructed)
}

// separatesZeroInJSON proves the zero value is null in JSON and back.
func separatesZeroInJSON[Object ValueObject[Object], Reference Pointer[Object]](t *testing.T, constructed Object) {
	t.Helper()

	var zero Object

	encoded, err := json.Marshal(zero)
	require.NoError(t, err)
	assert.JSONEq(t, "null", string(encoded), "the zero value must be null in JSON")

	decoded := constructed
	require.NoError(t, json.Unmarshal([]byte("null"), Reference(&decoded)))
	assert.True(t, decoded.IsZero(), "null must read back as the zero value")

	encoded, err = json.Marshal(constructed)
	require.NoError(t, err)

	var roundTripped Object

	require.NoError(t, json.Unmarshal(encoded, Reference(&roundTripped)))
	assert.True(t, constructed.Equal(roundTripped), "%s did not survive JSON", constructed.String())
}

// separatesZeroInSQL proves the zero value is NULL in SQL and back, when the
// value object has the SQL codec.
func separatesZeroInSQL[Object ValueObject[Object], Reference Pointer[Object]](t *testing.T, constructed Object) {
	t.Helper()

	var zero Object

	value, ok := valuer(zero)
	if !ok {
		return
	}

	stored, err := value.Value()
	require.NoError(t, err)
	assert.Nil(t, stored, "the zero value must be stored as NULL")

	constructedValue, _ := valuer(constructed)
	stored, err = constructedValue.Value()
	require.NoError(t, err)
	assert.NotNil(t, stored)

	scanned := constructed
	scan, _ := scanner[Object, Reference](&scanned)
	require.NoError(t, scan.Scan(nil))
	assert.True(t, scanned.IsZero(), "NULL must read back as the zero value")
}

// roundTrips proves a value object survives the text codec and, when it has
// one, the SQL codec.
func roundTrips[Object ValueObject[Object], Reference Pointer[Object]](t *testing.T, want Object) {
	t.Helper()

	text, err := want.MarshalText()
	require.NoError(t, err)

	var fromText Object

	require.NoError(t, Reference(&fromText).UnmarshalText(text))
	assert.True(t, want.Equal(fromText), "%s did not survive the text round trip", want.String())

	value, ok := valuer(want)
	if !ok {
		return
	}

	stored, err := value.Value()
	require.NoError(t, err)

	var fromSQL Object

	scan, ok := scanner[Object, Reference](&fromSQL)
	require.True(t, ok, "a value object with Value must also have Scan")
	require.NoError(t, scan.Scan(stored))
	assert.True(t, want.Equal(fromSQL), "%s did not survive the SQL round trip", want.String())
}
