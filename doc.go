// Package vogue turns one-line directives into fully typed, validated,
// documented value objects and their tests.
//
// It reads like go-playground/validator v10 — rule tags, descriptive field
// errors, custom rules — but resolves everything at generate time. Generated
// code contains no reflection, no runtime registry and no dynamic dispatch:
// every rule becomes either an inlined expression or a static function call.
//
// # Directives
//
// A value object is declared by a comment directive in an ordinary Go file:
//
//	//vogue:string Title required trim min=1 max=120
//	//vogue:int Age min=0 max=150
//	//vogue:enum Status oneof=draft published archived
//	//vogue:id OrderID
//
// Running the generator over that package produces, for each directive, a
// nominal type with an unexported field, a New<Name> constructor returning
// (<Name>, error), accessors, Equal, IsZero, a text codec (which
// encoding/json uses) and, on request, database/sql Scan/Value, plus a
// table-driven test file derived from the rules' own [Examples].
//
// A nominal type — rather than a generic wrapper — is what gives each value
// object its own methods and makes the zero value meaningful, so the generator
// emits one type per directive.
//
// # The error model
//
// Validation never stops at the first problem, and the type that collects the
// failures is yours. Generated constructors record each failure on a
// [Validation]-shaped type your project owns, named with `-validation`, and
// return its error:
//
//	title, err := NewTitle("")
//	// invalid title: is required (required)
//
// The type needs a zero value that is ready to use and two methods,
// Add(field, rule, message string) and Err() error, where Err is nil until
// something was added and the error it returns has Has(field, rule string)
// bool, which the generated tests use. examples/validation is a reference
// implementation to copy. Generated code therefore imports the standard
// library, the packages of the kinds it uses (uuid, decimal) and that one
// type, and nothing of vogue.
//
// A message does not name the field, because the failure already carries it,
// and never carries the offending value. The `required` rule returns at once
// when it fails, so an empty input is one failure and not one per rule.
//
// # Rules
//
// A rule is a plain value: see [Rule]. It declares the tag [Rule.Name], the
// [KindSet] it applies to, its [ParamSpec], a [Rule.Message] template rendered
// with [MessageData], its documentation, its [Examples], and exactly one of
// [Rule.Emit] (an inline expression), [Rule.Call] (a static [FuncRef] into a
// package the consumer owns, which may not depend on vogue) or [Rule.Method]
// (an unexported method the generator writes on each type that uses the rule).
//
// The rules vogue ships with live in their own package, so the catalogue can
// be read, extended or replaced wholesale:
//
//   - `rules` holds one [Rule] value per built-in tag — the
//     normalizers `trim`, `squish`, `lower` and `upper`, the string checks
//     `required`, `min`, `max`, `len`, `email`, `url`, `uuid`, `regex`,
//     `oneof`, `alpha`, `alphanum`, `numeric`, `ascii`, `printable`,
//     `nospace`, `prefix`, `suffix`, `contains` and `excludes`, and the
//     integer checks `positive`, `nonneg` and `multipleof` — together with
//     `rules.All`, `rules.Set` and `rules.MustSet`.
//     The checks too long to read inline — `email`, `url`, `uuid` and
//     `timezone` — are [Rule.Method] rules that use the standard library only,
//     so generated code imports no helper package of vogue for them.
//
// Rules live in a [RuleSet], which is ordered and name-unique and provides
// [RuleSet.Suggest] so an unknown tag is reported with a file:line position and
// the nearest known rule name.
//
// # Extension
//
// There is no runtime registration. A project that needs its own rules builds
// its own generator binary — `tools/vogue/main.go` — passing the extra rules to
// the generator alongside the built-in catalogue. That keeps the rule set
// statically known, which is what lets the generator reject an unknown rule,
// a rule applied to the wrong kind, or a malformed parameter before any code
// is written.
//
// # Running the generator
//
// The pipeline itself — resolve the catalogue, parse a directory, render, write
// — is `generator`, whose Run is what both front ends call:
//
//	//go:generate go run github.com/MathiasHilgert/vogue/cmd/vogue -validation=example.com/app/fault.Validation
//
//	generator.Run(
//		generator.WithValidation("example.com/app/fault", "Validation"),
//		generator.WithRules(cuitrule.Rule),
//	)
//
// It lives one package down rather than here because it depends on the parser,
// the code generator and the shipped rules, and all three depend on this
// package for [Rule]. This package defines what a rule is; that one is the
// program that uses it.
//
//   - `cmd/vogue` is the shipped command, the built-in catalogue and nothing
//     else, with `-list`, `-dry-run` and `-tests`.
//   - `examples/customrule` is the worked example of a project rule:
//     the rule, the one-file binary that registers it, the directives and the
//     committed output, checked for drift by an ordinary test.
//   - `README.md` holds the directive grammar, the rule catalogue
//     table and the design notes.
package vogue
