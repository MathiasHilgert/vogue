package gen

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/MathiasHilgert/vogue"
	"github.com/MathiasHilgert/vogue/parse"
)

// templates holds the source templates, one per generated shape plus the file
// header. They are embedded so the generator is a single binary with no runtime
// dependency on its own source tree.
//
//go:embed templates/*.tmpl
var templates embed.FS

// The suffixes appended to the base name of a source file to name its
// generated counterparts.
const (
	suffix     = "_vogue.go"
	testSuffix = "_vogue_test.go"
)

// FileKind tells a generated file apart from the test that proves it, so a
// caller can write, diff or skip one of the two without matching on the name.
type FileKind uint8

const (
	// CodeFile is a generated value-object source file.
	CodeFile FileKind = iota
	// TestFile is the generated test of a [CodeFile].
	TestFile
)

// String returns the name of the file kind.
func (k FileKind) String() string {
	switch k {
	case CodeFile:
		return "code"
	case TestFile:
		return "test"
	default:
		return fmt.Sprintf("FileKind(%d)", uint8(k))
	}
}

// Options configures a [Generator].
type Options struct {
	// Package is the parsed package to generate from. It is required.
	Package *parse.Package
	// Rules is the catalogue the directives were parsed against. The generator
	// reads the rules off the directives themselves, so this is carried for the
	// test generator and the CLI rather than used here.
	Rules *vogue.RuleSet
	// ImportPath is the import path of the package being generated into. When a
	// rule dispatches to a function in that same package, the call is emitted
	// unqualified and the package is not imported into itself. It may be empty.
	ImportPath string
	// OmitSQL leaves the database/sql/driver codec (Value and Scan) out of
	// the generated code. A hexagonal domain package that must not import
	// database/sql/driver sets it and converts at the persistence adapter
	// through the text codec or the accessors instead.
	OmitSQL bool
	// Schema adds a JSONSchema method to every value object, returning the
	// neutral github.com/MathiasHilgert/vogue/schema description an HTTP
	// adapter publishes in its OpenAPI document.
	Schema bool
}

// OutFile is one generated file: the path it belongs at and its formatted
// content. Nothing is written until the caller asks for it, so a generator run
// can be inspected, diffed or discarded.
type OutFile struct {
	// Kind says whether the file holds value objects or their tests.
	Kind FileKind
	// Path is the destination path of the file.
	Path string
	// Content is the gofmt-clean source.
	Content []byte
}

// Generator renders the value objects of one parsed package.
type Generator struct {
	opts Options
	tmpl *template.Template
}

// New validates the options and prepares the templates.
func New(opts Options) (*Generator, error) {
	if opts.Package == nil {
		return nil, fmt.Errorf("gen: package must not be nil")
	}
	if opts.Package.Name == "" {
		return nil, fmt.Errorf("gen: package name must not be empty")
	}

	tmpl, err := template.New("vogue").Funcs(template.FuncMap{"q": quote}).ParseFS(templates, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("gen: loading templates: %w", err)
	}
	return &Generator{opts: opts, tmpl: tmpl}, nil
}

// Files renders, for every source file that declares at least one directive,
// the value objects it declares and the test that proves them. The pair is
// returned in that order, tagged with [FileKind]. Source files without
// directives produce nothing, so the generator never leaves an empty file
// behind.
func (g *Generator) Files() ([]OutFile, error) {
	var out []OutFile
	for _, file := range g.opts.Package.Files {
		if len(file.Directives) == 0 {
			continue
		}
		content, err := g.file(file)
		if err != nil {
			return nil, err
		}
		out = append(out, OutFile{Kind: CodeFile, Path: outPath(file.Path, suffix), Content: content})

		test, err := g.testFile(file)
		if err != nil {
			return nil, err
		}
		out = append(out, OutFile{Kind: TestFile, Path: outPath(file.Path, testSuffix), Content: test})
	}
	return out, nil
}

// file renders and formats one generated file.
func (g *Generator) file(file parse.File) ([]byte, error) {
	imports := newImportSet(g.opts.ImportPath)
	decls := newDeclSet()

	var bodies bytes.Buffer
	for _, directive := range file.Directives {
		view, name, err := g.newView(directive, imports, decls)
		if err != nil {
			return nil, err
		}
		if err := g.tmpl.ExecuteTemplate(&bodies, name+".tmpl", view); err != nil {
			return nil, fmt.Errorf("gen: rendering %s: %w", directive.Name, err)
		}
	}

	var buf bytes.Buffer
	std, ext := imports.groups()
	header := fileView{Package: g.opts.Package.Name, Std: std, Ext: ext, Decls: decls.all()}
	if err := g.tmpl.ExecuteTemplate(&buf, "header.tmpl", header); err != nil {
		return nil, fmt.Errorf("gen: rendering header of %s: %w", file.Path, err)
	}
	buf.Write(bodies.Bytes())

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("gen: formatting %s: %w\n%s", file.Path, err, buf.String())
	}
	return formatted, nil
}

// outPath returns the destination of the file generated from src with the
// given suffix.
func outPath(src, suffix string) string {
	base := strings.TrimSuffix(filepath.Base(src), ".go")
	return filepath.Join(filepath.Dir(src), base+suffix)
}

// Write writes every file to its path. Each one is written to a temporary file
// in the destination directory and renamed into place, so a failed or
// interrupted run never leaves a half-written source file behind.
func Write(files []OutFile) error {
	for _, file := range files {
		if err := writeAtomic(file); err != nil {
			return err
		}
	}
	return nil
}

// writeAtomic writes one file through a temporary file and a rename.
func writeAtomic(file OutFile) error {
	dir := filepath.Dir(file.Path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(file.Path)+".*")
	if err != nil {
		return fmt.Errorf("gen: creating temporary file for %s: %w", file.Path, err)
	}
	// The temporary file is removed on every path; once the rename succeeded
	// there is nothing left at that name, and the failure is not actionable.
	defer func() { _ = os.Remove(tmp.Name()) }()

	if _, err := tmp.Write(file.Content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("gen: writing %s: %w", file.Path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("gen: closing %s: %w", file.Path, err)
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return fmt.Errorf("gen: setting mode of %s: %w", file.Path, err)
	}
	if err := os.Rename(tmp.Name(), file.Path); err != nil {
		return fmt.Errorf("gen: renaming into %s: %w", file.Path, err)
	}
	return nil
}

// quote renders a string as a Go literal.
func quote(s string) string { return strconv.Quote(s) }
