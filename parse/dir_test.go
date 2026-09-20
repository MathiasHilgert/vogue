package parse_test

import (
	"path/filepath"
	"testing"

	"github.com/MathiasHilgert/vogue/parse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDir(t *testing.T) {
	t.Parallel()

	// Arrange.
	dir := filepath.Join("testdata", "tab")

	// Act.
	pkg, err := parse.Dir(dir, testRules(t))

	// Assert.
	require.NoError(t, err)
	assert.Equal(t, dir, pkg.Path)
	assert.Equal(t, "tab", pkg.Name)

	names := make([]string, 0, 3)
	for _, d := range pkg.Directives() {
		names = append(names, d.Name)
	}
	assert.Equal(t, []string{"Title", "Covers", "TabStatus", "TabID"}, names,
		"generated and test files must be skipped")

	t.Run("doc comes from the lines above the directive", func(t *testing.T) {
		assert.Equal(t, "Title is the name of a tab.", pkg.Directives()[0].Doc)
	})
}

func TestDirReportsDiagnostics(t *testing.T) {
	t.Parallel()

	// Arrange.
	dir := filepath.Join("testdata", "broken")

	// Act.
	pkg, err := parse.Dir(dir, testRules(t))

	// Assert.
	require.Error(t, err)
	assert.Nil(t, pkg)
	assert.Equal(t,
		filepath.Join(dir, "vo.go")+`:3:22: unknown rule "mni" for kind string (did you mean "min"?)`,
		err.Error())
}

func TestDirMissing(t *testing.T) {
	t.Parallel()

	// Arrange / Act.
	pkg, err := parse.Dir(filepath.Join("testdata", "nope"), testRules(t))

	// Assert.
	require.Error(t, err)
	assert.Nil(t, pkg)
}
