# Security Policy

## Supported versions

errx is pre-1.0. Security fixes are applied to the latest `main` and the
most recent tagged release (once releases begin). Pin a specific version
and upgrade promptly while the API stabilizes.

## Reporting a vulnerability

**Please do not open a public GitHub issue for security vulnerabilities.**

Report privately via GitHub's
[Private vulnerability reporting](https://github.com/ubgo/errx/security/advisories/new)
("Report a vulnerability" under the repository's **Security** tab).

Please include:

- affected module(s) and version/commit,
- a description and impact assessment,
- reproduction steps or a proof of concept,
- any suggested remediation.

You can expect an acknowledgement within a few business days and a
remediation plan once the report is triaged. Please allow a reasonable
disclosure window before any public discussion.

## Scope and hardening notes

errx is an error-handling library; its most security-relevant guarantee is
**field redaction at trust boundaries**:

- Fields added with `With(...)` are **unsafe** and are replaced with a
  redaction marker by `errx.Snapshot` before any sink, wire encoder, or
  log handler sees them.
- Only fields added with `WithSafe(...)` cross a boundary.
- The operator message (`Error()`) is for logs/operators; only the
  end-user message (`WithPublic` / `WithLocalized`) is intended for
  clients. Transport adapters surface the public message, never the
  operator detail.

Reports of redaction bypasses, accidental leakage of unsafe fields through
any `contrib/*` adapter or the `codec` wire format, or panics reachable
from untrusted input are explicitly in scope and prioritized.
