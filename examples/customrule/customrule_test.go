package customrule_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MathiasHilgert/vogue/examples/customrule/cuitrule"
	"github.com/MathiasHilgert/vogue/generator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// domainPath is the import path of the example's value-object package. The
// generator is told it explicitly so the run under test does not depend on the
// go command being able to resolve a temporary directory.
const domainPath = "github.com/MathiasHilgert/vogue/examples/customrule/domain"

// generated names the files `go generate` is expected to keep up to date.
var generated = []string{"vo_vogue.go", "vo_vogue_test.go"}

func TestGoGenerate(t *testing.T) {
	t.Run("the committed output matches what the generator produces today", func(t *testing.T) {
		// Arrange
		source := filepath.Join("domain", "vo.go")
		body, err := os.ReadFile(source)
		require.NoError(t, err)

		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "vo.go"), body, 0o600))

		// Act
		err = generator.Run(
			generator.WithDir(dir),
			generator.WithImportPath(domainPath),
			generator.WithRules(cuitrule.Rule),
		)

		// Assert
		require.NoError(t, err)
		for _, name := range generated {
			fresh, err := os.ReadFile(filepath.Join(dir, name))
			require.NoError(t, err)
			committed, err := os.ReadFile(filepath.Join("domain", name))
			require.NoError(t, err)
			assert.Equal(t, string(committed), string(fresh),
				"domain/%s has drifted; re-run `go generate ./pkg/vogue/examples/...`", name)
		}
	})

	t.Run("a second run over its own output changes nothing", func(t *testing.T) {
		// Arrange
		dir := t.TempDir()
		for _, name := range append([]string{"vo.go"}, generated...) {
			body, err := os.ReadFile(filepath.Join("domain", name))
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(dir, name), body, 0o600))
		}

		// Act
		err := generator.Run(
			generator.WithDir(dir),
			generator.WithImportPath(domainPath),
			generator.WithRules(cuitrule.Rule),
		)

		// Assert
		require.NoError(t, err)
		for _, name := range generated {
			after, err := os.ReadFile(filepath.Join(dir, name))
			require.NoError(t, err)
			committed, err := os.ReadFile(filepath.Join("domain", name))
			require.NoError(t, err)
			assert.Equal(t, string(committed), string(after),
				"generating over generated output is expected to be a no-op")
		}
	})
}
