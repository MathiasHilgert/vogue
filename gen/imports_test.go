package gen_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// vogueModule is the module the generator belongs to.
const vogueModule = "github.com/MathiasHilgert/vogue"

// TestGenerate_ImportsNothingOfVogue proves the promise of the design: what
// vogue generates, tests included, depends on the standard library, on the
// packages of the kinds it uses, and on the consumer's own failure type, and
// on nothing of vogue and no test framework.
//
// It asks the go command for the dependencies of the golden packages, tests
// included. Only golden packages with no hand-written test qualify, because a
// hand-written test may import a framework.
func TestGenerate_ImportsNothingOfVogue(t *testing.T) {
	failureType := vogueModule + "/examples/validation"
	cases := []struct {
		pkg     string
		allowed []string
	}{
		{pkg: "./testdata/strict/domain", allowed: []string{failureType}},
		{pkg: "./testdata/strict/persistence", allowed: []string{failureType}},
		{pkg: "../examples/customrule/domain", allowed: []string{failureType, vogueModule + "/examples/customrule/cuit"}},
	}

	for _, test := range cases {
		t.Run(test.pkg, func(t *testing.T) {
			// Arrange
			command := exec.Command("go", "list", "-deps", "-test", "-f", "{{.ImportPath}}", test.pkg)

			// Act
			out, err := command.Output()

			// Assert
			require.NoError(t, err)
			var unexpected []string
			for _, path := range strings.Fields(string(out)) {
				path, _, _ = strings.Cut(path, " ")
				switch {
				case strings.Contains(path, "testify"):
					unexpected = append(unexpected, path)
				case strings.HasPrefix(path, vogueModule) && !ownedBy(path, test.pkg, test.allowed):
					unexpected = append(unexpected, path)
				}
			}
			assert.Empty(t, unexpected, "generated code and its tests import only the standard library, the kinds' packages and the failure type")
		})
	}
}

// ownedBy reports whether an import path is the package under test, its test
// binary, or one of the packages the generated code may import.
func ownedBy(path, pkg string, allowed []string) bool {
	name := pkg[strings.LastIndex(pkg, "/")+1:]
	for _, allow := range allowed {
		if path == allow {
			return true
		}
	}
	return strings.HasSuffix(strings.TrimSuffix(strings.TrimSuffix(path, ".test"), "_test"), "/"+name)
}
