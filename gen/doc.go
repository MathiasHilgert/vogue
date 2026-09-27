// Package gen turns a parsed vogue package into Go source: one file per source
// file, holding one nominal value-object type per directive.
//
// # What is generated
//
// Every value object is a struct with unexported fields, so its zero value is
// inert and only a constructor can produce a valid instance. Each kind gets the
// methods its shape calls for:
//
//	string   New<Name>, String, IsZero, Equal, text and SQL codecs
//	int      New<Name>, New<Name>FromString, Int64, the same methods
//	decimal  New<Name>, New<Name>FromString, Decimal, the same methods
//	enum     <Plural>{}.<Member>(), <Plural>{}.All(), <Plural>{}.Parse, the same methods
//	id       New<Name> (uuid) or New<Name>FromInt64 (int64), New<Name>FromString
//
// The generated API is methods only: the only package-level functions are the
// New constructors, and there are no package-level variables. An enum's
// members are methods of its catalogue, an empty struct named after the plural
// of the enum, so a package gains no identifier beyond its types and their
// constructors. The SQL codec, Value and Scan, is left out when
// [Options.OmitSQL] is set.
//
// The output is written to pass a strict golangci-lint configuration with no
// exclusion for generated code — gen/testdata/strict/golangci.yml, which CI
// runs against the golden output — so it carries its own constants instead of
// magic numbers, wraps sentinel errors, keeps receivers consistent and leaves a
// blank line before every return that is not alone in its block.
//
// Generated code contains no reflection, no maps and no runtime rule lookup:
// every rule is inlined as a static expression or a static function call, and
// every message is rendered at generate time. The output is formatted with
// go/format, so it is gofmt-clean by construction.
//
// # Generated tests
//
// Every generated file arrives with the test that proves it, named
// <basename>_vogue_test.go and tagged [TestFile]. The test lives in the
// package under test and states only what is particular to each value object —
// its constructor and the examples its directive and rules declare — and hands
// them to a suite of package voguetest, which runs every check as a parallel
// subtest. Keeping the assertions in one place is what keeps two value objects
// of the same kind from generating the same fifty lines twice.
//
// Its content is derived from [vogue.Examples] and from the directive's own
// `example=` tokens, never invented. Every rejected input becomes a row naming
// each rule that rejects it, and every declared [vogue.Normalization] a row of
// its own. The sample the round trips are proven with is chosen when the test
// runs, as the first declared example the constructor accepts: a rule declares
// its examples for itself, so only the constructor knows whether a value one
// rule accepts survives the others. A directive's own examples come first and
// must be accepted. Strings a test repeats often enough for goconst to report
// are declared once as constants.
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
//     notification.Reject("title", "min", "1", value, "title must be at least 1")
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
//   - A rule with [vogue.Rule.Declare] set contributes a package-level
//     declaration, emitted once between the imports and the first value
//     object. The built-in rules no longer use it, because a package-level
//     variable is what gochecknoglobals reports; it remains for custom rules.
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
