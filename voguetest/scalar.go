package voguetest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MathiasHilgert/vogue/validation"
)

// Rejection is an input the rules of a directive declare invalid, with every
// rule expected to reject it.
type Rejection[In any] struct {
	// Name names the subtest.
	Name string
	// In is handed to the constructor.
	In In
	// Rules are the rules that must each report a failure.
	Rules []string
}

// Normalization is a rewrite a normalizer declares: In is handed to the
// constructor, which must hold the same value as a value object built from
// Out.
type Normalization[In any] struct {
	// Name names the subtest.
	Name string
	// In is the raw input.
	In In
	// Out is what the normalizers are declared to turn it into.
	Out In
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
type Scalar[V ValueObject[V], P Pointer[V], In any] struct {
	// Field is the name the value object reports its failures under.
	Field string
	// New is the constructor the table drives. For a decimal it is
	// New<Name>FromString, because no Go literal spells a decimal.Decimal.
	New func(In) (V, error)
	// Get reads the value back as New took it. When it is set, an accepted
	// example must be held unchanged, which only holds when no normalizer of
	// the directive rewrites it.
	Get func(V) In
	// FromString is New<Name>FromString, for the kinds whose New takes
	// something other than text, and ParseRule the rule it reports an
	// unreadable representation under. Both are empty otherwise.
	FromString func(string) (V, error)
	ParseRule  string
	// Examples are the values the directive itself declares valid, with
	// `example=`. Every one of them must be accepted.
	Examples []In
	// Candidates are the values the rules of the directive declare valid, each
	// for itself. The first the constructor accepts becomes the sample.
	Candidates []In
	// Rejected are the inputs the rules declare invalid.
	Rejected []Rejection[In]
	// Normalized are the rewrites the normalizers declare.
	Normalized []Normalization[In]
	// RefusesFloat marks a kind whose Scan must refuse a binary float with
	// [validation.ErrLossySource].
	RefusesFloat bool
}

// Run runs the suite.
func (s Scalar[V, P, In]) Run(t *testing.T) {
	t.Helper()

	sample, found := s.sample()

	t.Run("accepts every example the directive declares", func(t *testing.T) {
		t.Parallel()

		for _, in := range s.Examples {
			got, err := s.New(in)
			require.NoError(t, err, "the directive declares %v valid", in)

			if s.Get != nil {
				assert.Equal(t, in, s.Get(got))
			}
		}
	})

	for _, row := range s.Rejected {
		t.Run("rejects "+row.Name, func(t *testing.T) {
			t.Parallel()

			got, err := s.New(row.In)

			require.ErrorIs(t, err, validation.ErrInvalid)
			assert.True(t, got.IsZero(), "a rejected input must not produce a usable value object")

			for _, rule := range row.Rules {
				assert.ErrorIs(t, err, validation.FieldError{Field: s.Field, Rule: rule},
					"the %q rule was expected to reject %v", rule, row.In)
			}
		})
	}

	for _, row := range s.Normalized {
		t.Run("normalizes "+row.Name, func(t *testing.T) {
			t.Parallel()
			s.normalizes(t, row)
		})
	}

	if !found {
		t.Run("round trips", func(t *testing.T) {
			t.Skip("voguetest: no example declared for " + s.Field +
				" is accepted by every rule of its directive; declare one with example=<value>")
		})

		return
	}

	t.Run("round trips", func(t *testing.T) {
		t.Parallel()
		roundTrips[V, P](t, sample)
	})

	if s.FromString != nil {
		t.Run("reads its textual representation", func(t *testing.T) {
			t.Parallel()
			s.fromString(t, sample)
		})
	}

	if hasScan[V, P]() {
		t.Run("scans", func(t *testing.T) {
			t.Parallel()
			s.scans(t)
		})
	}

	t.Run("separates the zero value", func(t *testing.T) {
		t.Parallel()

		var zero V

		assert.True(t, zero.IsZero())
		assert.False(t, sample.IsZero())
		assert.False(t, zero.Equal(sample))
	})
}

// sample returns the first declared value the constructor accepts that is not
// the zero value.
func (s Scalar[V, P, In]) sample() (V, bool) {
	for _, in := range append(append([]In(nil), s.Examples...), s.Candidates...) {
		got, err := s.New(in)
		if err == nil && !got.IsZero() {
			return got, true
		}
	}

	var zero V

	return zero, false
}

// normalizes proves one declared rewrite. When the rest of the directive
// rejects the rewritten value, the rewrite cannot be observed through the
// constructor, and the row is skipped — but only after proving that it is the
// rewritten value, and not the rewrite, that the directive rejects.
func (s Scalar[V, P, In]) normalizes(t *testing.T, row Normalization[In]) {
	t.Helper()

	want, wantErr := s.New(row.Out)
	got, err := s.New(row.In)

	if wantErr != nil {
		require.Error(t, err, "%v normalizes to %v, which the directive rejects", row.In, row.Out)
		t.Skipf("the directive rejects the normalized value %v, so the rewrite is not observable", row.Out)
	}

	require.NoError(t, err)
	assert.True(t, want.Equal(got), "%v was expected to normalize to %v, got %s", row.In, row.Out, got.String())
}

// fromString proves the textual constructor reads what String writes and
// reports what it cannot read under its own rule.
func (s Scalar[V, P, In]) fromString(t *testing.T, sample V) {
	t.Helper()

	got, err := s.FromString(sample.String())
	require.NoError(t, err)
	assert.True(t, sample.Equal(got))

	bad, err := s.FromString("not-a-number")
	require.ErrorIs(t, err, validation.FieldError{Field: s.Field, Rule: s.ParseRule})
	assert.True(t, bad.IsZero())
}

// scans covers the sources Scan accepts and refuses, when there is a Scan.
func (s Scalar[V, P, In]) scans(t *testing.T) {
	t.Helper()

	var got V

	scan, _ := scanner[V, P](&got)

	require.NoError(t, scan.Scan(nil))
	assert.True(t, got.IsZero(), "a NULL column must produce the zero value")

	err := scan.Scan(struct{}{})
	require.ErrorIs(t, err, validation.ErrUnsupportedSource)

	if len(s.Rejected) > 0 {
		err = scan.Scan(any(s.Rejected[0].In))
		require.ErrorIs(t, err, validation.ErrInvalid, "a stored value the rules reject must be refused")
	}

	if s.RefusesFloat {
		err = scan.Scan(1.5)
		require.ErrorIs(t, err, validation.ErrLossySource)
	}
}

// roundTrips proves a value object survives the text codec and, when it has
// one, the SQL codec.
func roundTrips[V ValueObject[V], P Pointer[V]](t *testing.T, want V) {
	t.Helper()

	text, err := want.MarshalText()
	require.NoError(t, err)

	var fromText V

	require.NoError(t, P(&fromText).UnmarshalText(text))
	assert.True(t, want.Equal(fromText), "%s did not survive the text round trip", want.String())

	value, ok := valuer(want)
	if !ok {
		return
	}

	stored, err := value.Value()
	require.NoError(t, err)

	var fromSQL V

	scan, ok := scanner[V, P](&fromSQL)
	require.True(t, ok, "a value object with Value must also have Scan")
	require.NoError(t, scan.Scan(stored))
	assert.True(t, want.Equal(fromSQL), "%s did not survive the SQL round trip", want.String())
}
