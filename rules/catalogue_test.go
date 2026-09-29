package rules_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/MathiasHilgert/vogue"
	"github.com/MathiasHilgert/vogue/gen"
	"github.com/MathiasHilgert/vogue/parse"
	"github.com/MathiasHilgert/vogue/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cataloguePath is the import path of the fixture package the whole catalogue
// is generated into.
const cataloguePath = "github.com/MathiasHilgert/vogue/rules/internal/catalogue"

// validationPath is the package of the failure type the catalogue fixture
// records its failures on.
const validationPath = "github.com/MathiasHilgert/vogue/examples/validation"

// updateCatalogue rewrites the committed fixture instead of comparing against
// it, for when a rule legitimately changes what it emits.
var updateCatalogue = flag.Bool("update", false, "rewrite the committed catalogue fixture")

func TestCatalogue_Fixture(t *testing.T) {
	t.Run("the committed catalogue fixture matches what the rules emit", func(t *testing.T) {
		// Arrange
		dir := filepath.Join("internal", "catalogue")
		pkg, err := parse.Dir(dir, rules.MustSet())
		require.NoError(t, err)

		g, err := gen.New(gen.Options{Validation: gen.Validation{ImportPath: validationPath, TypeName: "Validation"}, Package: pkg, Rules: rules.MustSet(), ImportPath: cataloguePath})
		require.NoError(t, err)

		// Act
		files, err := g.Files()

		// Assert
		require.NoError(t, err)
		require.Len(t, files, 2)
		if *updateCatalogue {
			require.NoError(t, gen.Write(files))
		}
		for _, file := range files {
			committed, err := os.ReadFile(file.Path)
			require.NoError(t, err)
			assert.Equal(t, string(committed), string(file.Content),
				"the committed %s file %s has drifted; re-run `go test ./pkg/vogue/rules -update`", file.Kind, file.Path)
		}
	})
}

func TestCatalogue_CoversEveryRule(t *testing.T) {
	t.Run("every shipped rule is exercised by a directive of the fixture", func(t *testing.T) {
		// Arrange
		pkg, err := parse.Dir(filepath.Join("internal", "catalogue"), rules.MustSet())
		require.NoError(t, err)

		used := map[string]struct{}{}
		for _, file := range pkg.Files {
			for _, directive := range file.Directives {
				for _, use := range directive.Rules {
					used[use.Rule.Name] = struct{}{}
				}
			}
		}

		// Assert
		for _, rule := range rules.All() {
			_, ok := used[rule.Name]
			assert.True(t, ok,
				"rule %q has no directive in internal/catalogue, so none of its examples is ever executed", rule.Name)
		}
	})

	t.Run("every kind a rule applies to is exercised by a directive of the fixture", func(t *testing.T) {
		// Arrange
		pkg, err := parse.Dir(filepath.Join("internal", "catalogue"), rules.MustSet())
		require.NoError(t, err)

		type pair struct {
			rule string
			kind vogue.Kind
		}
		used := map[pair]struct{}{}
		for _, file := range pkg.Files {
			for _, directive := range file.Directives {
				for _, use := range directive.Rules {
					used[pair{use.Rule.Name, directive.Kind}] = struct{}{}
				}
			}
		}

		// Assert
		for _, rule := range rules.All() {
			for _, kind := range rule.Kinds.Kinds() {
				_, ok := used[pair{rule.Name, kind}]
				assert.True(t, ok, "rule %q is never exercised on the %s kind", rule.Name, kind)
			}
		}
	})
}
