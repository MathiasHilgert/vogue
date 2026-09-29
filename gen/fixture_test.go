package gen_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MathiasHilgert/vogue/gen"
	"github.com/MathiasHilgert/vogue/gen/internal/testrules"
	"github.com/MathiasHilgert/vogue/parse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixturePath is the import path of the fixture package. The call-backed rule
// points at it, which is what makes the generator emit an unqualified call
// rather than importing the package into itself.
const fixturePath = "github.com/MathiasHilgert/vogue/gen/internal/fixture"

func TestGenerate_Fixture(t *testing.T) {
	t.Run("the committed fixture output matches what the generator produces", func(t *testing.T) {
		// Arrange
		dir := filepath.Join("internal", "fixture")
		pkg, err := parse.Dir(dir, testrules.Set(fixturePath))
		require.NoError(t, err)

		g, err := gen.New(gen.Options{Validation: testValidation, Package: pkg, ImportPath: fixturePath, SQL: true})
		require.NoError(t, err)

		// Act
		files, err := g.Files()

		// Assert
		require.NoError(t, err)
		require.Len(t, files, 2)
		if *update {
			require.NoError(t, gen.Write(files))
		}
		for _, file := range files {
			committed, err := os.ReadFile(file.Path)
			require.NoError(t, err)
			assert.Equal(t, string(committed), string(file.Content),
				"the committed %s file %s has drifted; re-run `go test ./pkg/vogue/gen -update`", file.Kind, file.Path)
		}
	})
}
