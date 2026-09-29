// Package generator is the entry point a generator binary calls.
//
// It is the pipeline every vogue front end runs — resolve the rule catalogue,
// parse the directives of a directory, render the value objects and their
// tests, write them next to their source — behind a single function with
// functional options:
//
//	//go:generate go run github.com/MathiasHilgert/vogue/cmd/vogue
//
//	func main() {
//	    if err := generator.Run(); err != nil {
//	        log.Fatal(err)
//	    }
//	}
//
// # Why it is not in package vogue
//
// The pipeline depends on parse, gen and rules, and all three depend on vogue
// for [vogue.Rule] and friends. Putting Run in vogue itself would close that
// cycle, so the entry point lives one package down instead. `vogue` stays
// the definition of what a rule and an error are; this package is the program
// that uses them.
//
// # Extension
//
// A project that needs its own rules builds its own binary and passes them in:
//
//	generator.Run(generator.WithRules(cuit.Rule))
//
// The extra rules are added on top of the built-in catalogue, and a custom rule
// whose name is already taken by a built-in is refused rather than silently
// shadowing it. [WithoutBuiltins] drops the catalogue entirely, for a project
// that wants only its own vocabulary.
//
// # Diagnostics
//
// Directive problems are reported the way a compiler reports them: one per
// line on the configured standard error, each with a file:line:column position,
// and Run returns an error stating how many there were. Nothing is written when
// a package has a single broken directive, so a failed run never leaves a
// package half generated.
package generator

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/MathiasHilgert/vogue"
	"github.com/MathiasHilgert/vogue/gen"
	"github.com/MathiasHilgert/vogue/parse"
	"github.com/MathiasHilgert/vogue/rules"
)

// config is the resolved configuration of one run. Its zero value is not
// usable; [newConfig] fills in the defaults before any option is applied.
type config struct {
	dir            string
	importPath     string
	custom         []vogue.Rule
	withoutBuiltin bool
	tests          bool
	sql            bool
	suffix         string
	perValueObject bool
	schema         bool
	dryRun         bool
	stdout, stderr io.Writer
}

// Option configures a [Run]. Options are applied in the order they are given,
// so a later one wins over an earlier one of the same kind.
type Option func(*config)

// WithDir sets the directory to generate from. It defaults to the directory of
// $GOFILE when the generator runs under `go generate`, and to the working
// directory otherwise.
func WithDir(dir string) Option {
	return func(c *config) { c.dir = dir }
}

// WithImportPath sets the import path of the package being generated into. It
// only matters when a rule dispatches to a function declared in that same
// package, which the generator then calls unqualified instead of importing the
// package into itself. When it is not given, it is resolved with `go list`.
func WithImportPath(path string) Option {
	return func(c *config) { c.importPath = path }
}

// WithRules adds rules on top of the built-in catalogue. A rule whose name is
// already taken by a built-in is refused: shadowing one silently would make the
// same directive mean different things in two packages of the same project.
func WithRules(extra ...vogue.Rule) Option {
	return func(c *config) { c.custom = append(c.custom, extra...) }
}

// WithoutBuiltins drops the built-in catalogue, leaving only the rules given
// with [WithRules].
func WithoutBuiltins() Option {
	return func(c *config) { c.withoutBuiltin = true }
}

// WithTests turns the generated tests on or off. They are on by default,
// because a value object that arrives without its test is the boilerplate this
// generator exists to remove.
func WithTests(on bool) Option {
	return func(c *config) { c.tests = on }
}

// WithSuffix sets what is appended to the base name of a source file to name
// the files generated from it; it defaults to "_vogue", so vo.go generates
// vo_vogue.go and vo_vogue_test.go. The empty suffix writes one file and one
// test per value object instead, named after it in snake case: CountryCode is
// written to country_code.go and country_code_test.go.
func WithSuffix(suffix string) Option {
	return func(c *config) { c.suffix, c.perValueObject = suffix, suffix == "" }
}

// WithSQL turns the database/sql/driver codec — Value and Scan — on or off.
// It is on by default. A hexagonal domain package whose linter forbids
// importing database/sql/driver turns it off and converts at the persistence
// adapter, through the text codec or the accessors.
func WithSQL(on bool) Option {
	return func(c *config) { c.sql = on }
}

// WithSchema adds a JSONSchema method to every value object, returning the
// neutral github.com/MathiasHilgert/vogue/schema description an HTTP adapter
// publishes in its OpenAPI document. It is off by default.
func WithSchema(on bool) Option {
	return func(c *config) { c.schema = on }
}

// WithDryRun reports the files that would be written, with their sizes, and
// writes nothing.
func WithDryRun(on bool) Option {
	return func(c *config) { c.dryRun = on }
}

// WithStdout redirects the run's ordinary output, which is what a dry run
// prints. It defaults to os.Stdout.
func WithStdout(w io.Writer) Option {
	return func(c *config) { c.stdout = w }
}

