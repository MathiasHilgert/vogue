package gen_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MathiasHilgert/vogue/gen"
	"github.com/MathiasHilgert/vogue/rules"
)

func TestNew_RequiresTheFailureType(t *testing.T) {
	t.Parallel()

	tests := map[string]gen.Validation{
		"no path":            {TypeName: "Validation"},
		"no type":            {ImportPath: "example.com/app/fault"},
		"an unexported type": {ImportPath: "example.com/app/fault", TypeName: "validation"},
		"nothing at all":     {},
		"an invalid type":    {ImportPath: "example.com/app/fault", TypeName: "1Validation"},
	}
	for name, validation := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			pkg := parseSource(t, "//vogue:string Code required\n", rules.MustSet())

			// Act
			_, err := gen.New(gen.Options{Package: pkg, Validation: validation})

			// Assert
			require.Error(t, err)
			assert.Contains(t, err.Error(), "Options.Validation")
		})
	}
}

func generateWith(t *testing.T, opts gen.Options, directives string) string {
	t.Helper()

	opts.Package = parseSource(t, directives, rules.MustSet())
	files, err := filesOf(t, opts)
	require.NoError(t, err)

	return string(files[0].Content)
}

func TestGenerator_FailureType(t *testing.T) {
	t.Parallel()

	t.Run("records failures on the type of the consumer, imported from its package", func(t *testing.T) {
		t.Parallel()

		// Act
		code := generateWith(t, gen.Options{Validation: gen.Validation{ImportPath: "example.com/app/fault", TypeName: "Failures"}},
			"//vogue:string Code required\n")

		// Assert
		assert.Contains(t, code, "\t\"example.com/app/fault\"\n")
		assert.Contains(t, code, "var failures fault.Failures")
		assert.Contains(t, code, `failures.Add("code", "required", "is required")`)
	})

	t.Run("uses the type unqualified inside its own package", func(t *testing.T) {
		t.Parallel()

		// Act
		code := generateWith(t, gen.Options{
			ImportPath: "example.com/app/fault",
			Validation: gen.Validation{ImportPath: "example.com/app/fault", TypeName: "Validation"},
		}, "//vogue:string Code required\n")

		// Assert
		assert.Contains(t, code, "var failures Validation")
		assert.NotContains(t, code, "example.com/app/fault", "a package does not import itself")
	})

	t.Run("keeps a receiver from shadowing the package of the failure type", func(t *testing.T) {
		t.Parallel()

		// Act
		code := generateWith(t, gen.Options{Validation: gen.Validation{ImportPath: "example.com/app/fault", TypeName: "Validation"}},
			"//vogue:string Fault required\n")

		// Assert
		assert.Contains(t, code, "var faultValue Fault // stays zero unless every rule passes")
		assert.Contains(t, code, "var failures fault.Validation")
	})

	t.Run("imports nothing of vogue", func(t *testing.T) {
		t.Parallel()

		// Act
		code := generateWith(t, gen.Options{Validation: testValidation, SQL: true, Schema: true},
			"//vogue:string Code trim required len=2\n//vogue:int Covers min=1\n//vogue:enum Kind a,b\n//vogue:id ID\n")

		// Assert
		for _, line := range strings.Split(code, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), `"github.com/MathiasHilgert/vogue/`) {
				assert.Equal(t, `"github.com/MathiasHilgert/vogue/examples/validation"`, strings.TrimSpace(line))
			}
		}
	})
}

func TestGenerator_Preconditions(t *testing.T) {
	t.Parallel()

	t.Run("required returns at once with the failures recorded so far", func(t *testing.T) {
		t.Parallel()

		// Act
		code := generateWith(t, gen.Options{Validation: testValidation}, "//vogue:string Code required len=2\n")

		// Assert
		assert.Contains(t, code, "\tif value == \"\" {\n\t\tfailures.Add(\"code\", \"required\", \"is required\")\n\n\t\treturn code, failures.Err()\n\t}\n")
		assert.Contains(t, code, "\tif utf8.RuneCountInString(value) != lengthParameter {\n\t\tfailures.Add(\"code\", \"len\"")
	})

	t.Run("a check that is no precondition keeps accumulating", func(t *testing.T) {
		t.Parallel()

		// Act
		code := generateWith(t, gen.Options{Validation: testValidation}, "//vogue:string Code len=2 required\n")

		// Assert
		assert.Equal(t, 1, strings.Count(code, "return code, failures.Err()"), "only required returns early")
	})
}

func TestGenerator_RegexMessage(t *testing.T) {
	t.Parallel()

	t.Run("replaces the generic message of the regex rule", func(t *testing.T) {
		t.Parallel()

		// Act
		code := generateWith(t, gen.Options{Validation: testValidation},
			"//vogue:string Code regex=^[A-Z]{2}$ regex_message=\"must be a two-letter code\"\n")

		// Assert
		assert.Contains(t, code, `failures.Add("code", "regex", "must be a two-letter code")`)
		assert.NotContains(t, code, "must match the pattern")
	})

	t.Run("keeps the generic message without one", func(t *testing.T) {
		t.Parallel()

		// Act
		code := generateWith(t, gen.Options{Validation: testValidation}, "//vogue:string Code regex=^[A-Z]{2}$\n")

		// Assert
		assert.Contains(t, code, `failures.Add("code", "regex", "must match the pattern ^[A-Z]{2}$")`)
	})
}

func TestGenerator_Codecs(t *testing.T) {
	t.Parallel()

	t.Run("has only the text codec: no JSON codec of its own", func(t *testing.T) {
		t.Parallel()

		// Act
		code := generateWith(t, gen.Options{Validation: testValidation, SQL: true},
			"//vogue:string Code\n//vogue:int Covers\n//vogue:decimal Rate\n//vogue:enum Kind a,b\n//vogue:id ID\n//vogue:id Number int64\n")

		// Assert
		assert.NotContains(t, code, "MarshalJSON")
		assert.NotContains(t, code, "UnmarshalJSON")
		assert.Equal(t, 6, strings.Count(code, ") MarshalText() ([]byte, error) {"))
		assert.Equal(t, 6, strings.Count(code, "cannot marshal the zero"))
		assert.Equal(t, 6+6+1, strings.Count(code, "errors.ErrUnsupported"),
			"the zero value and an unsupported Scan source per kind, and the float a decimal refuses")
	})

	t.Run("compares with == unless the kind carries a scale", func(t *testing.T) {
		t.Parallel()

		// Act
		code := generateWith(t, gen.Options{Validation: testValidation},
			"//vogue:string Code\n//vogue:int Covers\n//vogue:decimal Rate\n//vogue:enum Kind a,b\n//vogue:id ID\n")

		// Assert
		assert.Contains(t, code, "func (code Code) Equal(other Code) bool { return code == other }")
		assert.Contains(t, code, "func (rate Rate) Equal(other Rate) bool {\n\treturn rate.set == other.set && rate.value.Cmp(other.value) == 0")
	})
}
