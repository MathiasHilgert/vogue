package parse_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/MathiasHilgert/vogue"
	"github.com/MathiasHilgert/vogue/parse"
	"github.com/stretchr/testify/require"
)

// testRules builds the small rule catalogue the parser tests validate against,
// so they do not depend on the built-in rules shipped later.
func testRules(t *testing.T) *vogue.RuleSet {
	t.Helper()

	always := func(vogue.EmitContext) string { return "true" }
	set := &vogue.RuleSet{}
	err := set.Add(
		vogue.Rule{
			Name: "required", Kinds: vogue.Kinds(vogue.String, vogue.Int),
			Message: "{{.Field}} is required", Emit: always,
		},
		vogue.Rule{
			Name: "trim", Kinds: vogue.Kinds(vogue.String),
			Message: "{{.Field}} must not be padded", Emit: always,
		},
		vogue.Rule{
			Name: "lower", Kinds: vogue.Kinds(vogue.String),
			Message: "{{.Field}} must be lower case", Emit: always,
		},
		vogue.Rule{
			Name: "min", Kinds: vogue.Kinds(vogue.String, vogue.Int),
			Param:   vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamInt},
			Message: "{{.Field}} must be at least {{.Param}}", Emit: always,
		},
		vogue.Rule{
			Name: "max", Kinds: vogue.Kinds(vogue.String, vogue.Int),
			Param:   vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamInt},
			Message: "{{.Field}} must be at most {{.Param}}", Emit: always,
		},
		vogue.Rule{
			Name: "email", Kinds: vogue.Kinds(vogue.String),
			Message: "{{.Field}} must be an email", Emit: always,
		},
		vogue.Rule{
			Name: "regex", Kinds: vogue.Kinds(vogue.String),
			Param:   vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamRegex},
			Message: "{{.Field}} must match {{.Param}}", Emit: always,
		},
		vogue.Rule{
			Name: "oneof", Kinds: vogue.Kinds(vogue.String),
			Param:   vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamList},
			Message: "{{.Field}} must be one of {{.Param}}", Emit: always,
		},
	)
	require.NoError(t, err)
	return set
}

// parseSrc runs the parser over one in-memory file named vo.go.
func parseSrc(t *testing.T, src string) (*parse.Package, error) {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "vo.go", src, parser.ParseComments)
	require.NoError(t, err)
	return parse.Files(fset, []*ast.File{file}, testRules(t))
}

// at returns the "vo.go:line:col" prefix of the first occurrence of needle in
// src, so expected diagnostics stay readable and never hard-code a column.
func at(t *testing.T, src, needle string) string {
	t.Helper()

	i := strings.Index(src, needle)
	require.GreaterOrEqual(t, i, 0, "needle %q not found", needle)
	line := strings.Count(src[:i], "\n") + 1
	col := i - (strings.LastIndex(src[:i], "\n") + 1) + 1
	return fmt.Sprintf("vo.go:%d:%d", line, col)
}

// ruleUses flattens the rules of a directive into "name" or "name=param".
func ruleUses(d parse.Directive) []string {
	out := make([]string, len(d.Rules))
	for i, r := range d.Rules {
		out[i] = r.Rule.Name
		if r.Param != "" {
			out[i] += "=" + r.Param
		}
	}
	return out
}
