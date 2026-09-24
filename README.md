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

The template targets Dex Server `v0.11.4`, Dex Web v2 `v0.2.0`, and the Dex Go
SDK `v0.11.3`. `DEX_SERVER_BASELINE` and `DEX_WEB_V2_BASELINE` pin the runtime
and rendering releases used by CI.

Applications that do not need a custom process UI keep only a non-business
Hello World page and the Go/OpenAPI/React generation skeleton. Dex Web remains
the complete process-management surface. Remove the template's approval,
display, status, list, detail, mock-lifecycle, and Action-proxy routes and
components. A trigger webhook may remain, but it is integration ingress rather
than a management API. If the process later needs a custom UI, preserve this
architecture and return to the mock-first approval workflow before wiring new
production behavior.

## Start locally

For the fastest UI interaction loop, start the in-memory mock API and Vite HMR:

```bash
make bootstrap
make mock
```

Open <http://127.0.0.1:8080>. Mock Controls can advance the process, emit a
reminder, inject the next start/refresh/approval failure, or reset all state.
The server retains state across browser refreshes and resets it on restart.
See [Local mock](docs/local-mock.md) for the complete contract.

To run the real Dex Worker and API instead:

```bash
make bootstrap
make dev
```

Open <http://127.0.0.1:8080>. `make dev` owns a local `dexcli dev` process and
cleans it up on exit.

## Contract and generated code

`openapi/openapi.yaml` is authoritative. Ogen creates the Go server contract in
`internal/api/generated`; Hey API creates the TypeScript client in
`web/src/api/generated`.

```bash
make generate
make check-generated
make check-fdg-v2
```

Generated files are committed so a checkout is immediately understandable.
Never edit them manually.

## Verification

```bash
make test-unit
make test-integration
make test-e2e
make test-mock-e2e
make check
```

Integration tests start a real Dex Server with `dexcli dev`. Playwright drives
the production UI and uses `dexcli flow skip-timer` to exercise the reminder
branch without waiting fifteen minutes. Every poll has a deadline.

Mock E2E runs the same frontend against the Go in-memory server without Dex.
It validates loading, failure recovery, reminders, approval, refresh restore,
completion, and reset. It does not prove durable execution behavior.

`make check-fdg-v2` validates `internal/process/flow.go` with rendering schema
2.0 and requires a diagnostic-free graph with `valid: true`. The required
preview `dexcli` source is pinned in `DEX_WEB_V2_BASELINE`; schema v1 is not an
accepted fallback.

`make check` is the required completion gate for coding agents and CI.

The supported sandbox runtime is contract revision 2. It provides Go, Node.js,
npm, Python 3, an FDG 2.0-capable `dexcli`, Ogen's cached module dependencies,
and Chromium Headless Shell. JavaScript packages remain pinned by
`web/package-lock.json`. `make bootstrap` restores the versions declared by
the Go module files and npm lockfile; it does not authorize unrelated upgrades.
Project dependencies must stay pinned in their manifests and lockfiles. The
sandbox does not support global npm packages, operating-system package
installation, or remote installer scripts.

## Dex skills

The local `dex-app-builder` and `dex-sdk` entries delegate to one pinned public
`dex-skills` submodule. Initialize it with:

```bash
git submodule update --init --recursive
```

`dex-app-builder` is the product workflow entrypoint and loads the sibling
`dex-sdk` Core and Go guidance for backend implementation. Template maintainers
update the pin explicitly; generated applications never follow the skill
repository's `main` branch implicitly.