// WithStderr redirects the run's diagnostics. It defaults to os.Stderr.
func WithStderr(w io.Writer) Option {
	return func(c *config) { c.stderr = w }
}

// newConfig resolves the defaults and applies the options.
func newConfig(opts []Option) *config {
	c := &config{tests: true, sql: true, suffix: gen.DefaultSuffix, stdout: os.Stdout, stderr: os.Stderr}
	for _, opt := range opts {
		opt(c)
	}
	if c.dir == "" {
		c.dir = defaultDir()
	}
	return c
}

// defaultDir returns the directory `go generate` is invoking the binary for,
// falling back to the working directory. $GOFILE is the file carrying the
// //go:generate line, so its directory is the package being generated.
func defaultDir() string {
	if file := os.Getenv("GOFILE"); file != "" {
		if dir := filepath.Dir(file); dir != "" {
			return dir
		}
	}
	return "."
}

// Run generates the value objects of one directory.
//
// It resolves the rule catalogue, parses every hand-written Go file of the
// directory, renders the value objects each directive declares together with
// the test that proves it, and writes the pair next to its source. Generated
// files are recognised by their header and never parsed as input, so a second
// run over the same directory produces the same result.
//
// Every directive problem is reported on the configured standard error before
// Run returns, and nothing is written when there is one. A destination that
// exists without vogue's generated-code header is never overwritten: Run
// fails with [gen.ErrNotGenerated] and writes nothing. Generated files a
// previous run wrote and this one does not — a removed directive, a changed
// suffix — are removed, but only when they carry that header.
func Run(opts ...Option) error {
	c := newConfig(opts)

	set, err := c.ruleSet()
	if err != nil {
		return err
	}

	pkg, err := parse.Dir(c.dir, set)
	if err != nil {
		return c.report(err)
	}
	if err := c.refuseCallsIntoVogue(pkg); err != nil {
		return err
	}
	files, err := c.render(pkg, set)
	if err != nil {
		return err
	}

	stale, err := gen.Stale(c.dir, files)
	if err != nil {
		return err
	}
	if c.dryRun {
		for _, file := range files {
			fmt.Fprintf(c.stdout, "%s (%d bytes)\n", file.Path, len(file.Content))
		}
		for _, path := range stale {
			fmt.Fprintf(c.stdout, "remove %s\n", path)
		}
		return nil
	}
	if err := gen.Write(files); err != nil {
		return err
	}
	return c.retire(stale, pkg.Name)
}

// vogueModule is the module the generator itself belongs to. Generated code
// imports nothing from it, so a predicate the generated code calls must not
// either.
const vogueModule = "github.com/MathiasHilgert/vogue"

// refuseCallsIntoVogue fails when a directive uses a rule whose [vogue.Rule.Call]
// points at a package that is vogue or depends on it, directly or through
// other packages. The generated file would import that package, and with it
// the generator, its templates and its catalogue, into the consumer's domain.
//
// The dependency graph is asked of the go command, `go list -deps`, from the
// directory being generated. When the go command cannot answer, because the
// directory is outside a module or the package does not resolve yet, the
// generation goes ahead and a warning naming the unchecked package is written
// to the diagnostics: the compiler reports an import that does not resolve,
// and refusing here would make a directory outside a module ungeneratable. The
// package being generated into is skipped, since it is the consumer's own.
func (c *config) refuseCallsIntoVogue(pkg *parse.Package) error {
	own := c.resolveImportPath()
	checked := map[string]struct{}{}
	for _, directive := range pkg.Directives() {
		for _, use := range directive.Rules {
			call := use.Rule.Call
			if call == nil || call.Path == own {
				continue
			}
			if _, done := checked[call.Path]; done {
				continue
			}
			checked[call.Path] = struct{}{}

			dependency, err := c.vogueDependency(call.Path)
			if err != nil {
				fmt.Fprintf(c.stderr, "vogue: warning: %s: rule %q calls %s.%s, whose dependencies could not be checked for vogue: %v\n",
					use.Pos, use.Rule.Name, call.Path, call.Name, err)
				continue
			}
			if dependency != "" {
				return fmt.Errorf("vogue: %s: rule %q calls %s.%s, which depends on vogue (through %s): "+
					"generated code must not import vogue; move the predicate into a package that does not, "+
					"and keep the vogue.Rule value in a package of its own",
					use.Pos, use.Rule.Name, call.Path, call.Name, dependency)
			}
		}
	}
	return nil
}

