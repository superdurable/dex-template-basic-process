# Dex Basic Process Template

A Superverse `go-react-v1` application with a non-business Hello World page,
a generated Go/OpenAPI/React contract, and one minimal `process.ExampleFlow`:

```text
ExampleStep → Complete
```

The example takes no business input, has no external effects, and stores no
business state. Studio can render its strict FDG immediately after project
creation. Replace the example when implementing the first business feature;
do not retain an unrelated sample Flow beside the requested application.
`GetDexSummary` and `GetDexDisplay` are empty read-only views required by FDG 2.0.
Runs, configuration and inspection belong to Studio or standalone Dex management.

The template targets Dex Server `v1.4.1`, Dex CLI `v1.4.2`, and the Dex Go
SDK `v1.4.0`. `DEX_SERVER_BASELINE` and `DEX_CLI_BASELINE` pin the hosted
runtime and local tooling releases used by CI. Dex Web is embedded in both
release artifacts rather than published as a separate package.

`internal/worker` assembles the Dex Worker and
blob cache and manages their startup and shutdown through `worker.Worker`.
Repository-owned names must describe concrete responsibilities. The
case-insensitive stems `runtime` and `normaliz` are prohibited in names,
including package paths, types, aliases, receivers, and test helpers. Generated,
third-party, framework-mandated, and immutable legacy references are exempt.
Static checks in `tools/checkcontract` enforce this rule in
`make check-contracts` and `make check`.

The default page only calls `GetApplicationInfo`; the application has no Flow
management HTTP routes. The example Flow is available for adaptation. Add application APIs and a custom
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
Adding this official loader keeps Go `1.25.0` and Dex SDK `v1.4.0` unchanged.

## Verification

Template CI additionally runs `make check-template-repository` to validate its
release and dependency-update workflows. Managed Superverse applications are
exported without `.github/`; their `make check-contracts` and `make check-static`
retain all application checks without requiring template publishing automation.

```bash
make check
# Or rerun the source gate after dependencies have been restored:
make check-static
```

The template contains no application integration or browser test scaffold.
Do not generate tests, fixtures, mocks or test-framework dependencies by default.
This keeps the application small and avoids requiring private Connector keys
while authoring source. Explicitly requested tests must call real dependencies;
missing dependencies are an incomplete result, never a mock-backed pass.

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
artifacts without starting services or calling providers. `make check` restores
pinned dependencies and runs that source gate. After source handoff, configure
credentials through the host and validate actual business execution in Preview
and Live. Those host-owned acceptance checks are not copied into each app. A
successful source check permits configuration and deployment; it does not claim
that a provider call or business journey passed.

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

## Template 1.9.1 history

Version 1.9.1 separates template-repository automation checks from exported
application checks. Flow behavior, SDK pins, configuration loading and UI are
unchanged. The template CI still requires both checks.

### Dex Flow Changes

None. `process.BasicProcessFlow`, its Steps, primitive keys, start and recovery
semantics are unchanged. The application shell no longer proxies that Flow; host
management uses its declared typed RPCs directly.

### Database Schema Changes

None. Configuration uses the official versioned project objects; no database or
new lifecycle store is introduced.

### UI/UX

Hello World and public application information replace the sample approval UI.
Runs and Actions belong to the authenticated host management surface.

### Tests

Default applications contain no integration/browser test suite. Full source
checks and production builds are independently runnable before configured host
Preview/Live acceptance; runtime results remain explicitly unverified until
the real dependency path runs.

### Documentation

This README, AGENTS.md, template manifest and static contracts describe the same
source gate, configuration loader and host management boundary.


## Template 1.9.3 source-only baseline (historical)

### Dex Flow Changes

None. Existing Flow, Step, primitive, RPC, execution and cleanup contracts remain
unchanged; this release removes test scaffolding only.

### Database Schema Changes

None. No new state owner or storage is introduced.

### UI/UX

The Hello World shell and native host management surface remain unchanged.

### Tests

Default verification is `make check` and `make check-template-repository` in
Template CI. Real configured Preview/Live acceptance remains host-owned and must
be reported independently; no passing integration evidence is implied.

### Documentation

