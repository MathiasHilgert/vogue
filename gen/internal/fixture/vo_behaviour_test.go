package fixture_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/MathiasHilgert/vogue"
	"github.com/MathiasHilgert/vogue/gen/internal/fixture"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
				require.ErrorIs(t, err, vogue.FieldError{Field: "title", Rule: tc.wantRule},
					"want rule %q, got %v", tc.wantRule, err)
				assert.True(t, got.IsZero())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got.String())
			assert.False(t, got.IsZero())
		})
	}
}

func TestNewTitle_CollectsEveryFailure(t *testing.T) {
	t.Run("reports the required and the bound failure together", func(t *testing.T) {
		// Arrange
		raw := "   "

		// Act
		_, err := fixture.NewTitle(raw)

		// Assert
		require.Error(t, err)
		var n *vogue.Notification
		require.ErrorAs(t, err, &n)
		assert.Equal(t, 1, n.Len(), "required runs before trim, so only min fails here: %v", err)

		// Act: an empty value fails both the required rule and the lower bound.
		_, err = fixture.NewTitle("")

		// Assert
		require.Error(t, err)
		require.ErrorAs(t, err, &n)
		assert.Equal(t, 2, n.Len())
		assert.Equal(t, []string{"title.required", "title.min"}, codes(n))
	})
}

// codes returns the machine-readable code of every collected failure, in order.
func codes(n *vogue.Notification) []string {
	out := make([]string, 0, n.Len())
	for _, e := range n.Errors() {
		out = append(out, e.Code())
	}
	return out
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
		assert.ErrorIs(t, err, vogue.FieldError{Rule: "nodigits"})
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
		assert.ErrorIs(t, err, vogue.FieldError{Field: "covers", Rule: "min"})
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
		assert.ErrorIs(t, err, vogue.FieldError{Field: "covers", Rule: "int"})
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
		require.ErrorIs(t, err, vogue.FieldError{Field: "tabStatus", Rule: "oneof"})
		assert.Contains(t, err.Error(), "open, in_progress, closed")
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
		assert.ErrorIs(t, err, vogue.FieldError{Field: "tabID", Rule: "uuid"})
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
		assert.ErrorIs(t, err, vogue.FieldError{Field: "invoiceNumber", Rule: "positive"})
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