// vogueDependency returns the first package of the vogue module that path is
// or depends on, and the empty string when there is none.
func (c *config) vogueDependency(path string) (string, error) {
	cmd := exec.Command("go", "list", "-deps", "-f", "{{.ImportPath}}", path)
	cmd.Dir = c.dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("listing its dependencies with the go command: %w", err)
	}
	for _, dependency := range strings.Fields(string(out)) {
		if dependency == vogueModule || (dependency != path && strings.HasPrefix(dependency, vogueModule+"/")) {
			return dependency, nil
		}
	}
	return "", nil
}

// render generates the files of the package, none when it has no directive.
func (c *config) render(pkg *parse.Package, set *vogue.RuleSet) ([]gen.OutFile, error) {
	if len(pkg.Directives()) == 0 {
		return nil, nil
	}
	g, err := gen.New(gen.Options{
		Package:        pkg,
		Rules:          set,
		ImportPath:     c.resolveImportPath(),
		Suffix:         c.suffix,
		PerValueObject: c.perValueObject,
		OmitSQL:        !c.sql,
		Schema:         c.schema,
	})
	if err != nil {
		return nil, err
	}
	files, err := g.Files()
	if err != nil {
		return nil, err
	}
	return c.selected(files), nil
}

// retire takes the generated files a run no longer writes out of the
// package. They carry vogue's header, which is what makes touching them safe.
//
// Outside go generate they are removed. Under go generate they cannot be:
// go generate lists the files of the package before it runs the first
// directive and opens each of them afterwards to look for more, so a file
// removed now fails the whole go generate with "no such file or directory".
// Such a file is rewritten instead into a stub go generate and the compiler
// both skip — the header, a `//go:build ignore` constraint and the package
// clause — and the next run, for which go generate no longer lists it,
// removes it.
func (c *config) retire(paths []string, packageName string) error {
	underGoGenerate := os.Getenv("GOFILE") != "" && os.Getenv("GOPACKAGE") != ""
	stub := retiredStub(packageName)
	for _, path := range paths {
		content, err := os.ReadFile(path) //nolint:gosec // a stale file of the directory being generated.
		if err != nil {
			return fmt.Errorf("vogue: reading the stale generated file %s: %w", path, err)
		}
		if !underGoGenerate || string(content) == stub {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("vogue: removing the stale generated file %s: %w", path, err)
			}
			continue
		}
		if err := gen.Write([]gen.OutFile{{Kind: gen.CodeFile, Path: path, Content: []byte(stub)}}); err != nil {
			return fmt.Errorf("vogue: retiring the stale generated file %s: %w", path, err)
		}
	}
	return nil
}

// retiredStub is what a stale generated file becomes under go generate until
// the next run removes it.
func retiredStub(packageName string) string {
	return gen.Header + "\n\n//go:build ignore\n\n" +
		"// This file was generated by an earlier run of vogue and is no longer\n" +
		"// produced. It is emptied rather than removed while go generate runs, and\n" +
		"// the next run of vogue removes it.\n\n" +
		"package " + packageName + "\n"
}

// ruleSet builds the catalogue the directives are resolved against.
func (c *config) ruleSet() (*vogue.RuleSet, error) {
	set := &vogue.RuleSet{}
	if !c.withoutBuiltin {
		builtin, err := rules.Set()
		if err != nil {
			return nil, err
		}
		set = builtin
	}
	for _, rule := range c.custom {
		if _, taken := set.Get(rule.Name); taken {
			return nil, fmt.Errorf(
				"vogue: custom rule %q collides with the built-in rule %q: rename the custom rule, or pass WithoutBuiltins to replace the catalogue",
				rule.Name, rule.Name)
		}
		if err := set.Add(rule); err != nil {
			return nil, err
		}
	}
	return set, nil
}

// selected drops the generated tests when they are turned off.
func (c *config) selected(files []gen.OutFile) []gen.OutFile {
	if c.tests {
		return files
	}
	out := files[:0:0]
	for _, file := range files {
		if file.Kind == gen.CodeFile {
			out = append(out, file)
		}
	}
	return out
}

// report writes directive diagnostics one per line and returns the error the
// caller sees, which counts them rather than repeating them.
func (c *config) report(err error) error {
	var diagnostics parse.Errors
	if !errors.As(err, &diagnostics) {
		return err
	}
	for _, diagnostic := range diagnostics {
		fmt.Fprintln(c.stderr, diagnostic.Error())
	}
	return fmt.Errorf("vogue: %d %s", len(diagnostics), plural(len(diagnostics), "error", "errors"))
}

// plural picks the singular or the plural form for n.
func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// resolveImportPath returns the configured import path, or asks the go command
// for the one the directory belongs to.
//
// A failure is not fatal: the import path only decides whether a rule pointing
// at a function in the package being generated is called unqualified, so a
// directory outside a module still generates, it just cannot host such a rule.
func (c *config) resolveImportPath() string {
	if c.importPath != "" {
		return c.importPath
	}
	cmd := exec.Command("go", "list", "-f", "{{.ImportPath}}")
	cmd.Dir = c.dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
