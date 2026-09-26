package gen

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// The thresholds goconst applies with its default settings: a string of at
// least minHoistLen bytes written at least minHoistCount times is reported.
const (
	minHoistLen   = 3
	minHoistCount = 3
)

// hoistRepeatedStrings names every string literal a generated test writes
// often enough for goconst to report it, and refers to it by that name.
//
// The rows of a generated table are data declared by the rules, and the same
// example — "Tortilla", "EUR" — legitimately appears in several of them. A
// consumer linting generated code without exclusions would still be told to
// make each one a constant, so the generator does: the repeated strings are
// declared once, in a const block after the imports, named after their
// content.
func hoistRepeatedStrings(src []byte) ([]byte, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("gen: parsing a generated test to hoist its strings: %w", err)
	}

	lits := map[string][]*ast.BasicLit{}
	var order []string
	ast.Inspect(file, func(n ast.Node) bool {
		if _, isImport := n.(*ast.ImportSpec); isImport {
			return false
		}
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		value, err := strconv.Unquote(lit.Value)
		if err != nil || len(value) < minHoistLen {
			return true
		}
		if _, seen := lits[value]; !seen {
			order = append(order, value)
		}
		lits[value] = append(lits[value], lit)
		return true
	})

	type replacement struct {
		start, end int
		name       string
	}
	var (
		replacements []replacement
		decls        []string
	)
	names := map[string]struct{}{}
	for _, value := range order {
		uses := lits[value]
		if len(uses) < minHoistCount {
			continue
		}
		name := hoistName(value, names)
		decls = append(decls, "\t"+name+" = "+strconv.Quote(value))
		for _, lit := range uses {
			replacements = append(replacements, replacement{
				start: fset.Position(lit.Pos()).Offset,
				end:   fset.Position(lit.End()).Offset,
				name:  name,
			})
		}
	}
	if len(decls) == 0 {
		return src, nil
	}

	sort.Slice(replacements, func(i, j int) bool { return replacements[i].start > replacements[j].start })
	out := append([]byte(nil), src...)
	for _, r := range replacements {
		out = append(out[:r.start], append([]byte(r.name), out[r.end:]...)...)
	}

	block := "\n// The strings below are shared by several rows of the tables in this file.\nconst (\n" +
		strings.Join(decls, "\n") + "\n)\n"
	at := fset.Position(file.Name.End()).Offset
	if len(file.Imports) > 0 {
		at = importsEnd(fset, file)
	}
	out = append(out[:at], append([]byte("\n"+block), out[at:]...)...)

	formatted, err := format.Source(out)
	if err != nil {
		return nil, fmt.Errorf("gen: formatting a generated test after hoisting its strings: %w", err)
	}
	return formatted, nil
}

// importsEnd returns the offset just past the last import declaration.
func importsEnd(fset *token.FileSet, file *ast.File) int {
	end := 0
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			continue
		}
		end = fset.Position(gen.End()).Offset
	}
	return end
}

// hoistName returns the constant a hoisted string is declared as: "example"
// followed by the letters and digits of its words in PascalCase, or a number
// when it has none, made unique against the names already taken.
func hoistName(value string, taken map[string]struct{}) string {
	var b strings.Builder
	b.WriteString("example")
	for _, word := range strings.FieldsFunc(value, func(r rune) bool {
		return r > unicode.MaxASCII || (!unicode.IsLetter(r) && !unicode.IsDigit(r))
	}) {
		b.WriteString(strings.ToUpper(word[:1]))
		b.WriteString(word[1:])
	}
	base := b.String()
	if base == "example" {
		base = "example1"
	}
	name := base
	for i := 2; ; i++ {
		if _, dup := taken[name]; !dup {
			break
		}
		name = base + strconv.Itoa(i)
		if base == "example1" {
			name = "example" + strconv.Itoa(i)
		}
	}
	taken[name] = struct{}{}
	return name
}
