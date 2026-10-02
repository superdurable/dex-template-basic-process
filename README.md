# Dex Basic Process Template

A Superverse `go-react-v1` application with a non-business Hello World page, a
generated Go/OpenAPI/React contract, and a durable approval Flow example. Studio
or standalone Dex management provides Runs, Actions and inspection. The Dex Flow
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

The template targets Dex Server `v0.14.1`, Dex CLI `v0.14.1`, and the Dex Go
SDK `v0.13.1`. `DEX_SERVER_BASELINE` and `DEX_CLI_BASELINE` pin the hosted
runtime and local tooling releases used by CI. Dex Web is embedded in both
release artifacts rather than published as a separate package.

`internal/worker` assembles the process service, Dex Worker, Client, and
blob cache and manages their startup and shutdown through `worker.Worker`.
Repository-owned names must describe concrete responsibilities. The
case-insensitive stems `runtime` and `normaliz` are prohibited in names,
including package paths, types, aliases, receivers, and test helpers. Generated,
third-party, framework-mandated, and immutable legacy references are exempt.
Static checks in `tools/checkcontract` enforce this rule in
`make check-contracts` and `make check`.

The default page only calls `GetApplicationInfo`; the application has no Flow
management HTTP routes. The example Flow and its real-Dex integration scenarios
remain available for learning and adaptation. Add application APIs and a custom
UI only for actual product interactions not covered by the host. Preserve the
generated contract pipeline when adapting the Flow; check the full production
frontend as well as Go so removed APIs cannot leave broken imports behind.

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

## Hosted release artifacts

`dex-app.yaml` is the checked-in application manifest. It lists every Flow
Definition source and every exact connector connection used by the whole app;
it never contains configuration values or credentials. The file uses
JSON-compatible YAML so release generation has no undeclared parser dependency.

The Superverse build profile runs:

```bash
make superverse-release-artifacts SUPERVERSE_RELEASE_ARTIFACT_DIR=/tmp/release
```

The target requires the platform-provided Release, project, source commit, and
build-profile identities. It renders every declared Flow with FDG 2.0 and emits
`flow-definitions.json`, `connector-contract.json`,
`environment-contract.json`, and the exact `dex-app.yaml`. Invalid, duplicate,
or diagnostic-bearing Flow Definitions fail the Release build. Superverse pins
the resulting object versions and digests; application secrets are never part
of these artifacts.

At startup, `internal/connectorconfiguration` uses the released Connector SDK
`v0.17.0`. A project deployment provides trusted `DEX_PROJECT_*` scope, storage
and exact configuration key/version/digest. The official SDK verifies that
snapshot, resolves pinned application secrets and applies the environment before
clients or goroutines start. Shared Connector credentials are read only on actual
connection use through the typed `projectconfig/provider` adapter. There is no
mounted configuration file or credential broker. A partial or mixed deployment
contract prevents startup. Standalone development may use the official local
store selected by `DEX_CONNECTOR_CONFIG_FILE`.

Connection configuration (such as a provider or model), Step operation
configuration, and credentials are distinct. Use
`Configuration.DecodeConnectionConfiguration` for connection settings and the
operation configuration loader for exact Flow/Step settings. Match the pinned
Connector module's factory, config and credential types. Secret decoding and
refresh belong at that adapter boundary; never persist credentials in a Flow.
Adding this official loader keeps Go `1.25.0` and Dex SDK `v0.13.1` unchanged.

## Verification

Template CI additionally runs `make check-template-repository` to validate its
release and dependency-update workflows. Managed Superverse applications are
exported without `.github/`; their `make check-contracts` and `make check-static`
retain all application checks without requiring template publishing automation.

```bash
make check-contracts
make check-static
make test-integration
make test-e2e
make check
```

Integration tests start a real Dex Server with `dexcli dev`; the Go scenarios
cover approval, reminders, Worker restart and terminal reads. Playwright drives
the production Hello World page and generated application-info endpoint. Every
poll has a deadline. Go integration tests disable result caching so each run
calls the real Dex APIs.

Only integration tests that call real dependency APIs are permitted, including
the Playwright E2E journey against the production application. Unit tests,
isolated component tests, mock integration tests, fakes, stubs, and Playwright
API-response interception are prohibited, including for failure and edge cases.
Exercise those cases through real dependency APIs or report the coverage gap.
An integration label alone is insufficient. Missing dependencies fail the
check; tests must never skip or fall back to mocks. Do not add unit-test
frameworks, fixtures, or dependencies.

`make check-contracts` validates the template manifest, Go naming, test
placement, and known mocking APIs through static source inspection. It is
separate from API tests and does not prove dependency behavior. The dependency
updater keeps its release assertions in `tools/checkcontract/main.go`.

`make check-fdg-v2` validates every Flow source declared by `dex-app.yaml` and
its connector/environment contracts through the same renderer used for Release
artifacts. Each graph must use rendering schema 2.0, be diagnostic-free and have
`valid: true`. The required
preview `dexcli` source is pinned in `DEX_CLI_BASELINE`; schema v1 is not an
accepted fallback.

`make check-static` is the source handoff gate. It regenerates, validates FDG and
repository contracts, checks formatting/modules/vet, and builds both production
artifacts without starting services or calling providers. `make check` remains
the complete CI/standalone gate and adds real integration and browser scenarios.
In a hosted authoring sandbox, missing isolated integration services remain an
explicit acceptance gap; a successful source check permits Preview configuration
and deployment, not a claim that business execution passed.

The supported sandbox runtime is contract revision 3. It provides Go, Node.js,
npm, Python 3, an FDG 2.0-capable `dexcli`, Ogen's cached module dependencies,
and Chromium Headless Shell. JavaScript packages remain pinned by
`web/package-lock.json`. `make bootstrap` restores the versions declared by
the Go module files and npm lockfile; it does not authorize unrelated upgrades.
Project dependencies must stay pinned in their manifests and lockfiles. The
sandbox does not support global npm packages, operating-system package
installation, or remote installer scripts.

The exact Dex Go SDK, Dex Server, and Dex CLI pins are reproducibility
baselines, not compatibility ceilings. Coding agents may adopt a newer stable
patch release on the currently selected major/minor line without separate user
authorization, provided every affected pin, lockfile, document, and contract
assertion is updated together and `make check` passes. Prereleases, downgrades,
and minor or major version changes still require an explicit user request.

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

## Template 1.9 changes

Version 1.9.1 separates template-repository automation checks from exported
application checks. Flow behavior, SDK pins, configuration loading and UI are
unchanged. The template CI still requires both checks.

### Dex Flow Changes

None. `process.BasicProcessFlow`, its Steps, primitive keys, start and recovery
semantics are unchanged. The application shell no longer proxies that Flow; host
management and the existing typed integration client exercise it directly.

### Database Schema Changes

None. Configuration uses the official versioned project objects; no database or
new lifecycle store is introduced.

### UI/UX

Hello World and public application information replace the sample approval UI.
Runs and Actions belong to the authenticated host management surface.

### Tests

The real Dex scenarios remain. Browser coverage verifies the production shell,
generated endpoint and removal of the parallel management route. Full source
checks are independently runnable before hosted Preview acceptance.

### Documentation

This README, AGENTS.md, template manifest and static contracts describe the same
source gate, configuration loader and host management boundary.
