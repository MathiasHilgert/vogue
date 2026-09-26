# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project
adheres to [Semantic Versioning](https://semver.org/). While the major version
is zero, a minor release may break the API; every break is listed under
"Changed" or "Removed" together with how to migrate.

## [Unreleased]

### Added

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

### Changed

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

## [0.1.0-beta.1]

- First public release.

[Unreleased]: https://github.com/MathiasHilgert/vogue/compare/v0.1.0-beta.1...HEAD
[0.1.0-beta.1]: https://github.com/MathiasHilgert/vogue/releases/tag/v0.1.0-beta.1
