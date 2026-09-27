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
// (<Name>, error), accessors, Equal, IsZero, JSON marshalling and
// database/sql Scan/Value, plus a table-driven test file derived from the
// rules' own [Examples].
//
// A nominal type — rather than a generic wrapper — is what gives each value
// object its own methods and makes the zero value meaningful, so the generator
// emits one type per directive.
//
// # The error model
//
// Validation never stops at the first problem. A generated constructor
// collects failures in a [Notification] (Martin Fowler's Notification pattern)
// and returns them as a single error:
//
//	title, err := NewTitle("")
//	// 1 validation error:
//	//   - title: is required (rule "required")
//
// Each failure is a [FieldError] carrying the field, the rule, the rule
// parameter, the offending value and the rendered message. [FieldError.Error]
// renders
//
//	<field>: <message> (rule "<rule>", param "<param>")
//
// and [FieldError.Code] returns the stable `<field>.<rule>` identifier for API
// payloads and translation keys. Because [Notification] implements
// `Unwrap() []error`, callers use the standard library directly:
//
//	if errors.Is(err, vogue.FieldError{Rule: "email"}) { ... }
//
//	var fe vogue.FieldError
//	if errors.As(err, &fe) { log.Println(fe.Code()) }
//
// Both types live in the runtime package
// github.com/MathiasHilgert/vogue/validation, which is the only vogue package
// generated code imports and which depends on the standard library alone;
// the names here are aliases kept for compatibility. Every failure also
// matches the sentinel [ErrInvalid].
//
// The zero [Notification] is ready to use and allocates nothing until the
// first failure, so constructing a value object from valid input allocates
// nothing at all.
//
// # Rules
//
// A rule is a plain value: see [Rule]. It declares the tag [Rule.Name], the
// [KindSet] it applies to, its [ParamSpec], a [Rule.Message] template rendered
// with [MessageData], its documentation, its [Examples], and exactly one of
// [Rule.Emit] (an inline expression) or [Rule.Call] (a static [FuncRef]).
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
//   - `rules/rulecheck` holds the runtime helpers those rules dispatch to
//     through [Rule.Call], such as `rulecheck.Email`: pure `func(string) bool`
//     predicates a generated constructor calls statically.
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
//	//go:generate go run github.com/MathiasHilgert/vogue/cmd/vogue
//
//	generator.Run(generator.WithRules(cuit.Rule))
//
// It lives one package down rather than here because it depends on the parser,
// the code generator and the shipped rules, and all three depend on this
// package for [Rule] and [FieldError]. This package defines what a rule and a
// failure are; that one is the program that uses them.
//
//   - `cmd/vogue` is the shipped command, the built-in catalogue and nothing
//     else, with `-list`, `-dry-run` and `-tests`.
//   - `examples/customrule` is the worked example of a project rule:
//     the rule, the one-file binary that registers it, the directives and the
//     committed output, checked for drift by an ordinary test.
//   - `README.md` holds the directive grammar, the rule catalogue
//     table and the design notes.
package vogue
