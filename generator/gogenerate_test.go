package generator_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MathiasHilgert/vogue/generator"
)

// TestGoGenerate_SuffixSwitch runs the command the way a consumer does, with
// go generate inside a module of its own, and switches the suffix between two
// runs. go generate lists the files of the package before it runs the first
// directive and opens every one of them afterwards, so a generated file the
// run deletes must not be one go generate has yet to open.
func TestGoGenerate_SuffixSwitch(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the command and runs go generate")
	}

	// Arrange
	_, here, _, _ := runtime.Caller(0)
	repository := filepath.Dir(filepath.Dir(here))
	module := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(module, name)), 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(module, name), []byte(content), 0o600))
	}
	goMod, err := os.ReadFile(filepath.Join(repository, "go.mod"))
	require.NoError(t, err)
	goSum, err := os.ReadFile(filepath.Join(repository, "go.sum"))
	require.NoError(t, err)
	requires := string(goMod)[strings.Index(string(goMod), "require"):]
	write("go.mod", "module consumer\n\ngo 1.27\n\nrequire github.com/MathiasHilgert/vogue v0.0.0\n\n"+
		requires+"\nreplace github.com/MathiasHilgert/vogue => "+repository+"\n")
	write("go.sum", string(goSum))
	write("fault/fault.go", strings.ReplaceAll(generator.ValidationReference, "\n\t", "\n")[1:])
	directives := "\npackage place\n\n// CountryCode is an ISO 3166-1 alpha-2 country code.\n" +
		"//vogue:string CountryCode trim upper required len=2 example=AR\n\n" +
		"// GeoNamesID identifies a GeoNames record.\n//vogue:id GeoNamesID int64\n"
	generate := func(suffixFlag string) (string, error) {
		t.Helper()
		write("place/vo.go", "//go:generate go run github.com/MathiasHilgert/vogue/cmd/vogue -validation=consumer/fault.Validation "+suffixFlag+"\n"+directives)
		command := exec.Command("go", "generate", "./...")
		command.Dir = module
		command.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "GOPROXY=off", "GOWORK=off")
		out, err := command.CombinedOutput()
		return string(out), err
	}
	out, err := generate("")
	require.NoError(t, err, out)
	require.FileExists(t, filepath.Join(module, "place", "vo_vogue.go"))

	// Act
	out, err = generate("-suffix=")

	// Assert
	require.NoError(t, err, "switching the suffix under go generate must succeed on the first run:\n%s", out)
	assert.FileExists(t, filepath.Join(module, "place", "country_code.go"))
	assert.FileExists(t, filepath.Join(module, "place", "geo_names_id_test.go"))

	build := exec.Command("go", "vet", "./...")
	build.Dir = module
	build.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "GOPROXY=off", "GOWORK=off")
	vetOut, err := build.CombinedOutput()
	require.NoError(t, err, "the package must compile right after the switch:\n%s", vetOut)

	// A later run leaves only the files the directives generate.
	out, err = generate("-suffix=")
	require.NoError(t, err, out)
	assert.NoFileExists(t, filepath.Join(module, "place", "vo_vogue.go"))
	assert.NoFileExists(t, filepath.Join(module, "place", "vo_vogue_test.go"))
}
