package fixture_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/MathiasHilgert/vogue/gen/internal/fixture"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rejectedBy reports whether err says that the rule rejected the field, the
// way the generated tests ask it.
func rejectedBy(err error, field, rule string) bool {
	var failed interface{ Has(field, rule string) bool }

	return errors.As(err, &failed) && failed.Has(field, rule)
}

func TestNewTitle(t *testing.T) {
	// Arrange
	cases := []struct {
		name     string
		in       string
		want     string
		wantRule string
	}{
		{name: "accepts a plain title", in: "House Red", want: "house red"},
		{name: "trims and folds before checking", in: "  HOUSE RED  ", want: "house red"},
		{name: "rejects the empty value", in: "", wantRule: "required"},
		{name: "rejects a value above the bound", in: strings.Repeat("a", 121), wantRule: "max"},
		{name: "rejects digits", in: "table 4", wantRule: "nodigits"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			got, err := fixture.NewTitle(tc.in)

			// Assert
			if tc.wantRule != "" {
				require.Error(t, err)
				assert.True(t, rejectedBy(err, "title", tc.wantRule), "want rule %q, got %v", tc.wantRule, err)
				assert.True(t, got.IsZero())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got.String())
			assert.False(t, got.IsZero())
		})
	}
}

func TestNewTitle_Failures(t *testing.T) {
	t.Run("an empty value reports required alone, which stops the rules after it", func(t *testing.T) {
		// Act
		_, err := fixture.NewTitle("")

		// Assert
		require.Error(t, err)
		assert.Equal(t, "invalid title: is required (required)", err.Error())
		assert.False(t, rejectedBy(err, "title", "min"), "min would report the same empty value")
	})

	t.Run("collects every failure of a value that passes required", func(t *testing.T) {
		// Arrange
		raw := strings.Repeat("a", 120) + "1"

		// Act
		_, err := fixture.NewTitle(raw)

		// Assert
		require.Error(t, err)
		assert.True(t, rejectedBy(err, "title", "max"), "%v", err)
		assert.True(t, rejectedBy(err, "title", "nodigits"), "%v", err)
		assert.Equal(t, "invalid title: must be at most 120 (max); invalid title: must not contain digits (nodigits)",
			err.Error(), "the message never carries the value")
	})

	t.Run("a blank is checked after trim, so only min fails", func(t *testing.T) {
		// Act
		_, err := fixture.NewTitle("   ")

		// Assert
		require.Error(t, err)
		assert.True(t, rejectedBy(err, "title", "min"))
		assert.False(t, rejectedBy(err, "title", "required"), "required runs before trim, on the padded value")
	})
}

// TestNewTitle_ValidInputAllocatesNothing proves the happy path allocates
// nothing: the failure type only builds its error when there is one. It is
// not parallel: AllocsPerRun refuses to run beside other tests.
func TestNewTitle_ValidInputAllocatesNothing(t *testing.T) {
	// Act
	allocations := testing.AllocsPerRun(100, func() { _, _ = fixture.NewTitle("house red") })

	// Assert
	assert.Zero(t, allocations)
}

