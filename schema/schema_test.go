package schema_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/MathiasHilgert/vogue/schema"
)

func TestSchema_Accepts(t *testing.T) {
	t.Parallel()

	code := schema.Schema{
		Type:             schema.String,
		Format:           "",
		Pattern:          "^[A-Z]{2}$",
		Enum:             nil,
		MinLength:        schema.Length{Set: true, Value: 2},
		MaxLength:        schema.Length{Set: true, Value: 2},
		Minimum:          schema.Number{Set: false, Value: 0},
		Maximum:          schema.Number{Set: false, Value: 0},
		ExclusiveMinimum: schema.Number{Set: false, Value: 0},
	}
	latitude := schema.Schema{
		Type:             schema.String,
		Format:           schema.FormatDecimal,
		Pattern:          "",
		Enum:             nil,
		MinLength:        schema.Length{Set: false, Value: 0},
		MaxLength:        schema.Length{Set: false, Value: 0},
		Minimum:          schema.Number{Set: true, Value: -90},
		Maximum:          schema.Number{Set: true, Value: 90},
		ExclusiveMinimum: schema.Number{Set: false, Value: 0},
	}
	kind := schema.Schema{
		Type:             schema.String,
		Format:           "",
		Pattern:          "",
		Enum:             []string{"country", "city"},
		MinLength:        schema.Length{Set: false, Value: 0},
		MaxLength:        schema.Length{Set: false, Value: 0},
		Minimum:          schema.Number{Set: false, Value: 0},
		Maximum:          schema.Number{Set: false, Value: 0},
		ExclusiveMinimum: schema.Number{Set: false, Value: 0},
	}

	cases := []struct {
		name   string
		schema schema.Schema
		text   string
		want   bool
	}{
		{name: "a code of the right shape", schema: code, text: "AR", want: true},
		{name: "a code breaking the pattern", schema: code, text: "ar", want: false},
		{name: "a code too long, counted in runes", schema: code, text: "ARG", want: false},
		{name: "a latitude inside its bounds", schema: latitude, text: "-34.6", want: true},
		{name: "a latitude above its maximum", schema: latitude, text: "90.5", want: false},
		{name: "a latitude that is not a number", schema: latitude, text: "north", want: false},
		{name: "a member", schema: kind, text: "city", want: true},
		{name: "not a member", schema: kind, text: "town", want: false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// Act
			got := testCase.schema.Accepts(testCase.text)

			// Assert
			assert.Equal(t, testCase.want, got)
		})
	}
}
