package generator_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MathiasHilgert/vogue/gen"
	"github.com/MathiasHilgert/vogue/generator"
)

func TestRun_Suffix(t *testing.T) {
	t.Run("an empty suffix writes one file and one test per value object, in snake case", func(t *testing.T) {
		// Arrange
		dir := copyDir(t, filepath.Join("testdata", "tab"))

		// Act
		err := generator.Run(withValidation, generator.WithDir(dir), generator.WithSuffix(""))

		// Assert
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"vo.go", "title.go", "title_test.go", "covers.go", "covers_test.go"}, names(t, dir))
	})

	t.Run("a custom suffix names the files after their source", func(t *testing.T) {
		// Arrange
		dir := copyDir(t, filepath.Join("testdata", "tab"))

		// Act
		err := generator.Run(withValidation, generator.WithDir(dir), generator.WithSuffix("_gen"))

		// Assert
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"vo.go", "vo_gen.go", "vo_gen_test.go"}, names(t, dir))
	})
}

func TestRun_Ownership(t *testing.T) {
	t.Run("refuses to overwrite a hand-written file and writes nothing", func(t *testing.T) {
		// Arrange
		dir := copyDir(t, filepath.Join("testdata", "tab"))
		handWritten := filepath.Join(dir, "covers.go")
		require.NoError(t, os.WriteFile(handWritten, []byte("package tab\n\n// Mine.\n"), 0o600))

		// Act
		err := generator.Run(withValidation, generator.WithDir(dir), generator.WithSuffix(""))

		// Assert
		require.ErrorIs(t, err, gen.ErrNotGenerated)
		kept, readErr := os.ReadFile(handWritten)
		require.NoError(t, readErr)
		assert.Equal(t, "package tab\n\n// Mine.\n", string(kept))
		assert.NoFileExists(t, filepath.Join(dir, "title.go"))
	})

	t.Run("removes the files a previous run generated and this one does not", func(t *testing.T) {
		// Arrange
		dir := copyDir(t, filepath.Join("testdata", "tab"))
		require.NoError(t, generator.Run(withValidation, generator.WithDir(dir)))
		require.FileExists(t, filepath.Join(dir, "vo_vogue.go"))

		// Act
		err := generator.Run(withValidation, generator.WithDir(dir), generator.WithSuffix(""))

		// Assert
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"vo.go", "title.go", "title_test.go", "covers.go", "covers_test.go"}, names(t, dir))
	})

	t.Run("removes every generated file once the last directive is gone", func(t *testing.T) {
		// Arrange
		dir := copyDir(t, filepath.Join("testdata", "tab"))
		require.NoError(t, generator.Run(withValidation, generator.WithDir(dir)))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "vo.go"), []byte("package tab\n"), 0o600))

		// Act
		err := generator.Run(withValidation, generator.WithDir(dir))

		// Assert
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"vo.go"}, names(t, dir))
	})

	t.Run("a dry run reports what it would remove and removes nothing", func(t *testing.T) {
		// Arrange
		dir := copyDir(t, filepath.Join("testdata", "tab"))
		require.NoError(t, generator.Run(withValidation, generator.WithDir(dir)))
		var stdout bytes.Buffer

		// Act
		err := generator.Run(withValidation, generator.WithDir(dir), generator.WithSuffix(""),
			generator.WithDryRun(true), generator.WithStdout(&stdout))

		// Assert
		require.NoError(t, err)
		assert.Contains(t, stdout.String(), "remove "+filepath.Join(dir, "vo_vogue.go"))
		assert.FileExists(t, filepath.Join(dir, "vo_vogue.go"))
	})
}
