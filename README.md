# vogue

[![Go Reference](https://pkg.go.dev/badge/github.com/MathiasHilgert/vogue.svg)](https://pkg.go.dev/github.com/MathiasHilgert/vogue)
[![Go Report Card](https://goreportcard.com/badge/github.com/MathiasHilgert/vogue)](https://goreportcard.com/report/github.com/MathiasHilgert/vogue)
[![CI](https://github.com/MathiasHilgert/vogue/actions/workflows/ci.yml/badge.svg)](https://github.com/MathiasHilgert/vogue/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/MathiasHilgert/vogue)](LICENSE)
![status: beta](https://img.shields.io/badge/status-beta-yellow)

Turns one-line directives into typed, validated, documented value objects and
their tests. It reads like `go-playground/validator` — rule tags, descriptive
field errors, custom rules — but resolves everything at generate time: the code
it writes has no reflection, no runtime registry and no dynamic dispatch, only
inlined expressions and static calls. A value object arrives with its
constructor, its codecs and the table-driven test that proves it.

## Quick start

```sh
go install github.com/MathiasHilgert/vogue/cmd/vogue@latest
```

```go
// vo.go
//go:generate go run github.com/MathiasHilgert/vogue/cmd/vogue
package tab

// Title is the name of a tab, as the waiter typed it.
//vogue:string Title trim required min=1 max=120

// Covers is the number of guests seated at a tab.
//vogue:int Covers min=1 max=200

// TabStatus is the lifecycle state of a tab.
//vogue:enum TabStatus open,in_progress,closed

// TabID identifies a tab across services.
//vogue:id TabID
```

```sh
go generate ./...
```

```go
title, err := tab.NewTitle("  ")
// err: 1 validation error:
//   - title: title is required (rule "required")

covers, _ := tab.NewCovers(4)
open := tab.TabStatuses{}.Open()
status, _ := tab.TabStatuses{}.Parse("open")
id, _ := tab.NewTabID()
```

## Usage

```go
//go:generate go run github.com/MathiasHilgert/vogue/cmd/vogue
package tab

// Title is the name of a tab, as the waiter typed it.
//vogue:string Title trim required min=1 max=120

// Covers is the number of guests seated at a tab.
//vogue:int Covers min=1 max=200

// TabStatus is the lifecycle state of a tab.
//vogue:enum TabStatus open,in_progress,closed

// TabID identifies a tab across services.
//vogue:id TabID
```

`go generate ./...` writes `vo_vogue.go` and `vo_vogue_test.go` next to it.
Generated files carry the `// Code generated` header and are never read back as
input, so a second run is a no-op.

The command takes `-dir`, `-import-path`, `-tests`, `-sql`, `-dry-run`,
`-list` (the catalogue below) and `-version`. It exits 1 on a rejected
directive and 2 when it is called wrongly.

`-sql=false` (`generator.WithSQL(false)`) leaves the `database/sql/driver`
codec — `Value` and `Scan` — out of the generated code. That is what a
hexagonal domain package needs when its linter forbids importing
`database/sql/...` there: the persistence adapter converts through `String`,
`Int64` or `Decimal` and the `New` constructors instead.

## The directive grammar

```
//vogue:<kind> <Name> [<token> ...]
```

`<Name>` is an exported Go identifier; the generated type takes it, and the
field errors take its lower-camel form. The tokens depend on the kind:

| Kind | Tokens | Generated |
|------|--------|-----------|
| `string` | rules and examples | `New<Name>(string)`, `String`, `Equal`, `IsZero`, text and SQL codecs |
| `int` | rules and examples | `New<Name>(int64)`, `New<Name>FromString`, `Int64`, the same methods |
| `decimal` | rules and examples | `New<Name>(decimal.Decimal)`, `New<Name>FromString`, `Decimal`, the same methods |
| `enum` | one comma-separated list of at least two lower-snake values | the catalogue `<Plural>` with `<Member>()`, `All()` and `Parse`, the same methods |
| `id` | an optional strategy: `uuid7` (default), `uuid4`, `int64` | `New<Name>()` (uuid) or `New<Name>FromInt64` (int64), `New<Name>FromString`, the same methods |

The generated API is methods only. The only package-level functions are the
`New` constructors, and there are no package-level variables: parsing text is a
`New<Name>FromString` constructor, and the members of an enum are methods of
its catalogue, an empty struct named after the plural of the enum:

```go
kind := place.PlaceKinds{}.Country()
all := place.PlaceKinds{}.All()
parsed, err := place.PlaceKinds{}.Parse("city")
```

The plural is regular — `-es` after a sibilant, `-ies` after a consonant and
`y`, `-s` otherwise — and a catalogue that would take the name of another value
object, or a member that would shadow `All` or `Parse`, is a generate-time
error.

A rule is written `name` or `name=param`. Rules are evaluated in the order they
are written, which is why a normalizer goes first: every rule after one sees the
rewritten value, so `trim required` rejects a value that was nothing but
whitespace while `required trim` accepts it.

`example=<value>` is not a rule: it declares a value the directive accepts,
and may be repeated. The generated test requires the constructor to accept
every declared example and proves its round trips with the first. It is how a
directive whose rules declare no example that survives all of them still gets
a tested sample:

```go
//vogue:string CountryCode trim upper required len=2 regex=^[A-Z]{2}$ example=AR
```

An unknown rule, a rule applied to the wrong kind, a malformed parameter, a
duplicate name and a misspelled id strategy are all generate-time errors, each
reported with its `file:line:column` and, where there is one, the nearest known
name.

## The rule catalogue

<!-- rules:start -->
| Rule | Kinds | Param | Description |
|------|-------|-------|-------------|
| `trim` | string | none | Removes leading and trailing whitespace, in the Unicode sense: spaces, tabs, newlines and the exotic blanks a paste from a word processor carries. |
| `squish` | string | none | Collapses every run of whitespace inside the value to one space and removes the whitespace around it, so " Tortilla de patatas " becomes "Tortilla de patatas". |
| `lower` | string | none | Folds the value to lower case using the Unicode mapping, so "Í" becomes "í". |
| `upper` | string | none | Folds the value to upper case using the Unicode mapping. |
| `required` | string | none | Rejects the empty string. |
| `min` | string, int, decimal | required number | Rejects values below the bound. |
| `max` | string, int, decimal | required number | Rejects values above the bound, counting runes on a string and the value itself on an integer or a decimal, inclusive on every kind. |
| `len` | string | required int | Requires the value to be exactly the given number of runes long. |
| `email` | string | none | Requires a single email address in the RFC 5322 grammar, parsed by net/mail rather than matched against a regular expression. |
| `url` | string | none | Requires an absolute http or https URL, parsed by net/url as a request URI. |
| `uuid` | string | none | Requires a UUID written in the canonical RFC 4122 form, 8-4-4-4-12 hexadecimal digits separated by hyphens, in either case. |
| `timezone` | string | none | Requires the name of a zone in the IANA time zone database, such as "America/Argentina/Buenos_Aires" or "UTC", resolved with time.LoadLocation. |
| `regex` | string | required regex | Requires the value to match the given RE2 pattern. |
| `oneof` | string, int | required list | Restricts the value to one of the comma-separated items of the parameter, compared for exact equality: on a string it is case-sensitive, so pair it with `lower` or `upper` when the input is typed by a human, and on an integer every item must itself be an integer. |
| `alpha` | string | none | Requires every rune of the value to be a letter, in the Unicode sense: "Muñoz" passes and so does a name in Greek or Cyrillic, while a digit, a space, a hyphen or an apostrophe does not. |
| `alphanum` | string | none | Requires every rune of the value to be a letter or a digit, in the Unicode sense. |
| `numeric` | string | none | Requires every rune of the value to be an ASCII digit, 0 to 9. |
| `ascii` | string | none | Requires every rune of the value to be ASCII, below U+0080. |
| `printable` | string | none | Requires every rune of the value to be printable as Go defines it: letters, marks, numbers, punctuation, symbols and the ASCII space. |
| `nospace` | string | none | Rejects any whitespace anywhere in the value, in the Unicode sense: spaces, tabs, newlines and the exotic blanks a paste carries. |
| `prefix` | string | required string | Requires the value to start with the parameter, compared byte for byte and therefore case-sensitively. |
| `suffix` | string | required string | Requires the value to end with the parameter, compared byte for byte and therefore case-sensitively. |
| `contains` | string | required string | Requires the parameter to appear somewhere in the value, compared byte for byte and therefore case-sensitively. |
| `excludes` | string | required string | Rejects the value when the parameter appears anywhere in it, compared byte for byte and therefore case-sensitively — which is exactly why it is a weak guard: a denylist is defeated by a change of case or an encoding. |
| `positive` | int, decimal | none | Requires the value to be strictly greater than zero. |
| `nonneg` | int, decimal | none | Requires the value to be zero or greater. |
| `multipleof` | int | required int | Requires the value to be an exact multiple of the parameter, which is how a quantity sold by the box, a duration measured in whole slots or an amount in whole units is expressed. |
| `scale` | decimal | required int | Requires the value to carry at most the given number of decimal places, which is how a rate stored in a numeric(p,s) column, a unit price quoted to the cent or a weight measured to the gram is expressed. |
| `nonzero` | decimal | none | Rejects zero, at any scale: "0", "0.00" and "-0.0" are all the same number and all rejected. |
<!-- rules:end -->

The full documentation of each rule is the godoc of
[`rules`](rules). `vogue -list` prints the same table in a terminal.

## Your own rules

There is no runtime registration. A project that needs a rule of its own writes
it, builds a one-file generator around it and points `//go:generate` at that
binary:

```go
func main() {
	if err := generator.Run(generator.WithRules(cuit.Rule)); err != nil {
		log.Fatal(err)
	}
}
```

A rule is a plain value: a name, the kinds it applies to, its parameter
contract, a message template, documentation, examples, and exactly one of
`Emit` (an inline expression that is true when the value is valid) or `Call` (a
static function the generated code calls). Its `Examples` become the generated
test, so a rule that documents itself tests every value object that uses it.

The worked example is [`examples/customrule`](examples/customrule): the
Argentine CUIT, its generator, its directives and its committed output, checked
for drift by an ordinary test.

Extra rules are added on top of the catalogue; one whose name is already taken
is refused rather than shadowing a built-in. `generator.WithoutBuiltins()`
replaces the catalogue instead of extending it.

## The error model

Validation never stops at the first problem. A constructor collects failures in
a `Notification` — Fowler's pattern — and returns them as one error:

```go
title, err := NewTitle("")
// 1 validation error:
//   - title: title is required (rule "required")
```

Generated code imports one runtime package,
[`validation`](validation), and nothing else of vogue. It depends on the
standard library only, so a domain package built from generated value objects
does not link the generator, its templates or its rule catalogue.

Each failure is a `validation.FieldError{Field, Rule, Param, Value, Message}`.
`Error()` renders `<field>: <message> (rule "<rule>", param "<param>")`, and
`Code()` returns the stable `<field>.<rule>` identifier for payloads and
translation keys. `Notification` implements `Unwrap() []error`, so the
standard library is the whole API:

```go
if errors.Is(err, validation.ErrInvalid) { ... }                   // any rule failed
if errors.Is(err, validation.FieldError{Rule: "email"}) { ... }  // this one did
if errors.Is(err, validation.Failure("email", "email")) { ... } // the same, exhaustruct-clean

var fe validation.FieldError
if errors.As(err, &fe) { log.Println(fe.Code()) }
```

`validation.ErrInvalid` is matched by every failure however it was wrapped,
which is what an HTTP adapter maps to a 422. A generated `Scan` handed a
source it cannot read wraps `validation.ErrUnsupportedSource`, and a decimal
handed a binary float wraps `validation.ErrLossySource`; neither is an
`ErrInvalid`, because they describe a misconfigured driver rather than a value
that broke a rule.

`vogue.FieldError`, `vogue.Notification` and `vogue.ErrInvalid` remain as
aliases of the `validation` names, so code written against them keeps
compiling.

Mapping that to RFC 9457 `application/problem+json` is a loop, not a framework:

```json
{
  "type": "https://example.com/problems/validation",
  "title": "The request body is not valid",
  "status": 422,
  "errors": [
    { "code": "title.required", "field": "title", "detail": "title is required" },
    { "code": "covers.min",     "field": "covers", "detail": "covers must be at least 1" }
  ]
}
```

`Value` is carried on the error but never rendered by `Error()`, so an
offending value never reaches a log line by accident; a handler that wants it
has to ask.

## Design notes

**No reflection.** Every rule resolves at generate time into an expression or a
static call, and every message is rendered into a string literal. That is why a
message template may not reference `{{.Value}}`: the value only exists at run
time, and rendering it there would mean carrying a template engine into the
domain layer.

**Zero-value safety.** Every value object has unexported fields, so only a
constructor can produce a valid one and the zero value is inert and `IsZero`.
Nominal types — one per directive, rather than a generic wrapper — are what make
that possible and what give each value object its own methods.

**The zero value is NULL.** `IsZero` means "never constructed", not "holds the
zero of its type": a string, int or decimal value object records that its
constructor ran, so `NewPopulation(0)` and `NewNickname("")` (when the rules
accept them) are not `IsZero`, while `var p Population` is. That one notion
drives every boundary:

- `Value()` of the zero value returns `nil`, which `database/sql` writes as
  SQL `NULL`; any constructed value, `0` and `""` included, is written as
  itself. An unassigned identifier and the zero enum are `NULL` too.
- `Scan(nil)` — a `NULL` column — produces the zero value.
- `json:",omitzero"` asks `IsZero`, so an optional field of a value-object type
  is left out of a payload exactly when it was never set.

A nullable column is therefore a plain value-object field, not a pointer to
one.

**Enums are structs.** An enum is a struct with an unexported field whose
members are returned by the methods of its catalogue, not a defined string type
with constants, so the zero value is outside the member set. The cost is that
the `exhaustive` linter cannot check a switch over one; the catalogue's `All()`
covers that instead, since a test ranging over it grows a case the moment a
member is added.

**Strict lint, no exclusions.** Generated code is written to pass a strict
golangci-lint configuration with nothing excluded for generated files:
[`gen/testdata/strict/golangci.yml`](gen/testdata/strict/golangci.yml) is a
real consumer's configuration with `default: all`, and CI lints the golden
output, the whole rule catalogue and the custom-rule example with it. Bounds
are named constants local to the constructor (`const maximumParameter = 120`), a
failure is recorded with `Notification.Reject`, the negated check is written
without a `!(...)` wrapper, `Scan` wraps a sentinel error, and a regular
expression is compiled once by `rulecheck.Regexp` rather than held in a package-level
variable. The one directive the output carries is a `//nolint:recvcheck` on a
type with `Scan`, which needs a pointer receiver while every other method keeps
a value receiver; with `-sql=false` there is no `Scan` and no directive.

**Counting runes.** Every length bound counts runes, not bytes: `min=3` accepts
`añó`. That is what a person filling in a form counts. A column declared
`varchar(n)` needs a bound on the encoded byte length, which is a different
question and not this one.

**No float.** There is no `float` kind and there will not be one: binary floats
are unacceptable for money-adjacent values. `Money` and `Quantity` are
multi-field and stay hand-written.

**Decimals.** The `decimal` kind wraps a
[`github.com/govalues/decimal`](https://github.com/govalues/decimal) value: an
exact base-10 number with a sign, a coefficient and a scale, and no binary
float underneath. It is the kind for a rate, a percentage or a quantity
measured in fractional units.

Because a decimal carries its scale, `1.5` and `1.50` are the same number
written two ways. `Equal` therefore compares with `Cmp`, not with `==`, while
`String`, `MarshalText` and `Value` keep the scale the value was created with —
so a `numeric(10,4)` column reads back exactly as it was stored. `Value` writes
the canonical text, which PostgreSQL accepts for a `numeric` column and which
no float parameter could carry losslessly.

`Scan` takes text, bytes and `int64`, and refuses `float64` and `float32`
outright: a float source has already lost digits by the time it arrives, and
converting it would undo the reason the kind exists. With pgx that means the
numeric codec must deliver text; a `pgtype.Numeric` source needs a helper of
its own, which belongs with the bases rather than in generated code.

A bound is written as a decimal literal — `min=0 max=1` — validated at
generate time and declared inside the constructor as its coefficient and scale,
which `decimal.MustNew` turns into a value without parsing text, so the
constructor compares and never parses. `min=0.5` is a rate and a generate-time
error on an `int`. `scale=N` rejects a value carrying more decimal places than
`N`; it never rounds one, because how to round is a decision the domain makes
and not the constructor.

**Composite value objects.** vogue generates single-field value objects. A
value made of several — a point, an amount with its currency, a range — is
composed from generated ones by hand, and stays a value object by keeping the
same rules: unexported fields, one constructor that validates every part, and
every failure reported at once. `Notification.Collect` folds each part's error
into one notification, so the pattern is short:

```go
func NewCoordinates(latitude, longitude string) (Coordinates, error) {
	var n validation.Notification

	lat, latErr := NewLatitudeFromString(latitude)
	lon, lonErr := NewLongitudeFromString(longitude)
	for _, err := range []error{latErr, lonErr} {
		if unexpected := n.Collect(err); unexpected != nil {
			return Coordinates{}, unexpected
		}
	}
	if n.HasErrors() {
		return Coordinates{}, n.ErrOrNil()
	}
	return Coordinates{latitude: lat, longitude: lon}, nil
}
```

[`examples/composite`](examples/composite) is the complete example, with its
text codec and test, written to the same strict lint profile as the generated
code.

**Time zones.** The `timezone` rule resolves a name with `time.LoadLocation`,
so it is as current as the zone database the process can read. A binary that
runs in a minimal container image has none of its own and should import
`time/tzdata` to embed one.

**Identifiers.** `uuid7` is the default because time-ordered identifiers keep
an index from fragmenting; `uuid4` is there for values that must not leak a
creation time; `int64` wraps a database-assigned sequence and therefore has no
generator, only `New<Name>FromInt64` and `New<Name>FromString`.
`New<Name>FromString` accepts any RFC 4122 UUID, so existing data can be read.
A uuid identifier holds its `uuid.UUID` privately and exposes it through
`UUID()`, so it cannot be overwritten after construction. v1, v3 and v5 are deliberately
unsupported.

**Generated tests are derived, never invented.** A row exists because a rule
or the directive declared the example. The generated test states only what is
particular to its value object and runs a suite from
[`voguetest`](voguetest), so two value objects of the same kind do not generate
the same assertions twice. The sample the round trips use is chosen when the
test runs, as the first declared example the constructor accepts: a rule
declares its examples for itself, and `len=2` says nothing about the value
`required` declared valid, so only the constructor can tell. When none is
accepted, the round trips skip with a message asking for an `example=`; a
normalizer's rewrite is skipped, not failed, when the rest of the directive
rejects the rewritten value.

## Status

vogue is beta. The generator, the runtime and the rule catalogue are exercised
by the test suite and used as intended, but the API may still change before a
1.0 release — directive syntax, generated method names and the `Rule` contract
are the most likely places.

Not there yet:

- Multi-field value objects (`Money`, `Quantity`, `Coordinates`) stay
  hand-written, composed from generated ones as described above; vogue only
  generates single-field kinds.
- Message translation: `FieldError.Message` is English only, and there is no
  hook to localize it.
- `pgtype.Numeric` scanning for the `decimal` kind; `Scan` accepts text, bytes
  and `int64` but not the pgx-native numeric type.