func TestTitle_JSON(t *testing.T) {
	t.Run("round-trips through a struct field", func(t *testing.T) {
		// Arrange
		type tab struct {
			Title  fixture.Title  `json:"title"`
			Covers fixture.Covers `json:"covers"`
		}
		title, err := fixture.NewTitle("House Red")
		require.NoError(t, err)
		covers, err := fixture.NewCovers(4)
		require.NoError(t, err)

		// Act
		raw, err := json.Marshal(tab{Title: title, Covers: covers})
		require.NoError(t, err)
		var got tab
		err = json.Unmarshal(raw, &got)

		// Assert
		require.NoError(t, err)
		assert.JSONEq(t, `{"title":"house red","covers":"4"}`, string(raw))
		assert.True(t, got.Title.Equal(title))
		assert.True(t, got.Covers.Equal(covers))
	})

	t.Run("rejects a payload the constructor would reject", func(t *testing.T) {
		// Arrange
		var got fixture.Title

		// Act
		err := json.Unmarshal([]byte(`"table 4"`), &got)

		// Assert
		require.Error(t, err)
		assert.True(t, rejectedBy(err, "title", "nodigits"), "%v", err)
	})

	t.Run("the zero value cannot be marshaled, so an optional field says omitzero", func(t *testing.T) {
		// Arrange
		type required struct {
			Title fixture.Title `json:"title"`
		}
		type optional struct {
			Title fixture.Title `json:"title,omitzero"`
		}

		// Act
		_, requiredErr := json.Marshal(required{})
		optionalBody, optionalErr := json.Marshal(optional{})

		// Assert
		var marshalerErr *json.MarshalerError
		require.ErrorAs(t, requiredErr, &marshalerErr, "encoding/json reports the zero value as a marshaler error")
		require.NoError(t, optionalErr)
		assert.JSONEq(t, `{}`, string(optionalBody))
	})

	t.Run("null leaves the zero value", func(t *testing.T) {
		// Arrange
		var got fixture.Title

		// Act
		err := json.Unmarshal([]byte(`null`), &got)

		// Assert
		require.NoError(t, err)
		assert.True(t, got.IsZero())
	})
}

func TestTitle_SQL(t *testing.T) {
	// Arrange
	cases := []struct {
		name    string
		src     any
		want    string
		wantErr bool
	}{
		{name: "scans a string", src: "House Red", want: "house red"},
		{name: "scans bytes", src: []byte("House Red"), want: "house red"},
		{name: "scans nil as the zero value", src: nil},
		{name: "rejects a row that no longer validates", src: "table 4", wantErr: true},
		{name: "rejects an unsupported source", src: 42, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			var got fixture.Title
			err := got.Scan(tc.src)

			// Assert
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got.String())

			value, err := got.Value()
			require.NoError(t, err)
			if tc.src == nil {
				assert.Nil(t, value, "a NULL column must be written back as NULL")
				return
			}
			assert.Equal(t, tc.want, value)
		})
	}
}

func TestCovers(t *testing.T) {
	t.Run("rejects a party below the bound", func(t *testing.T) {
		// Act
		_, err := fixture.NewCovers(0)

		// Assert
		require.Error(t, err)
		assert.True(t, rejectedBy(err, "covers", "min"), "%v", err)
	})

	t.Run("round-trips through SQL", func(t *testing.T) {
		// Arrange
		var got fixture.Covers

		// Act
		err := got.Scan(int64(12))

		// Assert
		require.NoError(t, err)
		assert.Equal(t, int64(12), got.Int64())
		value, err := got.Value()
		require.NoError(t, err)
		assert.Equal(t, int64(12), value)
	})

	t.Run("parses a base-ten representation", func(t *testing.T) {
		// Act
		got, err := fixture.NewCoversFromString("8")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "8", got.String())
	})

	t.Run("rejects text that is not a number", func(t *testing.T) {
		// Act
		_, err := fixture.NewCoversFromString("eight")

		// Assert
		require.Error(t, err)
		assert.True(t, rejectedBy(err, "covers", "int"), "%v", err)
	})
}

func TestTabStatus(t *testing.T) {
	t.Run("parses every declared member", func(t *testing.T) {
		// Arrange
		members := fixture.TabStatuses{}.All()
		require.Len(t, members, 3)

		for _, member := range members {
			t.Run(member.String(), func(t *testing.T) {
				// Act
				got, err := fixture.TabStatuses{}.Parse(member.String())

				// Assert
				require.NoError(t, err)
				assert.True(t, got.Equal(member))
				assert.False(t, got.IsZero())
			})
		}
	})

	t.Run("rejects an unknown member", func(t *testing.T) {
		// Act
		got, err := fixture.TabStatuses{}.Parse("refunded")

		// Assert
		require.Error(t, err)
		assert.True(t, got.IsZero())
		assert.True(t, rejectedBy(err, "tabStatus", "oneof"), "%v", err)
		assert.Equal(t, "invalid tabStatus: must be one of: open, in_progress, closed (oneof)", err.Error())
	})

	t.Run("the zero value is no member", func(t *testing.T) {
		// Arrange
		var got fixture.TabStatus

		// Assert
		assert.True(t, got.IsZero())
		for _, member := range (fixture.TabStatuses{}).All() {
			assert.False(t, got.Equal(member))
		}
	})

	t.Run("round-trips through JSON and SQL", func(t *testing.T) {
		// Act
		raw, err := json.Marshal(fixture.TabStatuses{}.InProgress())
		require.NoError(t, err)
		var got fixture.TabStatus
		require.NoError(t, json.Unmarshal(raw, &got))
		var scanned fixture.TabStatus
		err = scanned.Scan([]byte("closed"))

		// Assert
		require.NoError(t, err)
		assert.JSONEq(t, `"in_progress"`, string(raw))
		assert.True(t, got.Equal(fixture.TabStatuses{}.InProgress()))
		assert.True(t, scanned.Equal(fixture.TabStatuses{}.Closed()))
	})
}

