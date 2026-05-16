# Security policy

## Reporting a vulnerability

Email <opensource@simtabi.com> with a description of the issue and any
reproduction steps. Please do not open a public issue or a public
discussion for security reports.

We aim to:

- Acknowledge receipt within 72 hours.
- Provide an initial assessment within 7 days.
- Coordinate a fix and disclosure timeline with you, typically within
  90 days of the initial report.

If you do not receive a response within the acknowledgement window,
please follow up — mail delivery is the most common cause of silence.

## Supported versions

| Version | Supported |
|---------|-----------|
| `0.x.y` (current pre-1.0) | Yes — fixes land on `main` and ship in the next minor release |

Pre-1.0 means breaking changes can happen between minor versions; security
fixes are still backported to the most recent release.

## Scope

This policy covers the source code in this repository and the binaries
distributed via:

- GitHub Releases under `simtabi/osaat`
- The `simtabi/homebrew-tap` Homebrew tap formula `osaat`

It does not cover:

- Third-party packages this tool reads (Homebrew, mas, dpkg, rpm, etc.).
  Report issues in those tools upstream.
- Bugs in macOS, Linux, or Unix utilities `osaat` shells out to.

## Disclosure

After a fix ships, we publish a release note in [CHANGELOG.md](CHANGELOG.md)
and, where appropriate, a GitHub Security Advisory. Reporters are
credited unless they ask to remain anonymous.
