// Command vogue generates value objects and their tests from the `//vogue:`
// directives of a Go package.
//
// It is the front end for the built-in rule catalogue. A project that needs
// rules of its own builds its own one-file binary around
// [github.com/MathiasHilgert/vogue/generator.Run] instead; see
// pkg/vogue/examples/customrule for a worked example.
//
// Usage:
//
//	//go:generate go run github.com/MathiasHilgert/vogue/cmd/vogue
//
//	vogue [flags]
//
//	-dir string          directory to generate from (default: $GOFILE's, else ".")
//	-import-path string  import path of that directory (default: asked of `go list`)
//	-tests               generate the test of every value object (default true)
//	-suffix string       appended to a source file's name to name its output (default "_vogue");
//	                     empty writes one <snake_name>.go and <snake_name>_test.go per value object
//	-sql                 generate the database/sql codec, Value and Scan (default true)
//	-schema              generate a JSONSchema method returning a schema.Schema
//	-dry-run             report the files that would be written, write nothing
//	-list                print the rule catalogue and exit
//	-version             print the version and exit
//
// It exits 0 on success, 1 when a directive is rejected, and 2 when it is
// called wrongly.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"text/tabwriter"

	"github.com/MathiasHilgert/vogue/generator"
	"github.com/MathiasHilgert/vogue/rules"
)

// The exit codes the command uses, named so the reason for each one is stated
// where it is returned rather than at the call site.
const (
	exitOK = 0
	// exitFailed is a directive the generator rejected, or a run that could
	// not write what it generated.
	exitFailed = 1
	// exitUsage is the command being called wrongly, which is a mistake in
	// whatever invoked it rather than in the source it was pointed at.
	exitUsage = 2
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is main with its environment passed in, so the command is testable
// without a subprocess. It never panics and never calls os.Exit.
func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("vogue", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dir := flags.String("dir", "", "directory to generate from (default: the directory of $GOFILE, else the working directory)")
	importPath := flags.String("import-path", "", "import path of that directory (default: asked of `go list`)")
	tests := flags.Bool("tests", true, "generate the test of every value object")
	suffix := flags.String("suffix", "_vogue", "appended to a source file's base name to name the files generated from it; empty writes <snake_name>.go and <snake_name>_test.go per value object")
	schema := flags.Bool("schema", false, "generate a JSONSchema method describing each value object for OpenAPI")
	sql := flags.Bool("sql", true, "generate the database/sql codec (Value and Scan); turn it off for a domain package that must not import database/sql/driver")
	dryRun := flags.Bool("dry-run", false, "report the files that would be written, and write nothing")
	list := flags.Bool("list", false, "print the rule catalogue and exit")
	version := flags.Bool("version", false, "print the version and exit")

	if err := flags.Parse(args); err != nil {
		return exitUsage
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "vogue: takes no positional arguments, got %q; name the directory with -dir\n", flags.Arg(0))
		flags.Usage()
		return exitUsage
	}

	switch {
	case *version:
		fmt.Fprintf(stdout, "vogue %s\n", buildVersion())
		return exitOK
	case *list:
		printCatalogue(stdout)
		return exitOK
	}

	opts := []generator.Option{
		generator.WithTests(*tests),
		generator.WithSQL(*sql),
		generator.WithSuffix(*suffix),
		generator.WithSchema(*schema),
		generator.WithDryRun(*dryRun),
		generator.WithStdout(stdout),
		generator.WithStderr(stderr),
	}
	if *dir != "" {
		opts = append(opts, generator.WithDir(*dir))
	}
	if *importPath != "" {
		opts = append(opts, generator.WithImportPath(*importPath))
	}

	if err := generator.Run(opts...); err != nil {
		fmt.Fprintln(stderr, err)
		return exitFailed
	}
	return exitOK
}

// printCatalogue writes the built-in rules as an aligned table: the tag as it
// is written in a directive, the kinds it applies to, its parameter contract
// and the first sentence of its documentation. The full documentation is the
// godoc of the rules package.
func printCatalogue(w io.Writer) {
	table := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(table, "RULE\tKINDS\tPARAM\tDESCRIPTION")
	for _, rule := range rules.All() {
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\n", rule.Name, rule.Kinds, rule.Param, rule.Summary())
	}
	// A tabwriter only fails when the writer under it does, and a command that
	// cannot write to its own standard output has nothing left to report.
	_ = table.Flush()
}

// buildVersion returns the module version the binary was built from, or
// "(devel)" when it was built from a working tree.
func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" {
		return "(devel)"
	}
	return info.Main.Version
}
