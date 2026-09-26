package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixture writes a directive package into a fresh temporary directory.
func fixture(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	source := "package tab\n\n// Title is the name of a tab.\n//vogue:string Title trim required min=1 max=120\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "vo.go"), []byte(source), 0o600))
	return dir
}

func TestRun(t *testing.T) {
	t.Run("generates the value objects of a directory", func(t *testing.T) {
		// Arrange
		dir := fixture(t)
		var stdout, stderr bytes.Buffer

		// Act
		code := run([]string{"-dir", dir}, &stdout, &stderr)

		// Assert
		assert.Equal(t, 0, code)
		assert.Empty(t, stderr.String())
		assert.FileExists(t, filepath.Join(dir, "vo_vogue.go"))
		assert.FileExists(t, filepath.Join(dir, "vo_vogue_test.go"))
	})

	t.Run("leaves the tests out when they are turned off", func(t *testing.T) {
		// Arrange
		dir := fixture(t)
		var stdout, stderr bytes.Buffer

		// Act
		code := run([]string{"-dir", dir, "-tests=false"}, &stdout, &stderr)

		// Assert
		assert.Equal(t, 0, code)
		assert.NoFileExists(t, filepath.Join(dir, "vo_vogue_test.go"))
	})

	t.Run("leaves the SQL codec out when it is turned off", func(t *testing.T) {
		// Arrange
		dir := fixture(t)
		var stdout, stderr bytes.Buffer

		// Act
		code := run([]string{"-dir", dir, "-sql=false"}, &stdout, &stderr)

		// Assert
		assert.Equal(t, 0, code)
		generated, err := os.ReadFile(filepath.Join(dir, "vo_vogue.go"))
		require.NoError(t, err)
		assert.NotContains(t, string(generated), "database/sql/driver")
	})

	t.Run("a dry run reports the files and writes none", func(t *testing.T) {
		// Arrange
		dir := fixture(t)
		var stdout, stderr bytes.Buffer

		// Act
		code := run([]string{"-dir", dir, "-dry-run"}, &stdout, &stderr)

		// Assert
		assert.Equal(t, 0, code)
		assert.Contains(t, stdout.String(), "vo_vogue.go")
		assert.NoFileExists(t, filepath.Join(dir, "vo_vogue.go"))
	})

	t.Run("exits 1 and reports the diagnostics of a broken directive", func(t *testing.T) {
		// Arrange
		dir := t.TempDir()
		source := "package tab\n\n//vogue:string Title mni=1\n"
		require.NoError(t, os.WriteFile(filepath.Join(dir, "vo.go"), []byte(source), 0o600))
		var stdout, stderr bytes.Buffer

		// Act
		code := run([]string{"-dir", dir}, &stdout, &stderr)

		// Assert
		assert.Equal(t, 1, code)
		assert.Contains(t, stderr.String(), `unknown rule "mni"`)
		assert.Contains(t, stderr.String(), "vogue: 1 error")
	})

	t.Run("exits 2 on an unknown flag and explains the usage", func(t *testing.T) {
		// Arrange
		var stdout, stderr bytes.Buffer

		// Act
		code := run([]string{"-nonsense"}, &stdout, &stderr)

		// Assert
		assert.Equal(t, 2, code)
		assert.Contains(t, stderr.String(), "-dir")
		assert.Empty(t, stdout.String())
	})

	t.Run("exits 2 when a positional argument is given", func(t *testing.T) {
		// Arrange
		var stdout, stderr bytes.Buffer

		// Act
		code := run([]string{"./..."}, &stdout, &stderr)

		// Assert
		assert.Equal(t, 2, code)
		assert.Contains(t, stderr.String(), "takes no positional arguments")
	})

	t.Run("lists the rule catalogue as an aligned table", func(t *testing.T) {
		// Arrange
		var stdout, stderr bytes.Buffer

		// Act
		code := run([]string{"-list"}, &stdout, &stderr)

		// Assert
		assert.Equal(t, 0, code)
		assert.Empty(t, stderr.String())
		out := stdout.String()
		assert.Contains(t, out, "RULE")
		assert.Contains(t, out, "KINDS")
		assert.Contains(t, out, "PARAM")
		assert.Contains(t, out, "min")
		assert.Contains(t, out, "required int")
		assert.Contains(t, out, "string, int")
		lines := bytes.Split(bytes.TrimSpace(stdout.Bytes()), []byte("\n"))
		require.Greater(t, len(lines), 2)
		column := bytes.Index(lines[0], []byte("KINDS"))
		require.Positive(t, column)
		for _, line := range lines[1:] {
			require.Greater(t, len(line), column)
			assert.Equal(t, byte(' '), line[column-1], "the kinds column is padded up to its header")
			assert.NotEqual(t, byte(' '), line[column], "the kinds column starts under its header")
		}
	})

	t.Run("prints its version", func(t *testing.T) {
		// Arrange
		var stdout, stderr bytes.Buffer

		// Act
		code := run([]string{"-version"}, &stdout, &stderr)

		// Assert
		assert.Equal(t, 0, code)
		assert.Contains(t, stdout.String(), "vogue ")
		assert.Empty(t, stderr.String())
	})
}
