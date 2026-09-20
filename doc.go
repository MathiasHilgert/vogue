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
//   - `pkg/vogue/rules` holds one [Rule] value per built-in tag — the
//     normalizers `trim`, `squish`, `lower` and `upper`, the string checks
//     `required`, `min`, `max`, `len`, `email`, `url`, `uuid`, `regex`,
//     `oneof`, `alpha`, `alphanum`, `numeric`, `ascii`, `printable`,
//     `nospace`, `prefix`, `suffix`, `contains` and `excludes`, and the
//     integer checks `positive`, `nonneg` and `multipleof` — together with
//     `rules.All`, `rules.Set` and `rules.MustSet`.
//   - `pkg/vogue/rules/fn` holds the runtime helpers those rules dispatch to
//     through [Rule.Call], such as `fn.Email`: pure `func(string) bool`
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
package vogue
