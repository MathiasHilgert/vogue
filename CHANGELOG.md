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

### Changed

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
