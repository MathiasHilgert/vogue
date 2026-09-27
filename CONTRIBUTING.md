# Contributing

## Workflow

- Write tests first (TDD): a failing test, the minimal code to pass it, then
  refactor. Tests use [testify](https://github.com/stretchr/testify)
  (`assert`/`require`) and follow Arrange-Act-Assert, with the three sections
  commented when a test is more than a couple of lines.
- Prefer table-driven tests for anything with more than two cases.
- Run `golangci-lint run ./...` before sending a change; the configuration is
  `.golangci.yml` at the repository root.
- Generated files (`vo_vogue.go`, `vo_vogue_test.go`, and the fixtures under
  `gen/internal/fixture` and `examples/customrule`) are committed. After
  changing a directive, a rule, or the generator itself, regenerate them
  instead of hand-editing:

  ```sh
  go generate ./...
  ```

  Fixture-comparison tests take an `-update` flag to rewrite their golden
  output when the change is intentional, for example:

  ```sh
  go test ./gen/... -update
  go test . -update   # rewrites the README's rule catalogue table
  ```

  CI fails the build if `go generate ./...` produces a diff, so regenerate and
  commit the result before opening a pull request.

- Generated code must pass a strict lint profile with no exclusion for
  generated files. The golden packages under `gen/testdata/strict` are
  rewritten by `go test ./gen/... -update`; their tests and the strict lint
  run like this:

  ```sh
  go test ./gen/testdata/strict/...
  golangci-lint run --config gen/testdata/strict/golangci.yml \
    ./gen/testdata/strict/domain/ ./gen/testdata/strict/persistence/ \
    ./rules/internal/catalogue/ ./examples/customrule/domain/ \
    ./examples/composite/geo/
  ```

## Commit messages

This repository uses [Conventional Commits](https://www.conventionalcommits.org/)
(`feat:`, `fix:`, `docs:`, `test:`, `chore:`, `ci:`, ...). Keep the subject
line under about 70 characters and explain the "why" in the body when it is
not obvious from the diff.

## Adding a rule

A built-in rule lives in `rules/` and is a plain `vogue.Rule` value: a name,
the kinds it applies to, its parameter contract, a message template,
documentation, and examples. Its `Examples` become the generated test for
every value object that uses it, so a rule without examples has no test
coverage of its own. See `rules/rules.go` for the shape and
`examples/customrule` for a project-local rule that is not part of the
built-in catalogue.
