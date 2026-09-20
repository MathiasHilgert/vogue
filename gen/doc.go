// Package gen turns a parsed vogue package into Go source: one file per source
// file, holding one nominal value-object type per directive.
//
// # What is generated
//
// Every value object is a struct with unexported fields, so its zero value is
// inert and only a constructor can produce a valid instance. Each kind gets the
// methods its shape calls for:
//
//	string  New<Name>, String, IsZero, Equal, text and SQL codecs
//	int     New<Name>, Int64,  IsZero, Equal, text and SQL codecs
//	enum    <Name><Member> vars, <Name>Values, Parse<Name>, the same methods
//	id      New<Name> (uuid strategies only), Parse<Name>, IsZero, Equal
//
// Generated code contains no reflection, no maps and no runtime rule lookup:
// every rule is inlined as a static expression or a static function call, and
// every message is rendered at generate time. The output is formatted with
// go/format, so it is gofmt-clean by construction.
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
//     if !(utf8.RuneCountInString(v) >= 1) {
//     n.Add(vogue.FieldError{Field: "title", Rule: "min", ...})
//     }
//
//   - A rule with [vogue.Rule.Normalize] set returns a statement assigning to
//     [vogue.EmitContext.Var] instead, and records no failure. Normalizers are
//     applied in directive order, so every rule written after one sees the
//     rewritten value.
//
//   - A rule with [vogue.Rule.Call] set is emitted as a static call,
//     `pkg.Func(v)` or `pkg.Func(v, "<param>")`, with the same
//     true-means-valid direction, and its package is added to the imports.
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
// An enum is a struct with an unexported field and package-level member
// variables, not a defined string type with constants. That keeps the zero
// value out of the member set: an uninitialised enum is IsZero and matches no
// member, which a constant-backed enum cannot promise. The cost is that the
// exhaustive linter, which only understands constant members, cannot check a
// switch over one. Exhaustiveness is covered by <Name>Values instead: a test
// that ranges over it grows a case the moment a member is added to the
// directive.
//
// # Imports
//
// Imports are collected per file from the kinds generated and from the rules
// used, then emitted sorted. Two import paths whose last element is the same
// identifier are reported as a collision rather than aliased.
package gen
