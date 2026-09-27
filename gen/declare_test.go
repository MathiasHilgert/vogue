package gen_test

import (
	"strings"
	"testing"

	"github.com/MathiasHilgert/vogue"
	"github.com/MathiasHilgert/vogue/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// declaringRule is a rule that needs a package-level declaration, the shape the
// `regex` rule of the built-in catalogue has: the pattern is compiled once, at
// process start, and the constructor only matches against it.
func declaringRule() vogue.Rule {
	name := func(param string) string { return "_vogueRe" + strings.ToUpper(param[:1]) + param[1:] }
	return vogue.Rule{
		Name:    "pattern",
		Kinds:   vogue.Kinds(vogue.String),
		Doc:     "Rejects values the pattern does not match.",
		Param:   vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamString},
		Message: "{{.Field}} must match {{.Param}}",
		Imports: []string{"regexp"},
		Declare: func(c vogue.EmitContext) string {
			return "var " + name(c.Param) + " = regexp.MustCompile(`" + c.Param + "`)"
		},
		Emit: func(c vogue.EmitContext) string {
			return name(c.Param) + ".MatchString(" + c.Var + ")"
		},
		Examples: vogue.Examples{
			Valid:   []vogue.Example{{Param: "abc", In: "abc"}},
			Invalid: []vogue.Example{{Param: "abc", In: "x"}},
		},
	}
}

func TestGenerator_Declarations(t *testing.T) {
	// Arrange
	set := &vogue.RuleSet{}
	require.NoError(t, set.Add(declaringRule()))

	t.Run("emits the declaration a rule asks for, after the imports", func(t *testing.T) {
		// Arrange
		pkg := parseSource(t, "//vogue:string Code pattern=abc\n", set)

		// Act
		files, err := filesOf(t, gen.Options{Package: pkg, Rules: set})

		// Assert
		require.NoError(t, err)
		require.NotEmpty(t, files)
		code := string(files[0].Content)
		assert.Contains(t, code, "var _vogueReAbc = regexp.MustCompile(`abc`)")
		assert.Contains(t, code, "_vogueReAbc.MatchString(value)")
		assert.Less(t, strings.Index(code, `"regexp"`), strings.Index(code, "var _vogueReAbc"),
			"the declaration must come after the import block")
		assert.Less(t, strings.Index(code, "var _vogueReAbc"), strings.Index(code, "type Code"),
			"the declaration must come before the generated types")
	})

	t.Run("emits one declaration for two directives sharing a parameter", func(t *testing.T) {
		// Arrange
		pkg := parseSource(t, "//vogue:string Code pattern=abc\n\n//vogue:string Other pattern=abc\n", set)

		// Act
		files, err := filesOf(t, gen.Options{Package: pkg, Rules: set})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 1, strings.Count(string(files[0].Content), "var _vogueReAbc ="))
	})

	t.Run("emits one declaration per distinct parameter", func(t *testing.T) {
		// Arrange
		pkg := parseSource(t, "//vogue:string Code pattern=abc\n\n//vogue:string Other pattern=xyz\n", set)

		// Act
		files, err := filesOf(t, gen.Options{Package: pkg, Rules: set})

		// Assert
		require.NoError(t, err)
		code := string(files[0].Content)
		assert.Contains(t, code, "var _vogueReAbc = regexp.MustCompile(`abc`)")
		assert.Contains(t, code, "var _vogueReXyz = regexp.MustCompile(`xyz`)")
	})
}
