package gen_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MathiasHilgert/vogue/gen"
	"github.com/MathiasHilgert/vogue/rules"
)

// generateSchema renders one directive with the JSONSchema method on.
func generateSchema(t *testing.T, directive string) string {
	t.Helper()

	pkg := parseSource(t, directive, rules.MustSet())
	files, err := filesOf(t, gen.Options{Package: pkg, Schema: true})
	require.NoError(t, err)

	return string(files[0].Content)
}

func TestGenerate_Schema(t *testing.T) {
	t.Parallel()

	t.Run("is left out unless asked for", func(t *testing.T) {
		t.Parallel()

		// Act
		pkg := parseSource(t, "//vogue:string Code required\n", rules.MustSet())
		files, err := filesOf(t, gen.Options{Package: pkg})

		// Assert
		require.NoError(t, err)
		assert.NotContains(t, string(files[0].Content), "JSONSchema")
	})

	t.Run("describes a string from its rules", func(t *testing.T) {
		t.Parallel()

		// Act
		code := generateSchema(t, "//vogue:string CountryCode trim upper required len=2 regex=^[A-Z]{2}$\n")

		// Assert
		assert.Contains(t, code, "func (CountryCode) JSONSchema() schema.Schema {")
		assert.Contains(t, code, `"github.com/MathiasHilgert/vogue/schema"`)
		assert.Regexp(t, `Pattern:\s+"\^\[A-Z\]\{2\}\$",`, code)
		assert.Regexp(t, `minLength\s+= 2\n`, code)
		assert.Regexp(t, `MinLength:\s+schema.Length\{Set: true, Value: minLength\},`, code)
		assert.Regexp(t, `MaxLength:\s+schema.Length\{Set: true, Value: maxLength\},`, code)
	})

	t.Run("describes the bounds of a number and the format of its text", func(t *testing.T) {
		t.Parallel()

		// Act
		code := generateSchema(t, "//vogue:decimal Latitude min=-90 max=90.5\n")

		// Assert
		assert.Regexp(t, `Format:\s+schema.FormatDecimal,`, code)
		assert.Regexp(t, `minimum\s+= -90\n`, code)
		assert.Regexp(t, `maximum\s+= 90.5\n`, code)
		assert.Regexp(t, `Maximum:\s+schema.Number\{Set: true, Value: maximum\},`, code)
	})

	t.Run("lists the members of an enum", func(t *testing.T) {
		t.Parallel()

		// Act
		code := generateSchema(t, "//vogue:enum PlaceKind country,city\n")

		// Assert
		assert.Regexp(t, `Enum:\s+\[\]string\{"country", "city"\},`, code)
	})

	t.Run("names the format of an identifier", func(t *testing.T) {
		t.Parallel()

		// Act
		code := generateSchema(t, "//vogue:id PlaceID\n//vogue:id RunID int64\n")

		// Assert
		assert.Regexp(t, `Format:\s+schema.FormatUUID,`, code)
		assert.Regexp(t, `ExclusiveMinimum:\s+schema.Number\{Set: true, Value: exclusiveMinimum\},`, code)
	})
}
