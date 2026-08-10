<!--
Copyright (c) Anecdotes AI
SPDX-License-Identifier: MPL-2.0
-->

# Contributing to the Anecdotes Terraform Provider

Thanks for your interest in contributing! This document covers how to build, test, and
submit changes.

## Code of Conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md). By participating, you
agree to uphold it.

## Prerequisites

- [Go](https://go.dev/dl/) (see the version pinned in [`go.mod`](go.mod))
- [Terraform](https://developer.hashicorp.com/terraform/install) >= 1.0
- An Anecdotes account and API key for running acceptance/smoke tests

## Build & test

```bash
make build      # compile the provider
make test       # unit tests, with the race detector
make lint       # golangci-lint
make vulncheck  # govulncheck against known advisories
make fmt        # gofmt + terraform fmt
make docs       # regenerate registry docs (required if you change schemas/examples)
```

Acceptance tests talk to a live Anecdotes tenant and are gated behind `TF_ACC=1`:

```bash
TF_ACC=1 ANECDOTES_API_KEY="your-key" go test -v -timeout 120m ./internal/provider/
```

See [docs/guides/TESTING.md](docs/guides/TESTING.md) and
[docs/guides/LOCAL_DEVELOPMENT.md](docs/guides/LOCAL_DEVELOPMENT.md) for details.

## Documentation

Registry docs under [`docs/`](docs/) are **generated** by `tfplugindocs` from the Go schema
descriptions, [`examples/`](examples/), and [`templates/`](templates/). Do not hand-edit
`docs/` — change the source and run `make docs`. CI fails if generated docs drift.

## Submitting changes

1. Fork the repo and create a topic branch.
2. Make your change, including tests and regenerated docs where relevant.
3. Run `make fmt test lint vulncheck docs` and ensure everything passes.
4. **Never commit secrets** — API keys, tokens, or `*.tfvars` with credentials. Use the
   `ANECDOTES_API_KEY` environment variable for local testing.
5. Open a pull request with a clear description of the change and motivation.

## Reporting bugs & requesting features

Open a [GitHub issue](../../issues) for provider bugs and feature requests. For account or provider help, see [Support](README.md#support). For security vulnerabilities, follow [SECURITY.md](SECURITY.md) instead of filing a public issue.

## License

By contributing, you agree that your contributions will be licensed under the
[Mozilla Public License 2.0](LICENSE), the same license as this project.
