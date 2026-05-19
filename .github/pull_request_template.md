<!-- Thanks for contributing to errx. See CONTRIBUTING.md. -->

## What

<!-- One sentence: what does this change do? -->

## Why

<!-- The motivation / issue this addresses. Closes #... -->

## Checklist

- [ ] `gofmt -l -s .` prints nothing
- [ ] `go vet ./...` passes
- [ ] `go test ./... -race -count=1` passes (core + any changed contrib module)
- [ ] `golangci-lint run ./...` passes
- [ ] Core module gained **no** new third-party dependency
- [ ] Tests added/updated (redaction assertion if a new output path)
- [ ] `go generate ./...` run and tree is clean (if `cmd/errxgen` or its example changed)
- [ ] `CHANGELOG.md` updated under **Unreleased**
- [ ] Relevant `README.md` updated (root and/or contrib)
- [ ] No version bump / tag in this PR
