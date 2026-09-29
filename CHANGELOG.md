# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project
adheres to [Semantic Versioning](https://semver.org/). While the major version
is zero, a minor release may break the API; every break is listed under
"Changed" or "Removed" together with how to migrate.

## [Unreleased]

Generated code now imports nothing of vogue: the standard library, the
packages of the kinds it uses (`uuid`, `decimal`) and the failure type of your
own project. Regenerate with `go generate ./...` after following the
migration below. The entries under "Earlier in this release" were written
against the beta.2 runtime and are partly superseded by what follows.

### Changed (breaking)

- **`-validation` is required.** Generated constructors record their failures
  on a type your project owns, named `-validation=<import path>.<Type>`
  (`generator.WithValidation(path, typeName)`, `gen.Options.Validation`). It
  needs a zero value that is ready to use, `Add(field, rule, message string)`
  and `Err() error`, and the error `Err` returns needs
  `Has(field, rule string) bool`. Without the flag the command exits 2 and
  prints the contract with a reference implementation, also in
  `examples/validation`. Migration: copy that type into your module, add the
  flag to your `//go:generate` line and regenerate.
- **`errors.Is(err, validation.ErrInvalid)` and `validation.FieldError` are
  gone**, with `validation.Notification`, `vogue.FieldError`,
  `vogue.Notification` and `vogue.ErrInvalid`. Match on your own type instead:
  `errors.As(err, &yourError)`, or `errors.As` into an
  `interface{ Has(field, rule string) bool }`. An HTTP adapter that mapped
  `ErrInvalid` to a 422 now maps your error type.
- **Messages do not name the field and never carry the value**:
  `invalid slug: is required (required)`, where the field and the rule come
  from the failure and the message is `is required`. The built-in messages of
  `min` and `max` read `length must be at most 120` on a string.
  `vogue.MessageData` gains `Kind` for a rule spanning several kinds.
- **`required` stops the constructor when it fails**
  (`vogue.Rule.Precondition`), so an empty input is one failure and not one per
  rule that also rejects the empty string. Other rules keep accumulating.
- **No MarshalJSON, no UnmarshalJSON.** `encoding/json` uses the text codec,
  so a value object is a JSON string as before, but the zero value no longer
  marshals as `null`: it has no text form and fails. Add `omitzero` to the
  optional fields that hold value objects (`json:"title,omitzero"`). A JSON
  `null` is skipped by the decoder and leaves the field as it was.
- **The zero value fails to marshal with a wrapped `errors.ErrUnsupported`**
  (was `validation.ErrZeroValue`). `encoding/json` does not print the cause of
  an `ErrUnsupported`, only that the marshaler failed for the type.
- **`-sql` is off by default** (`generator.WithSQL` defaults to `false`,
  `gen.Options.OmitSQL` is now `gen.Options.SQL`). Pass `-sql` where
  `Value`/`Scan` are wanted. With it, `Scan` wraps `errors.ErrUnsupported`
  for a source it cannot read, and for a binary float on a decimal;
  `validation.ErrUnsupportedSource` and `validation.ErrLossySource` are gone.
- **`JSONSchema()` returns `map[string]any`**, keyed by the JSON Schema
  keywords `type`, `format`, `pattern`, `enum`, `minLength`, `maxLength`,
  `minimum`, `maximum` and `exclusiveMinimum` that the directive sets, instead
  of a `schema.Schema`. An adapter that translated a `schema.Provider`
  type-asserts on `interface{ JSONSchema() map[string]any }`.
- **Generated tests use the standard library only** (`testing`, `errors`, and
  `regexp`, `unicode/utf8` and `slices` for a schema), in a few small
  functions per value object instead of one `voguetest` suite. Regenerate them.
- **`Rule.Declare` gives way to `Rule.Method`** for checks too long to read
  inline. `Method` returns the name and the body of an unexported method, which
  the generator writes once on every type that uses the rule
  (`func (Code) isEmail(value string) bool`), and the constructor calls through
  its receiver. `email`, `url`, `uuid` and `timezone` are method rules that use
  the standard library only (`net/mail`, `net/url`, a hand-written 8-4-4-4-12
  check, a `switch` over the IANA zone names). `Declare` stays for a
  package-level declaration such as a compiled pattern, and `EmitContext`
  gains `Type` and `Receiver` to name it per type. Migration: a custom rule
  that used `Declare` for anything but a compiled pattern returns a `Method`
  instead, and its `Emit` goes away.
- **`regex` compiles its pattern once**, into one unexported
  `<type>Pattern` variable per type, the only package-level variable generated
  code declares. An invalid pattern still fails at generate time.
- **A `Rule.Call` into vogue is refused at generate time**: into the module
  itself, or into a package that depends on it (`go list -deps`); the
  generated code would import vogue into the domain. Keep the predicate in a
  package that imports nothing of vogue and the `vogue.Rule` value in another,
  as `examples/customrule` now does (`cuit` and `cuitrule`).
- **Generated code is shorter.** One `failures.Add(field, rule, message)` line
  per rule, `Equal` through `==` (a decimal still compares with `Cmp`), and
  one-line doc comments.

### Added

- `regex_message="..."` directive token: replaces the generic message of the
  `regex` rule of the same directive, which prints the pattern.
- `vogue.Rule.Precondition`, `vogue.Rule.Method`, `vogue.EmitContext.Type` and
  `vogue.EmitContext.Receiver`, `vogue.MessageData.Kind`.
- `examples/validation`, the reference failure type, linted with the strict
  profile like the generated code.
- `generator.ParseValidation`, `generator.ValidationContract`.
- A test runs `go list -deps` on the golden packages and fails on an import of
  vogue or of a test framework; the strict lint configuration allows generated
  code the standard library, `uuid`, `decimal` and the failure type only
  (`depguard`).

### Removed

- Packages `validation`, `textjson`, `schema`, `voguetest`, `rules/rulecheck`
  and the aliases of the root package (`vogue.FieldError`,
  `vogue.Notification`, `vogue.ErrInvalid`). Nothing generated imports them.

### Earlier in this release

#### Fixed

- Switching `-suffix` under `go generate` no longer fails the first run with
  "no such file or directory": go generate opens every file it listed before
  running, so a stale generated file is emptied into a `//go:build ignore`
  stub while go generate runs, and removed by the next run.

#### Added

- `-suffix` / `generator.WithSuffix` / `gen.Options.Suffix` name the generated
  files; the default `_vogue` keeps today's names. `-suffix=` writes
  `<snake_name>.go` and `<snake_name>_test.go` per value object
  (`gen.Options.PerValueObject`).
- vogue refuses to overwrite a file that does not start with
  `// Code generated by vogue. DO NOT EDIT.` (`gen.ErrNotGenerated`, exit 1,
  nothing written), and removes the generated files a previous run wrote that
  this run no longer does — only when they carry that header (`gen.Stale`).

- Package `validation`, the runtime generated code depends on. It holds
  `FieldError`, `Notification` and the sentinel errors `ErrInvalid`,
  `ErrUnsupportedSource` and `ErrLossySource`, and imports the standard
  library only.
- `errors.Is(err, validation.ErrInvalid)` matches every validation failure,
  however deeply it was wrapped.
- `Notification.Reject(field, rule, param, value, message)`, the one-call form
  generated constructors record a failure with.
- Generated code passes a strict golangci-lint profile with no exclusion for
  generated files (`gen/testdata/strict/golangci.yml`, a copy of a real
  consumer's configuration with `default: all`). CI lints the golden output,
  the whole rule catalogue and the custom-rule example with it, and runs the
  tests generated into the golden packages.
- `-sql=false` / `generator.WithSQL(false)` / `gen.Options.OmitSQL` leave the
  `database/sql/driver` codec (`Value`, `Scan`) out, for domain packages whose
  linter forbids importing `database/sql/...`.
- `example=<value>` directive tokens declare values the directive accepts; the
  generated test requires every one of them to be accepted and proves the
  round trips with the first.
- Generated code, the runtime packages and `voguetest` spell identifiers out
  (`value`, `notification`, receivers named after their type such as
  `countryCode`, constants such as `lengthParameter`); a test fails on any
  identifier under three characters in the generated goldens except `ok`,
  `id`, `ctx`, `err` (and `t` in generated tests). The rule helper package
  is `rules/rulecheck` (was `rules/fn` in pre-release drafts).
- Package `voguetest`, the suites generated tests run (`Scalar`, `Enum`,
  `UUID`, `Int64ID`).
- `vogue.Rule.Local` and `vogue.EmitContext.Ident`: a rule may declare a
  constant inside the constructor, named after the identifier reserved for it,
  instead of writing a magic number or a package-level variable.
- `rulecheck.Regexp`, which compiles a pattern once and caches it.
- The `timezone` rule: a canonical IANA time zone name (`rulecheck.TimeZone`),
  compared case-exactly against the zones of the Go toolchain's database, so
  `europe/madrid`, `Factory`, `localtime` and `posixrules` are rejected on
  every platform, without reading a zone database at run time.
- Package `textjson`, the JSON string codec the generated JSON methods use.
- `validation.ErrZeroValue`.
- `Notification.Collect(err)`, which folds the failures of a part's error into
  a notification, for composite value objects; `validation.Failure(field,
  rule)`, an exhaustruct-clean `errors.Is` target.
- `-schema` / `generator.WithSchema(true)` / `gen.Options.Schema` add a
  `JSONSchema() schema.Schema` method to every value object. Package `schema`
  (standard library only) describes type, format, pattern, enum members and
  limits, for an HTTP adapter to translate into its OpenAPI schema type; the
  generated test checks that accepted values satisfy it.
- `examples/composite`: a hand-written `Coordinates` composed from generated
  `Latitude` and `Longitude`, with every failure reported at once.

#### Changed

- **Breaking: field names keep Go initialisms.** The name a value object
  reports its failures under is the lower-camel spelling Go uses, so
  `GeoNamesID` reports `geoNamesID` (was `geoNamesId`) and
  `SubdivisionISOCode` reports `subdivisionISOCode` (was
  `subdivisionIsoCode`). A leading initialism is still lower-cased whole:
  `CUITNumber` reports `cuitNumber`. Receivers follow the same spelling.
- The JSON schema is tighter and truthful: the tightest of several bounds
  wins, `required` makes a minimum length of at least 1, `nonneg` combines
  with `min` into the higher minimum, checks written before a normalizer are
  left out (they constrain the input, not the canonical text), integer `oneof`
  items are published in canonical base-10 form, and a `regex` using an
  RE2-only construct is left out instead of being published in the wrong
  dialect. The generated test checks every accepted example against the
  schema and, via the new `voguetest.Rejection.Described`, that rejected rows
  of modeled rules fail it.

- Integer rule parameters are emitted as the base-10 number they parse to, so
  `min=010` compares against 10, not the octal 8; every item of an integer
  `oneof` is validated at generate time.
- Generated tests derive no rejected row for a check that a normalizer runs
  before (`lower oneof=...`, `trim nospace`, ...), since the normalizer may turn
  the example into an accepted value. Strings repeated across the generated
  tests of a package are declared as constants inside each test function, so
  two generated files in one package never collide.
- An enum catalogue's `Parse` switches over its members instead of ranging
  over `All()`, and never allocates.

- **Breaking: the generated API is methods only.** The only package-level
  functions are `New` constructors and there are no package-level variables.
  Regenerate, then migrate callers:

  | 0.1                               | now                                         |
  |-----------------------------------|---------------------------------------------|
  | `Parse<Name>(s)` (int, decimal)   | `New<Name>FromString(s)`                    |
  | `Parse<Name>(s)` (id)             | `New<Name>FromString(s)`                    |
  | `<Name>FromInt64(n)` (int64 id)   | `New<Name>FromInt64(n)`                     |
  | `<Name><Member>` (enum variable)  | `<Plural>{}.<Member>()`                     |
  | `<Name>Values()`                  | `<Plural>{}.All()`                          |
  | `Parse<Name>(s)` (enum)           | `<Plural>{}.Parse(s)`                       |
  | `id.UUID` (embedded field, uuid)  | `id.UUID()`                                 |
  | `MarshalBinary`/`UnmarshalBinary`, `Version`, `URN`, ... promoted from `uuid.UUID` | `id.UUID().MarshalBinary()`, `id.UUID().Version()`, ... |
  | `IsZero()` of a constructed `0`/`""` was true | false: `IsZero` means never constructed |
  | `Value()` of the zero value was `""`/`0`/nil UUID text | `nil` (SQL `NULL`) |
  | `MarshalText()` of the zero value was `""`/`"0"` | error wrapping `validation.ErrZeroValue`; JSON is `null` |
  | `NewXFromString(nilUUID)` accepted  | rejected (`required`), like int64 id `0` |

  `<Plural>` is the regular plural of the enum name: `TabStatus` has
  `TabStatuses`, `PlaceKind` has `PlaceKinds`. `parse.EnumValue.Const` is kept
  for tools; the generator no longer declares it.
- **Breaking: generated tests** are one `Test<Name>` per value object running
  a `voguetest` suite, instead of several spelled-out test functions. The
  sample used for the round trips is picked when the test runs, as the first
  declared example the constructor accepts, which fixes generated tests that
  failed when a sample valid for one rule broke another (`len=2` with
  `regex=...`). A rewrite a normalizer declares is skipped, not failed, when
  the rest of the directive rejects its output.
- **Breaking: the zero value is NULL.** String, int and decimal value objects
  record that their constructor ran, so `IsZero` means "never constructed":
  `NewPopulation(0)` is no longer `IsZero`. `Value()` of the zero value of
  every kind returns `nil` (SQL `NULL`) instead of `""`, `0` or the nil UUID,
  and `Scan(nil)` still produces the zero value. `json:",omitzero"` follows
  `IsZero`.
- **Breaking: the zero value has no text form and is JSON `null`.** Every
  value object generates `MarshalJSON` (`null` for the zero value) and
  `UnmarshalJSON` (`null` reads back as it) through package `textjson`, which
  needs no `encoding/json`; `MarshalText` of the zero value wraps
  `validation.ErrZeroValue`. Before, an unset value marshaled as `""`/`"0"`
  and read back as a constructed one.
- **Breaking: `New<Name>FromString` of a uuid identifier rejects the nil UUID**
  as a failure of `required`, as the int64 kind rejects 0.
- `validation.Collect` walks the branches of an `errors.Join`: validation
  failures are merged and every other branch is returned; an empty
  notification collects nothing.
- **Breaking: uuid identifiers no longer embed `uuid.UUID`.** The value is
  private and read with `UUID()`; `String`, `MarshalText`, `UnmarshalText`,
  `Value` and `Scan` are generated (the SQL pair only without `-sql=false`),
  and `Scan` reads both the text and the 16-byte form of a uuid column.
- The built-in bounds (`min`, `max`, `len`, `multipleof`, `scale`) declare
  their parameter as a constant inside the constructor; decimal bounds are
  built with `decimal.MustNew` from a coefficient and a scale instead of a
  package-level `decimal.MustParse` variable. `regex` calls `rulecheck.Regexp`
  instead of declaring a package-level compiled pattern, and `oneof` on an
  integer is a `slices.Contains` over a literal.

- Generated code imports `github.com/MathiasHilgert/vogue/validation` instead
  of the root `vogue` package, so a binary using generated value objects no
  longer links `text/template` or the rule definitions. Regenerate with
  `go generate ./...`; hand-written code keeps compiling because
  `vogue.FieldError`, `vogue.Notification` and `vogue.ErrInvalid` are aliases
  of the new names.
- A generated `Scan` wraps `validation.ErrUnsupportedSource` (and a decimal
  `Scan` handed a binary float wraps `validation.ErrLossySource`) instead of
  returning an unwrapped `fmt.Errorf` error. The messages still contain
  "cannot scan".

Proposed version for this release: `v0.2.0-beta.1` (breaking changes to the
generated API while the major version is zero).

## [0.1.0-beta.1]

- First public release.

[Unreleased]: https://github.com/MathiasHilgert/vogue/compare/v0.1.0-beta.1...HEAD
[0.1.0-beta.1]: https://github.com/MathiasHilgert/vogue/releases/tag/v0.1.0-beta.1
