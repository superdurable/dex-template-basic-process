# Dex Basic Process Template

A complete Superverse `go-react-v1` template for a durable approval automation.
The Go backend and React TypeScript UI share one OpenAPI contract. The Dex Flow
validates a request, waits durably for approval, emits recurring reminders, runs
the approved automation, and completes with a typed result.

The durable lifecycle is:

1. `StartProcess`
2. `ValidateRequest`
3. `WaitForApproval`
4. `EmitReminder`
5. `ExecuteApprovedAutomation`
6. `CompleteProcess`

`WaitForApproval` races an Approval Channel against a durable 15-minute Timer.
When the Timer fires, `EmitReminder` increments the durable reminder count and
returns to the waiting step. Approval advances to the execution and completion
steps.

The Flow is also a complete Dex Web v2 definition. Indexed title and state
Attributes drive list/search and Action eligibility. `GetDexSummary` and
`GetDexDisplay` provide the read-only Web views, while `ApproveProcess` is a
native Web v2 Action. Every Step has an FDG 2.0 group and explanation.

The template targets Dex Server `v0.14.0`, Dex CLI `v0.14.0`, and the Dex Go
SDK `v0.13.1`. `DEX_SERVER_BASELINE` and `DEX_CLI_BASELINE` pin the hosted
runtime and local tooling releases used by CI. Dex Web is embedded in both
release artifacts rather than published as a separate package.

Applications that do not need a custom process UI keep only a non-business
Hello World page and the Go/OpenAPI/React generation skeleton. Dex Web remains
the complete process-management surface. Remove the template's approval,
display, status, list, detail, mock-lifecycle, and Action-proxy routes and
components. A trigger webhook may remain, but it is integration ingress rather
than a management API. If the process later needs a custom UI, preserve this
architecture, confirm a static wireframe, and then wire the generated client to
the real Go and Dex backend.

## Start locally

Bootstrap the locked dependencies, then start the real Dex Worker and API:

```bash
make bootstrap
make dev
```

Open <http://127.0.0.1:8080>. `make dev` owns a local `dexcli dev` process and
cleans it up on exit.

## Contract and generated code

`openapi/openapi.yaml` is authoritative. Ogen creates the Go server contract in
`internal/api/generated`; Hey API creates the TypeScript client in
`web/src/api/generated`. Both directories are ignored local build outputs.
`make generate` writes both outputs to a temporary directory and replaces the
working copies only after both generators succeed.

```bash
make generate
make check-fdg-v2
```

Run `make bootstrap` once after a fresh checkout so the locked generators are
available. Build, test, development, and full-check targets regenerate the
clients before compiling. Never edit or commit generated files.

## Verification

```bash
make test-unit
make test-integration
make test-e2e
make check
```

Integration tests start a real Dex Server with `dexcli dev`. Playwright drives
the production UI and uses `dexcli flow skip-timer` to exercise the reminder
branch without waiting fifteen minutes. Every poll has a deadline.

Vitest mocks the generated client for isolated loading, failure, and terminal
UI states. A test may use Playwright request interception for a browser-only
edge case, but the template does not ship a mock API, a second business state
machine, or user-visible Mock Controls. Mock evidence does not prove durable
execution behavior.

`make check-fdg-v2` validates `internal/process/flow.go` with rendering schema
2.0 and requires a diagnostic-free graph with `valid: true`. The required
preview `dexcli` source is pinned in `DEX_CLI_BASELINE`; schema v1 is not an
accepted fallback.

`make check` is the required completion gate for coding agents and CI.

The supported sandbox runtime is contract revision 3. It provides Go, Node.js,
npm, Python 3, an FDG 2.0-capable `dexcli`, Ogen's cached module dependencies,
and Chromium Headless Shell. JavaScript packages remain pinned by
`web/package-lock.json`. `make bootstrap` restores the versions declared by
the Go module files and npm lockfile; it does not authorize unrelated upgrades.
Project dependencies must stay pinned in their manifests and lockfiles. The
sandbox does not support global npm packages, operating-system package
installation, or remote installer scripts.

## Dex skills

Develop this template with the released
[Dex plugin](https://github.com/superdurable/dex-skills#install) installed in
the coding-agent host. Invoke `dex-app-builder` as the product workflow
entrypoint; it loads the matching `dex-sdk` guidance. Superverse Coding Sandbox
preinstalls a pinned release, while external developers install the plugin in
Codex, Claude, Cursor, or another Agent Skills client. The template never
assumes a fixed skill path. Generated projects do not contain, initialize, or
read a skill submodule.

### Automated dependency maintenance

The **Update Dex dependencies** workflow runs daily and can also be started
manually from GitHub Actions. It compares the template with the latest stable
Dex Go SDK, Dex Server, and Dex CLI releases. When any component is newer, the
workflow refreshes module locks, release baselines, documentation, and the
template contract on the fixed `automation/update-dex-dependencies` branch. It
then creates or refreshes one pull request and explicitly dispatches Template
CI.

The updater ignores prereleases and unreleased branches, and it never merges
its pull request automatically. Every merged template change publishes the
manifest's version as an immutable GitHub release after Template CI passes.
Dex Skills and Superverse then advance their release pins through separate
reviewed pull requests.
