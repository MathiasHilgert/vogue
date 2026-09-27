package textjson_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestImports_StandardLibraryOnly pins the promise the package makes to every
// project that depends on generated code: importing it pulls in nothing but
// the standard library.
func TestImports_StandardLibraryOnly(t *testing.T) {
	t.Parallel()

	// Arrange
	sources, err := filepath.Glob("*.go")
	require.NoError(t, err)
	fset := token.NewFileSet()

	for _, source := range sources {
		if strings.HasSuffix(source, "_test.go") {
			continue
		}
		content, err := os.ReadFile(source)
		require.NoError(t, err)

		// Act
		file, err := parser.ParseFile(fset, source, content, parser.ImportsOnly)
		require.NoError(t, err)

		// Assert
		for _, spec := range file.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			require.NoError(t, err)
			assert.NotContains(t, strings.Split(path, "/")[0], ".",
				"%s imports %s, which is outside the standard library", source, path)
		}
	}
}
