package gen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHoistRepeatedStrings(t *testing.T) {
	t.Parallel()

	t.Run("names a string used three times and refers to it by name", func(t *testing.T) {
		t.Parallel()

		// Arrange
		src := "package x\n\nimport \"testing\"\n\nvar _ = []string{\"Tortilla\", \"Tortilla\", \"Tortilla\", \"EUR\", \"EUR\"}\n\nfunc TestX(t *testing.T) {}\n"

		// Act
		got, err := hoistRepeatedStrings([]byte(src))

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "package x\n\nimport \"testing\"\n\n"+
			"// The strings below are shared by several rows of the tables in this file.\n"+
			"const (\n\texampleTortilla = \"Tortilla\"\n)\n\n"+
			"var _ = []string{exampleTortilla, exampleTortilla, exampleTortilla, \"EUR\", \"EUR\"}\n\n"+
			"func TestX(t *testing.T) {}\n", string(got))
	})

	t.Run("leaves short strings and the import paths alone", func(t *testing.T) {
		t.Parallel()

		// Arrange
		src := "package x\n\nimport \"testing\"\n\nvar _ = []string{\"ab\", \"ab\", \"ab\"}\n\nfunc TestX(t *testing.T) {}\n"

		// Act
		got, err := hoistRepeatedStrings([]byte(src))

		// Assert
		require.NoError(t, err)
		assert.Equal(t, src, string(got))
	})

	t.Run("numbers a string that spells no identifier and keeps names unique", func(t *testing.T) {
		t.Parallel()

		// Arrange
		src := "package x\n\nvar _ = []string{\"   \", \"   \", \"   \", \"a-b\", \"a-b\", \"a-b\", \"a_b\", \"a_b\", \"a_b\"}\n"

		// Act
		got, err := hoistRepeatedStrings([]byte(src))

		// Assert
		require.NoError(t, err)
		assert.Regexp(t, `example1\s+= "   "`, string(got))
		assert.Regexp(t, `exampleAB\s+= "a-b"`, string(got))
		assert.Regexp(t, `exampleAB2\s+= "a_b"`, string(got))
	})
}