func TestTabID(t *testing.T) {
	t.Run("mints a usable identifier", func(t *testing.T) {
		// Act
		got, err := fixture.NewTabID()

		// Assert
		require.NoError(t, err)
		assert.False(t, got.IsZero())
		assert.Equal(t, uuid.Version(7), got.UUID().Version(), "the uuid7 strategy must mint time-ordered identifiers")
	})

	t.Run("parses and compares", func(t *testing.T) {
		// Arrange
		minted, err := fixture.NewTabID()
		require.NoError(t, err)

		// Act
		parsed, err := fixture.NewTabIDFromString(minted.String())

		// Assert
		require.NoError(t, err)
		assert.True(t, parsed.Equal(minted))
	})

	t.Run("rejects text that is not a UUID", func(t *testing.T) {
		// Act
		got, err := fixture.NewTabIDFromString("not-a-uuid")

		// Assert
		require.Error(t, err)
		assert.True(t, got.IsZero())
		assert.True(t, rejectedBy(err, "tabID", "uuid"), "%v", err)
	})

	t.Run("the zero value is the nil UUID", func(t *testing.T) {
		// Arrange
		var got fixture.TabID

		// Assert
		assert.True(t, got.IsZero())
	})
}

func TestInvoiceNumber(t *testing.T) {
	t.Run("accepts a database-assigned number", func(t *testing.T) {
		// Act
		got, err := fixture.NewInvoiceNumberFromInt64(1024)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, int64(1024), got.Int64())
		assert.False(t, got.IsZero())
	})

	t.Run("rejects a non-positive number", func(t *testing.T) {
		// Act
		_, err := fixture.NewInvoiceNumberFromInt64(0)

		// Assert
		require.Error(t, err)
		assert.True(t, rejectedBy(err, "invoiceNumber", "positive"), "%v", err)
	})

	t.Run("crosses JSON as a string so precision survives", func(t *testing.T) {
		// Arrange
		number, err := fixture.NewInvoiceNumberFromInt64(9007199254740993)
		require.NoError(t, err)

		// Act
		raw, err := json.Marshal(number)
		require.NoError(t, err)
		var got fixture.InvoiceNumber
		err = json.Unmarshal(raw, &got)

		// Assert
		require.NoError(t, err)
		assert.JSONEq(t, `"9007199254740993"`, string(raw))
		assert.True(t, got.Equal(number))
	})

	t.Run("round-trips through SQL", func(t *testing.T) {
		// Arrange
		var got fixture.InvoiceNumber

		// Act
		err := got.Scan(int64(7))

		// Assert
		require.NoError(t, err)
		value, err := got.Value()
		require.NoError(t, err)
		assert.Equal(t, int64(7), value)
	})
}

// TestTabStatuses_ParseAllocatesNothing proves the catalogue parses a member
// by comparing against the members themselves, without building a slice. It
// is not parallel: AllocsPerRun refuses to run beside other tests.
func TestTabStatuses_ParseAllocatesNothing(t *testing.T) {
	for _, raw := range []string{"open", "in_progress", "closed"} {
		// Act
		allocations := testing.AllocsPerRun(10, func() { _, _ = fixture.TabStatuses{}.Parse(raw) })

		// Assert
		assert.Zero(t, allocations, "parsing %q allocated", raw)
	}
}
