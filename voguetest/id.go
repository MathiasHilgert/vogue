package voguetest

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MathiasHilgert/vogue/validation"
)

// The UUID literals the identifier suites parse: one of each version vogue
// mints, which proves New<Name>FromString reads any RFC 4122 UUID and not
// only the version the type produces.
const (
	sampleUUIDv4 = "3f333df6-90a4-4fda-8dd3-9485d27cee36"
	sampleUUIDv7 = "018ff1d4-9c2a-7b3e-9f6a-6c1d2e3f4a5b"
	nilUUID      = "00000000-0000-0000-0000-000000000000"
)

// UUID is the suite of a uuid identifier.
type UUID[Object ValueObject[Object], Reference Pointer[Object]] struct {
	// Field is the name the identifier reports its failures under.
	Field string
	// New mints an identifier.
	New func() (Object, error)
	// FromString is New<Name>FromString.
	FromString func(string) (Object, error)
	// Version is the UUID version New mints.
	Version uuid.Version
}

// Run runs the suite.
func (suite UUID[Object, Reference]) Run(t *testing.T) {
	t.Helper()

	t.Run("mints fresh identifiers of the declared version", func(t *testing.T) {
		t.Parallel()

		first, err := suite.New()
		require.NoError(t, err)

		second, err := suite.New()
		require.NoError(t, err)

		assert.False(t, first.IsZero())
		assert.False(t, first.Equal(second), "two mints must not collide")

		versioned, ok := any(first).(interface{ Version() uuid.Version })
		require.True(t, ok)
		assert.Equal(t, suite.Version, versioned.Version())
	})

	t.Run("reads any RFC 4122 UUID", func(t *testing.T) {
		t.Parallel()

		for _, raw := range []string{sampleUUIDv7, sampleUUIDv4, nilUUID} {
			got, err := suite.FromString(raw)
			require.NoError(t, err)
			assert.Equal(t, raw, got.String())
			assert.Equal(t, raw == nilUUID, got.IsZero())
		}
	})

	t.Run("rejects what is not a UUID", func(t *testing.T) {
		t.Parallel()

		for _, raw := range []string{"not-a-uuid", ""} {
			got, err := suite.FromString(raw)
			require.ErrorIs(t, err, validation.FieldError{Field: suite.Field, Rule: "uuid"})
			assert.True(t, got.IsZero())
		}
	})

	t.Run("round trips", func(t *testing.T) {
		t.Parallel()

		minted, err := suite.New()
		require.NoError(t, err)
		roundTrips[Object, Reference](t, minted)
	})

	t.Run("separates an unassigned identifier from a minted one", func(t *testing.T) {
		t.Parallel()

		var zero Object

		minted, err := suite.New()
		require.NoError(t, err)
		assert.True(t, zero.IsZero())
		assert.False(t, zero.Equal(minted))
	})
}

// Int64ID is the suite of a database-assigned integer identifier.
type Int64ID[Object ValueObject[Object], Reference Pointer[Object]] struct {
	// Field is the name the identifier reports its failures under.
	Field string
	// FromInt64 is New<Name>FromInt64.
	FromInt64 func(int64) (Object, error)
	// FromString is New<Name>FromString.
	FromString func(string) (Object, error)
}

// largeID is above 2^53, the largest integer a JavaScript number holds
// exactly, which is why the identifier crosses JSON as a string.
const largeID = 9007199254740993

// Run runs the suite.
func (suite Int64ID[Object, Reference]) Run(t *testing.T) {
	t.Helper()

	t.Run("accepts what a sequence hands out", func(t *testing.T) {
		t.Parallel()

		for _, input := range []int64{1, largeID} {
			got, err := suite.FromInt64(input)
			require.NoError(t, err)
			assert.False(t, got.IsZero())
		}
	})

	t.Run("rejects what no sequence hands out", func(t *testing.T) {
		t.Parallel()

		for _, input := range []int64{0, -1} {
			got, err := suite.FromInt64(input)
			require.ErrorIs(t, err, validation.FieldError{Field: suite.Field, Rule: "positive"})
			assert.True(t, got.IsZero())
		}
	})

	t.Run("reads its textual representation", func(t *testing.T) {
		t.Parallel()

		got, err := suite.FromString("42")
		require.NoError(t, err)
		assert.Equal(t, "42", got.String())

		_, err = suite.FromString("not-a-number")
		require.ErrorIs(t, err, validation.FieldError{Field: suite.Field, Rule: "int"})

		_, err = suite.FromString("0")
		require.ErrorIs(t, err, validation.FieldError{Field: suite.Field, Rule: "positive"})
	})

	t.Run("round trips", func(t *testing.T) {
		t.Parallel()

		want, err := suite.FromInt64(largeID)
		require.NoError(t, err)

		text, err := want.MarshalText()
		require.NoError(t, err)
		assert.Equal(t, "9007199254740993", string(text))
		roundTrips[Object, Reference](t, want)
	})

	if hasScan[Object, Reference]() {
		t.Run("scans", func(t *testing.T) {
			t.Parallel()

			var got Object

			scan, _ := scanner[Object, Reference](&got)

			require.NoError(t, scan.Scan(nil))
			assert.True(t, got.IsZero(), "a NULL column must produce the unassigned identifier")
			require.ErrorIs(t, scan.Scan(int64(0)), validation.ErrInvalid)
			require.ErrorIs(t, scan.Scan(struct{}{}), validation.ErrUnsupportedSource)
		})
	}

	t.Run("separates an unassigned identifier from an assigned one", func(t *testing.T) {
		t.Parallel()

		var zero Object

		assigned, err := suite.FromInt64(1)
		require.NoError(t, err)
		assert.True(t, zero.IsZero())
		assert.Equal(t, "0", zero.String())
		assert.False(t, zero.Equal(assigned))
	})
}
