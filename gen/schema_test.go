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
	files, err := filesOf(t, gen.Options{Validation: testValidation, Package: pkg, Schema: true})
	require.NoError(t, err)

	return string(files[0].Content)
}

func TestGenerate_Schema(t *testing.T) {
	t.Parallel()

	t.Run("is left out unless asked for", func(t *testing.T) {
		t.Parallel()

		// Act
		pkg := parseSource(t, "//vogue:string Code required\n", rules.MustSet())
		files, err := filesOf(t, gen.Options{Validation: testValidation, Package: pkg})

		// Assert
		require.NoError(t, err)
		assert.NotContains(t, string(files[0].Content), "JSONSchema")
	})

	t.Run("describes a string from its rules", func(t *testing.T) {
		t.Parallel()

		// Act
		code := generateSchema(t, "//vogue:string CountryCode trim upper required len=2 regex=^[A-Z]{2}$\n")

		// Assert
		assert.Contains(t, code, "func (CountryCode) JSONSchema() map[string]any {")
		assert.NotContains(t, code, "vogue/schema", "the schema is a plain map, no package of vogue")
		assert.Regexp(t, `"type":\s+"string",`, code)
		assert.Regexp(t, `"pattern":\s+`+"`"+`\^\[A-Z\]\{2\}\$`+"`"+`,`, code)
		assert.Regexp(t, `minimumLength\s+= 2\n`, code)
		assert.Regexp(t, `"minLength":\s+minimumLength,`, code)
		assert.Regexp(t, `"maxLength":\s+maximumLength,`, code)
	})

	t.Run("describes the bounds of a number and the format of its text", func(t *testing.T) {
		t.Parallel()

		// Act
		code := generateSchema(t, "//vogue:decimal Latitude min=-90 max=90.5\n")

		// Assert
		assert.Regexp(t, `"format":\s+"decimal",`, code)
		assert.Regexp(t, `minimum\s+float64 = -90\n`, code)
		assert.Regexp(t, `maximum\s+float64 = 90.5\n`, code)
		assert.Regexp(t, `"maximum":\s+maximum,`, code)
	})

	t.Run("lists the members of an enum", func(t *testing.T) {
		t.Parallel()

		// Act
		code := generateSchema(t, "//vogue:enum PlaceKind country,city\n")

		// Assert
		assert.Contains(t, code, "members := PlaceKinds{}.All()", "the members are read from the catalogue, not spelled a third time")
		assert.Regexp(t, `"enum":\s+values,`, code)
	})

	t.Run("names the format of an identifier", func(t *testing.T) {
		t.Parallel()

		// Act
		code := generateSchema(t, "//vogue:id PlaceID\n//vogue:id RunID int64\n")

		// Assert
		assert.Regexp(t, `"format":\s+"uuid",`, code)
		assert.Regexp(t, `"exclusiveMinimum":\s+exclusiveMinimum,`, code)
	})
}

func TestGenerate_SchemaDerivation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		directive string
		want      []string
		wantNot   []string
	}{
		{
			name:      "the tightest length wins, whatever order the rules are in",
			directive: "//vogue:string Code min=3 len=2 max=9 required\n",
			want:      []string{`minimumLength\s+= 3\n`, `maximumLength\s+= 2\n`},
		},
		{
			name:      "required raises a minimum length of zero to one",
			directive: "//vogue:string Code min=0 required\n",
			want:      []string{`minimumLength\s+= 1\n`},
		},
		{
			name:      "nonneg and min combine into the higher of the two minimums",
			directive: "//vogue:int Floor min=-5 nonneg\n//vogue:int Covers nonneg min=3\n",
			want:      []string{`minimum\s+float64 = 0\n`, `minimum\s+float64 = 3\n`},
			wantNot:   []string{`minimum\s+float64 = -5\n`},
		},
		{
			name:      "a check a normalizer runs after says nothing about the canonical text",
			directive: "//vogue:string Code len=2 regex=^[a-z]+$ upper\n",
			wantNot:   []string{`"pattern"`, `"minLength"`},
		},
		{
			name:      "the items of an integer list are normalized",
			directive: "//vogue:int Courses oneof=01,+2\n",
			want:      []string{`"enum":\s+\[\]string\{"1", "2"\},`},
		},
		{
			name:      "an RE2-only pattern is left out rather than published in the wrong dialect",
			directive: "//vogue:string Code regex=\\A[a-z]+\\z\n//vogue:string Other regex=(?i)^abc$\n",
			wantNot:   []string{`"pattern"`},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Act
			code := generateSchema(t, tc.directive)

			// Assert
			for _, pattern := range tc.want {
				assert.Regexp(t, pattern, code)
			}
			for _, pattern := range tc.wantNot {
				assert.NotRegexp(t, pattern, code)
			}
		})
	}
}
