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

// hoistRepeatedStrings names every string literal the generated tests of one
// package write often enough for goconst to report it, and refers to it by that
// name.
//
// The rows of a generated table are data declared by the rules, and the same
// example — "Tortilla", "EUR" — legitimately appears in several of them, and in
// the tests of several value objects. goconst counts across the whole package,
// so the literals are counted across every file given; each function that uses
// a repeated string then declares it as a constant of its own. Declaring the
// constants inside the functions, rather than once at package level, is what
// keeps two generated files of the same package from declaring the same name.
func hoistRepeatedStrings(sources [][]byte) ([][]byte, error) {
	fset := token.NewFileSet()
	files := make([]*ast.File, len(sources))
	counts := map[string]int{}
	var order []string
	for i, src := range sources {
		file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("gen: parsing a generated test to hoist its strings: %w", err)
		}
		files[i] = file
		for _, lit := range stringLiterals(file) {
			value, _ := strconv.Unquote(lit.Value)
			if counts[value] == 0 {
				order = append(order, value)
			}
			counts[value]++
		}
	}

	names := map[string]string{}
	taken := map[string]struct{}{}
	for _, value := range order {
		if counts[value] >= minHoistCount {
			names[value] = hoistName(value, taken)
		}
	}

	out := make([][]byte, len(sources))
	for i, file := range files {
		hoisted, err := hoistFile(fset, file, sources[i], names)
		if err != nil {
			return nil, err
		}
		out[i] = hoisted
	}
	return out, nil
}

// stringLiterals returns the string literals of a file long enough for goconst
// to count, leaving out the import paths.
func stringLiterals(file *ast.File) []*ast.BasicLit {
	var out []*ast.BasicLit
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
		out = append(out, lit)
		return true
	})
	return out
}

// hoistEdit replaces the bytes from start to end of a source with text.
type hoistEdit struct {
	start, end int
	text       string
}

// hoistFile rewrites one file: in every function, the literals that have a name
// are replaced by it, and the names used are declared at the top of the body.
func hoistFile(fset *token.FileSet, file *ast.File, src []byte, names map[string]string) ([]byte, error) {
	var edits []hoistEdit
	for _, decl := range file.Decls {
		function, ok := decl.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}
		edits = append(edits, hoistFunction(fset, function, names)...)
	}
	if len(edits) == 0 {
		return src, nil
	}

	sort.SliceStable(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	out := append([]byte(nil), src...)
	for _, e := range edits {
		out = append(out[:e.start], append([]byte(e.text), out[e.end:]...)...)
	}
	formatted, err := format.Source(out)
	if err != nil {
		return nil, fmt.Errorf("gen: formatting a generated test after hoisting its strings: %w", err)
	}
	return formatted, nil
}

// hoistFunction returns the edits of one function: every named literal
// replaced by its name, and the declaration of the names it uses.
func hoistFunction(fset *token.FileSet, function *ast.FuncDecl, names map[string]string) []hoistEdit {
	var (
		edits []hoistEdit
		used  []string
	)
	seen := map[string]struct{}{}
	ast.Inspect(function.Body, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		value, err := strconv.Unquote(lit.Value)
		name, hoisted := names[value]
		if err != nil || !hoisted {
			return true
		}
		edits = append(edits, hoistEdit{
			start: fset.Position(lit.Pos()).Offset,
			end:   fset.Position(lit.End()).Offset,
			text:  name,
		})
		if _, dup := seen[value]; !dup {
			seen[value] = struct{}{}
			used = append(used, value)
		}
		return true
	})
	if len(used) == 0 {
		return nil
	}
	at := fset.Position(function.Body.Lbrace).Offset + 1
	return append(edits, hoistEdit{start: at, end: at, text: constDecl(used, names)})
}

// constDecl declares the named strings a function uses, followed by the blank
// line that separates them from the body.
func constDecl(values []string, names map[string]string) string {
	if len(values) == 1 {
		return "\n\tconst " + names[values[0]] + " = " + strconv.Quote(values[0]) + "\n"
	}
	var b strings.Builder
	b.WriteString("\n\tconst (\n")
	for _, value := range values {
		b.WriteString("\t\t" + names[value] + " = " + strconv.Quote(value) + "\n")
	}
	b.WriteString("\t)\n")
	return b.String()
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
