# Contributing to errx

Thanks for your interest in improving **errx**. This document explains the
repository layout, local setup, and the checks every change must pass.

## Code of conduct

This project follows the [Contributor Covenant](./CODE_OF_CONDUCT.md). By
participating you agree to uphold it.

## Repository layout

errx is a **monorepo of independent Go modules**:

| Path | Module | Dependencies |
|---|---|---|
| `.` | `github.com/ubgo/errx` | **standard library only** |
| `cmd/errxgen` | (part of core) | stdlib only |
| `diag/`, `result/`, `errtest/` | (part of core) | stdlib only |
| `contrib/<name>/` | `github.com/ubgo/errx/contrib/<name>` | one third-party integration each |

Hard rules:

- **The core module must not gain a third-party dependency.** Anything that
  needs one belongs in a `contrib/*` module.
- **No `init()` side effects.** All registries (sinks, context extractors,
  doc/status maps, code migrations) are populated by explicit calls.
- **Adapters consume only `errx.Report`** via `Snapshot`; they must not
  reach into `*errx.Error` internals, and the core must not import an
  adapter.
- **Redaction is sacred.** Unsafe fields must never reach a sink, wire
  format, or log. If you add an output path, add a test asserting a
  planted secret value is absent from the output.

## Local setup

Requires **Go 1.24+**. A local `go.work` makes cross-module work easy (it
is git-ignored — CI builds each module standalone):

```sh
go work init .
go work use ./contrib/httpx ./contrib/grpc ./contrib/connect \
            ./contrib/graphql ./contrib/sentry ./contrib/otel \
            ./contrib/prometheus ./contrib/goerr
```

[Task](https://taskfile.dev) targets are available in the core module:

```sh
task            # list tasks
task check      # fmt + vet + race tests (run before every PR)
task test       # go test ./... -race
task lint       # golangci-lint
```

## Checks every change must pass

Run from the module you changed (and the core module if you touched it):

```sh
gofmt -l -s .            # must print nothing
go vet ./...
go test ./... -race -count=1
golangci-lint run ./...  # config: .golangci.yml
```

If you changed anything under `cmd/errxgen` **or** the generator's example,
regenerate and ensure the tree is clean (CI enforces this):

```sh
go generate ./...
git diff --exit-code        # generated code must be committed and fresh
```

Each module owns its own CI workflow under `.github/workflows/`
(`ci.yml` for core, `ci-contrib-<name>.yml` per adapter). A new `contrib`
module must add a matching workflow.

## Pull requests

- One logical change per PR; keep the core dependency-free.
- Add or update tests. New behavior without a test will not be merged.
- Update `CHANGELOG.md` under **Unreleased**.
- Update the relevant `README.md` (root and/or the contrib module) so the
  docs never drift from the code.
- Use clear, imperative commit subjects (e.g. `feat: add X`,
  `fix: Y`, `docs: Z`). Conventional Commits are encouraged but not
  enforced.
- Do not bump versions or create tags in a PR; releases are tagged
  separately by maintainers.

## Adding a new contrib adapter

1. `contrib/<name>/go.mod` with `module github.com/ubgo/errx/contrib/<name>`,
   `require github.com/ubgo/errx`, and a local `replace` for dev.
2. Implement against `errx.Report` only. Prefer the `errx.Sink` interface
   for "report to X" integrations; provide a plain function for "encode as
   X" transports.
3. Add `<name>_test.go` with a redaction assertion.
4. Add `contrib/<name>/README.md` following the existing template
   (Why → Install → Step by step → Example → API → Safety).
5. Add `.github/workflows/ci-contrib-<name>.yml`.
6. Link it from the root README integration table.

## Reporting bugs / security

- Functional bugs: open a [GitHub issue](https://github.com/ubgo/errx/issues)
  using the template.
- Security vulnerabilities: **do not** open a public issue — see
  [`SECURITY.md`](./SECURITY.md).

## License

By contributing you agree your contributions are licensed under the
project's [Apache-2.0](./LICENSE) license.
