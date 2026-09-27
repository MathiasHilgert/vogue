package textjson_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MathiasHilgert/vogue/textjson"
)

func TestQuote(t *testing.T) {
	t.Parallel()

	cases := []string{
		"", "AR", "Tortilla de patatas", `a "quoted" \ backslash`, "tab\tnew\nline\r",
		"\x00\x1f control", "añó 日本 😀", "line separator ", "invalid \xff utf-8", "<&>",
	}

	for _, text := range cases {
		t.Run(text, func(t *testing.T) {
			t.Parallel()

			// Act
			quoted := textjson.Quote([]byte(text))

			// Assert
			var decoded string
			require.NoError(t, json.Unmarshal(quoted, &decoded), "%s is not a JSON string", quoted)
			want, err := json.Marshal(text)
			require.NoError(t, err)
			var reference string
			require.NoError(t, json.Unmarshal(want, &reference))
			assert.Equal(t, reference, decoded)
		})
	}
}

func TestUnquote(t *testing.T) {
	t.Parallel()

	t.Run("reads what encoding/json writes", func(t *testing.T) {
		t.Parallel()

		for _, text := range []string{"", "AR", `"q" \ /`, "\t\n ", "añó 😀", "\x01"} {
			// Arrange
			encoded, err := json.Marshal(text)
			require.NoError(t, err)

			// Act
			got, isNull, err := textjson.Unquote(encoded)

			// Assert
			require.NoError(t, err)
			assert.False(t, isNull)
			assert.Equal(t, text, string(got))
		}
	})

	t.Run("reads escapes encoding/json accepts", func(t *testing.T) {
		t.Parallel()

		// Act
		got, _, err := textjson.Unquote([]byte(`"\/\b\fé😀"`))

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "/\b\fé😀", string(got))
	})

	t.Run("reports null", func(t *testing.T) {
		t.Parallel()

		// Act
		got, isNull, err := textjson.Unquote([]byte("null"))

		// Assert
		require.NoError(t, err)
		assert.True(t, isNull)
		assert.Nil(t, got)
	})

	t.Run("refuses what is not a JSON string", func(t *testing.T) {
		t.Parallel()

		for _, data := range []string{"42", `"unterminated`, `"bad \x escape"`, `"\u12"`, "", "true", `"raw` + "\n" + `"`} {
			// Act
			_, _, err := textjson.Unquote([]byte(data))

			// Assert
			require.ErrorIs(t, err, textjson.ErrNotString, "%q", data)
		}
	})
}

func TestNull(t *testing.T) {
	t.Parallel()

	// Act & Assert
	assert.Equal(t, "null", string(textjson.Null()))
}
