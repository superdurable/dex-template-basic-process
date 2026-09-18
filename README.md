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

## Start locally

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
```

Generated files are committed so a checkout is immediately understandable.
Never edit them manually.

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

`make check` is the required completion gate for coding agents and CI.

The supported sandbox runtime is contract revision 2. It provides Go, Node.js,
npm, Python 3, `dexcli`, Ogen's cached module dependencies, and Chromium
Headless Shell. JavaScript packages remain pinned by `web/package-lock.json`.

## Dex skill

The local skill entry delegates to the pinned public
`skill-dex-developer` submodule. Initialize it with:

```bash
git submodule update --init --recursive
```

Template maintainers update the pin explicitly; generated applications never
follow the skill repository's `main` branch implicitly.
