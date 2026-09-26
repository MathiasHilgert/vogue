package gen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHoistRepeatedStrings(t *testing.T) {
	t.Parallel()

	t.Run("names a string used three times inside every function that uses it", func(t *testing.T) {
		t.Parallel()

		// Arrange
		src := "package x\n\nimport \"testing\"\n\n" +
			"func TestA(t *testing.T) {\n\t_ = []string{\"Tortilla\", \"Tortilla\", \"EUR\"}\n}\n\n" +
			"func TestB(t *testing.T) {\n\t_ = []string{\"Tortilla\", \"EUR\"}\n}\n"

		// Act
		got, err := hoistRepeatedStrings([][]byte{[]byte(src)})

		// Assert
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "package x\n\nimport \"testing\"\n\n"+
			"func TestA(t *testing.T) {\n\tconst exampleTortilla = \"Tortilla\"\n\n\t_ = []string{exampleTortilla, exampleTortilla, \"EUR\"}\n}\n\n"+
			"func TestB(t *testing.T) {\n\tconst exampleTortilla = \"Tortilla\"\n\n\t_ = []string{exampleTortilla, \"EUR\"}\n}\n", string(got[0]))
	})

	t.Run("counts across every file of the package, and declares nothing at package level", func(t *testing.T) {
		t.Parallel()

		// Arrange
		first := "package x\n\nfunc TestA() {\n\t_ = []string{\"Tortilla\", \"Tortilla\"}\n}\n"
		second := "package x\n\nfunc TestB() {\n\t_ = []string{\"Tortilla\"}\n}\n"

		// Act
		got, err := hoistRepeatedStrings([][]byte{[]byte(first), []byte(second)})

		// Assert
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Contains(t, string(got[0]), "\tconst exampleTortilla = \"Tortilla\"\n")
		assert.Contains(t, string(got[1]), "\tconst exampleTortilla = \"Tortilla\"\n")
		assert.NotContains(t, string(got[0]), "\nconst")
		assert.NotContains(t, string(got[1]), "\nconst")
	})

	t.Run("leaves short strings and the import paths alone", func(t *testing.T) {
		t.Parallel()

		// Arrange
		src := "package x\n\nimport \"testing\"\n\nfunc TestX(t *testing.T) {\n\t_ = []string{\"ab\", \"ab\", \"ab\"}\n}\n"

		// Act
		got, err := hoistRepeatedStrings([][]byte{[]byte(src)})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, src, string(got[0]))
	})

	t.Run("numbers a string that spells no identifier and keeps names unique", func(t *testing.T) {
		t.Parallel()

		// Arrange
		src := "package x\n\nfunc TestX() {\n\t_ = []string{\"   \", \"   \", \"   \", \"a-b\", \"a-b\", \"a-b\", \"a_b\", \"a_b\", \"a_b\"}\n}\n"

		// Act
		got, err := hoistRepeatedStrings([][]byte{[]byte(src)})

		// Assert
		require.NoError(t, err)
		assert.Regexp(t, `example1\s+= "   "`, string(got[0]))
		assert.Regexp(t, `exampleAB\s+= "a-b"`, string(got[0]))
		assert.Regexp(t, `exampleAB2\s+= "a_b"`, string(got[0]))
	})
}
