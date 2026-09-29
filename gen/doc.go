// Package gen turns a parsed vogue package into Go source: one file per source
// file, holding one nominal value-object type per directive.
//
// # What is generated
//
// Every value object is a struct with unexported fields, so its zero value is
// inert and only a constructor can produce a valid instance. Each kind gets the
// methods its shape calls for:
//
//	string   New<Name>, String, IsZero, Equal, the text codec
//	int      New<Name>, New<Name>FromString, Int64, the same methods
//	decimal  New<Name>, New<Name>FromString, Decimal, the same methods
//	enum     <Plural>{}.<Member>(), <Plural>{}.All(), <Plural>{}.Parse, the same methods
//	id       New<Name> (uuid) or New<Name>FromInt64 (int64), New<Name>FromString
//
// The codec is encoding.TextMarshaler and TextUnmarshaler only, which
// encoding/json uses, so there is no MarshalJSON: the zero value has no text
// form, and a field that may be unset is tagged `json:",omitzero"`. The SQL
// codec, Value and Scan, is added when [Options.SQL] is set, and a JSONSchema
// method returning a map[string]any when [Options.Schema] is.
//
// The generated API is methods only: the only package-level functions are the
// New constructors, and the only package-level variables are the compiled
// patterns of `regex` rules, one unexported `<type>Pattern` per type. An
// enum's members are methods of its catalogue, an empty struct named after the
// plural of the enum, so a package gains no identifier beyond its types and
// their constructors.
//
// Generated code imports the standard library, the packages of the kinds it
// uses (uuid and decimal) and the consumer's failure type named by
// [Options.Validation], and nothing of vogue.
//
// The output is written to pass a strict golangci-lint configuration with no
// exclusion for generated code — gen/testdata/strict/golangci.yml, which CI
// runs against the golden output — so it carries its own constants instead of
// magic numbers, wraps sentinel errors, keeps receivers consistent and leaves a
// blank line before every return that is not alone in its block.
//
// Generated code contains no reflection and no runtime rule lookup: every
// rule is inlined as a static expression, a call to a method of the type or a
// call to a predicate the consumer owns, and every message is rendered at
// generate time. The output is formatted with
// go/format, so it is gofmt-clean by construction.
//
// # Generated tests
//
// Every generated file arrives with the test that proves it, named
// <basename>_vogue_test.go and tagged [TestFile]. The test lives in the
// package under test and uses the standard library only: testing, errors and,
// when a schema is generated, regexp, utf8 and slices. It is a handful of
// small functions per value object, each about one concern — the examples a
// directive declares valid, the rejections its rules declare, the rewrites its
// normalizers declare, the text round trip, the zero value — so two value
// objects of the same kind do not generate the same long function twice.
//
// Its content is derived from [vogue.Examples] and from the directive's own
// `example=` tokens, never invented. Every rejected input becomes a row naming
// each rule that rejects it, which the test checks through the Has method of
// the consumer's error, and every declared [vogue.Normalization] a row of its
// own. A rule declares its examples for itself, so only the constructor knows
// whether a value one rule accepts survives the others: the round trips try
// every declared example and skip the ones the constructor rejects. Strings a
// test repeats often enough for goconst to report are declared once as
// constants.
//
// # The Emit contract
//
// A rule contributes source through [vogue.Rule.Emit], and the direction of the
// boolean matters:
//
//   - An ordinary rule returns an expression that is true when the value is
//     VALID. The generator writes the negation itself, so a rule never has to
//     know how a failure is recorded:
//
//     if utf8.RuneCountInString(value) < minimumParameter {
//     failures.Add("title", "min", "length must be at least 1")
//     }
//
//     The negation is pushed into the expression — a comparison flips its
//     operator, && and || swap under De Morgan — rather than wrapped around
//     it, which staticcheck would report.
//
//   - A rule with [vogue.Rule.Normalize] set returns a statement assigning to
//     [vogue.EmitContext.Var] instead, and records no failure. Normalizers are
//     applied in directive order, so every rule written after one sees the
//     rewritten value.
//
//   - A rule with [vogue.Rule.Call] set is emitted as a static call,
//     `pkg.Func(value)` or `pkg.Func(value, "<param>")`, with the same
//     true-means-valid direction, and its package is added to the imports.
//
//   - A rule with [vogue.Rule.Local] set also contributes a declaration at
//     the top of the constructor, named after [vogue.EmitContext.Ident]:
//     the built-in bounds declare the number they compare against there.
//
//   - A rule with [vogue.Rule.Method] set is written once per generated type
//     as an unexported method, `func (Type) isEmail(value string) bool`, placed
//     after the exported methods, and the constructor calls it through a
//     receiver it declares, `if !email.isEmail(value)`. The standard-library
//     packages the body uses are added to the imports.
//
//   - A rule with [vogue.Rule.Declare] set contributes a package-level
//     declaration, emitted once between the imports and the first value
//     object. The `regex` rule uses it for its compiled pattern, one
//     unexported `<type>Pattern` variable per type, the only package-level
//     variable generated code declares.
//
// # Messages
//
// Failure messages are rendered once, at generate time, with the field name and
// the rule parameter. A message template may therefore not reference
// {{.Value}}: the offending value is only known at run time, and rendering it
// there would mean carrying a template engine into the domain layer. Such a
// template is rejected with a diagnostic naming the rule. The value is still
// available to callers through [vogue.FieldError.Value].
//
// # Enum exhaustiveness
//
// An enum is a struct with an unexported field whose members are returned by
// the methods of its catalogue, not a defined string type with constants. That
// keeps the zero value out of the member set: an uninitialised enum is IsZero
// and matches no member, which a constant-backed enum cannot promise. The cost
// is that the exhaustive linter, which only understands constant members,
// cannot check a switch over one. Exhaustiveness is covered by the
// catalogue's All instead: a test that ranges over it grows a case the moment
// a member is added to the directive.
//
// # Imports
//
// Imports are collected per file from the kinds generated and from the rules
// used, then emitted sorted. Two import paths whose last element is the same
// identifier are reported as a collision rather than aliased.
package gen
