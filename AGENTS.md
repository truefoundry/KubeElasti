# AGENTS.md

Guidance for AI agents working in the KubeElasti repository.

## What this is

KubeElasti is a Kubernetes-native controller that provides scale-to-zero: it scales a
service to zero when there is no traffic and back up to one when traffic arrives. Two
components:

- **operator** (`operator/`) - the controller/manager. Watches the `ElastiService` CRD,
  manages scaling, and switches services between "proxy" (scaled to zero, resolver in front)
  and "serve" (running) modes.
- **resolver** (`resolver/`) - the request proxy. Holds incoming requests while a scaled-to-zero
  target spins up, then forwards them; signals the operator to scale from zero.
- **pkg** (`pkg/`) - shared libraries used by both (k8shelper, scaling, logger, values, etc).

These are three separate Go modules stitched together by a `go.work` (Go 1.26.5).

## Build, test, lint

Run from the repo root:

- `make test` - all tests (operator + resolver + pkg). Per-component: `make test-operator`,
  `make test-resolver`, `make test-pkg`.
- Per module you can also `cd operator && make <target>`: `build`, `lint`, `lint-fix`,
  `fmt`, `vet`, `manifests`, `generate`, `test-e2e`.
- Operator `test`/`build` targets auto-run `manifests generate fmt vet` first, so CRD manifests
  and deepcopy code stay in sync. If you change the API types in `operator/api/`, re-run
  `cd operator && make manifests generate` and commit the results.
- Lint is `golangci-lint` (config: `.golangci.yaml`).
- E2E tests live in `tests/e2e`; load tests in `tests/load`.

**Code coverage:** the goal is 100% unit-test coverage across the codebase, reached
incrementally. Every change you make - new code or edits to existing code - must ship with
unit tests that cover 100% of the lines and branches it touches. Do not leave new or modified
code uncovered.

Use Go at `/Users/tf/.homebrew/bin/go` (1.26.5) if `go` on PATH is older.

## Conventions

- Handle every error explicitly; no silent `_ =` on error returns. Use `Result`/wrapped errors idiomatically.
- Match surrounding style; don't add comments unless the code is genuinely non-obvious.
- Every commit is signed off: `git commit -s`. Never add an agent as co-author or put any
  agent signature in commits or PRs.
- Work on a branch, never commit directly to `main`.
- Never hand-edit auto-generated files: `charts/elasti/README.md` (helm readme generator),
  `docs/index.yaml` (helm index), CRD manifests, and `CHANGELOG.md` release-automation output.

## Helm chart

The chart is `charts/elasti/`. `Chart.yaml` holds `version`/`appVersion`; `values.yaml` holds
the operator and resolver image tags. Images and chart publish to `ghcr.io/kubeelasti`.

## Releases

The release process is documented in `RELEASE.md` and automated by the `release` skill
(`.claude/skills/release/`). In short: bump `charts/elasti/Chart.yaml` + `charts/elasti/values.yaml`
+ `CHANGELOG.md`, open a PR to `main`, then create a `v*` GitHub release/tag which triggers
`.github/workflows/release.yaml` to build, cosign-sign, SBOM, and publish images and the chart.
The version in files carries no leading `v` (e.g. `0.1.31-rc1`); the git tag does (`v0.1.31-rc1`).