The template manifest, Make commands, CI, agent guidance and this README describe
the same source-only default. Go, SDK, Server and CLI baselines are unchanged.


## Manifest source validation

Version 1.9.3 requires each Connector's exact released `modulePath` in
`dex-app.yaml`, preserving it in `connector-contract.json`. The short Connector
ID does not identify a Go module. The immutable published Connector manifest
supplies that module identity. Declared operations and Trigger bindings are
validated, and release output normalizes absent bindings to an empty list.
This aligns source checks with Studio's deployment schema; it adds no tests,
Flow primitives, database, UI surface, or dependency upgrades. The source-only
checks and configured host acceptance boundary above remain unchanged.

## Template 1.9.5: focused authoring and actionable diagnostics

Strict FDG failures report the renderer's bounded diagnostic details directly.
When the host commit tool runs `make check`, that tool owns the final full source
gate; run focused checks while editing rather than repeating the full gate just
before committing. Run `make generate` before resolving imports of the ignored
generated API packages. A clear UI request can be implemented directly; a
wireframe is useful when a material interaction choice needs clarification.
Go, Dex SDK, Connector SDK and frontend pins are unchanged.

## Template 1.9.4: minimal initial design

### Dex Flow Changes

`process.ExampleFlow` is an independent top-level Flow with a single
`process.ExampleStep` that executes synchronously and completes the engine
execution. There is no archival state, SubFlow, Continue-As-New or business
state transfer. Studio creates a business ID for each explicit start:
`flow-<UUIDv4>`, for example `flow-550e8400-e29b-41d4-a716-446655440000`.
Studio owns authorization and the complete logical Start RequestID, uses
`IDReuseDisallow` and same-request `IgnoreError`, and recovers uncertain starts
using that original identity. An explicit rerun creates a new FlowID; automatic
retry never deliberately creates a new RunID.

Attributes, AttributeMaps, Channels, ChannelMaps and Streams: **None**. There
are no indexed fields, locks, CAS writes, pending messages, secrets or growing
collections. No stream reconnection or primitive cleanup is required. Retention
is the owning Dex Server's completed-execution retention; Project deletion owns
final resource cleanup. There is no database schema or mirrored business state.

`GetDexSummary` and `GetDexDisplay` read no primitives and return empty objects.
Studio authorizes those reads at its project boundary. `ExampleStep.Execute`
returns `GracefulComplete(nil)` with a ten-second attempt timeout, one attempt
and synchronous durability. There is no WaitFor, I/O, heartbeat loop or external
effect to reconcile. Timeout, cancellation or local failure is diagnosed through
the engine; an authorized operator can recover the same execution. No application
cleanup is needed.

This is a new-project template change. Existing generated applications retain
their source and `BasicProcessFlow` executions; they are neither renamed nor
reset. The old sample service, approval primitives and six Steps are removed
only from the new template. Go, SDK and Connector SDK pins remain unchanged.

### Database Schema Changes

None. No new persistent storage is introduced.

### UI/UX

The Hello World shell is unchanged. The host shows the exact template design
before coding and uses business definitions after the Agent replaces the sample.

### Tests

`make check` validates strict FDG, generation, modules, formatting, vet and both
production builds. Real start/completion and first-feature replacement are
verified by Superverse Native acceptance after release; static checks do not
claim that runtime result. No application test scaffold is added.

### Documentation

README, AGENTS.md, the manifest and the worker registration describe the same
single-Step example and replacement policy.

## Template 1.9.6: application authoring boundaries

### Dex Flow Changes

None. ExampleFlow and ExampleStep keep their existing identity and completion.

### Database Schema Changes

None.

### UI/UX

No user interface changes. An exported application's README describes its own
business behavior without repeating template maintenance prose.

### Tests

`make check` still validates source, names, pins, strict FDG, generated clients
and production builds. `make check-template-repository` additionally validates
all template documentation and updater requirements. Release artifacts preserve
the canonical empty `authMethodId` of a single-method Connector; the host checks
the exact released manifest before configuration. This is static validation,
not real provider execution.

### Documentation

Template publishing rules apply only to the upstream template repository.
Generated business applications preserve their accepted template version.
