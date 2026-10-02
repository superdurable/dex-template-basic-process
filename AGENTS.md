# Basic Process Template Instructions

This is a complete Dex `go-react-v1` application. Read
`.superverse/template.json` and `openapi/openapi.yaml`, then load the installed
`dex-app-builder` skill through the current coding-agent host before changing
product behavior. Superverse Coding Sandbox preinstalls a pinned Dex Skills
release; external developers install the released Dex plugin in their coding
agent. Never assume a fixed skill path. This repository must not vendor, clone,
or initialize a project-local copy.

The template starts without a custom process UI. Keep its Hello World shell;
Studio provides process management in Superverse, and standalone developers may
use Dex Web. A clear implementation request is authorization to implement: infer
routine defaults from that request and the host, and ask only for missing
business decisions that materially change behavior. Do not make users choose
Go types, Connector API methods, or debug compiler errors. Add a custom UI only
for a requested interaction that the host management surface cannot provide;
confirm that interaction's static wireframe before wiring it to real APIs.

`openapi/openapi.yaml` is the only HTTP contract source. Never edit files below
`internal/api/generated` or `web/src/api/generated` by hand. Change the spec,
run `make generate`, and update server, UI, integration, and E2E coverage in the
same change. Both generated directories are ignored local build outputs; never
add them to Git or include them in a pull request.

Use precise domain names that describe the concrete responsibility. Do not use
the case-insensitive stems `runtime` or
`normaliz` in repository-owned package, directory, file, type, interface,
method, function, field, parameter, variable, constant, schema, configuration,
or resource names. Name the concrete execution role or transformation instead,
such as `TrimWhitespace`, `CanonicalizeURL`, or
`ValidateAndSortSelections`. Generated and third-party code,
framework-mandated identifiers, and migration code or tests that must reference
immutable legacy names are exempt.
This prohibition includes aliases, receivers, test helpers, abbreviations that
retain either stem, and compound names such as `AppRuntime`, `RuntimeManager`,
or `NormalizeInput`. Renaming must update the package path, declarations,
imports, callers, and tests together; do not retain compatibility aliases for
repository-owned names. `internal/worker` owns the Dex Worker
and blob cache lifecycle. The static Go naming check in `tools/checkcontract`
runs with `make check-contracts` and `make check`; do not bypass it or add
exceptions for new repository-owned names.

The initial `process.ExampleFlow` has one `ExampleStep` and completes. On the
first business feature, replace this scaffold: remove its Flow, Step, obsolete
RPCs and registration, and update `dex-app.yaml` to list only the requested
business definitions. Do not leave ExampleFlow (or an older BasicProcessFlow)
alongside the new business Flow unless the user explicitly requests it. Preserve
unrelated business Flows in imported or already-developed applications.

Keep external effects in `Execute`, register every durable primitive, and
preserve open business Flow compatibility unless migration is authorized.
The initial example has no durable primitives or external effects.

Every Flow is a strict FDG 2.0 definition with `GetDexSummary` and
`GetDexDisplay` RPCs. Keep Flow metadata and control flow together (initially
`internal/process/example_flow.go`). Every Step has exactly one group and
explanation. Run `make check-fdg-v2`; never fall back to rendering schema v1.

After each edit batch, run the narrowest relevant Make target. Before a source
handoff, `make check` must pass. It restores pinned dependencies and runs
`make check-static`: generate clients, validate strict FDG and contracts, check
Go formatting/modules/vet, and build the production Go binary and frontend.
These commands do not start test services or call providers.

Do not add application integration or browser test suites, mocks, fake providers,
fixtures, or test-framework dependencies unless the user explicitly requests
them. The template intentionally has no application test scaffold. Existing
source-only applications follow the same default. A separately requested real
business acceptance belongs to the host after credentials are configured.
Missing Connector credentials must not block a complete, source-verified commit:
report that the application is ready to configure and real execution is pending.
Never claim that compilation proves a provider call. A failed source check is a
source blocker and must be fixed when possible. Do not remove unrelated existing
tests from an imported application without authorization.

Stable commands are `make bootstrap`, `make generate`, `make check-fdg-v2`,
`make check-contracts`, `make check-static`, `make superverse-release-artifacts`,
`make build`, `make dev`, and `make check`.

`dex-app.yaml` declares every application Flow source and exact connector
connection. Update it whenever either set changes. Never put endpoint values,
tokens, keys, refresh tokens, webhook secrets, or other configuration material
in this manifest or the generated Release artifacts.

`internal/connectorconfiguration` loads the official Connector SDK before any
Worker or application goroutine starts. Project deployments use `DEX_PROJECT_*`
and the exact snapshot version/digest; standalone connections use
`DEX_CONNECTOR_CONFIG_FILE`. When adding a Connector, retain the loaded store
and use the released typed adapter. Connection settings, operation configuration,
and credentials have separate APIs: decode connection settings with
`Configuration.DecodeConnectionConfiguration`, operation settings with the
operation configuration loader, and current credentials with the official
`projectconfig/provider` adapter. Inspect the pinned source for exact signatures;
never guess fields or copy provider secrets into application code or Flow state.

`make bootstrap`, `npm ci`, and `go mod download` may restore dependencies
already declared by the committed manifests and lockfiles. Before adding or
upgrading a project dependency, verify that the standard library and existing
dependencies cannot satisfy explicit requested behavior. Pin the selected
version, update the manifest and lockfile together, explain why it is needed,
and run `make check`. Do not add convenience-only dependencies, perform
unrelated upgrades or audit auto-fixes such as `npm audit fix`, install global
or operating-system packages, or run remote installation scripts.

Exact Dex Go SDK, Dex Server, and Dex CLI versions in manifests, lockfiles, and
`DEX_*_BASELINE` files are reproducibility baselines, not compatibility
ceilings. A coding agent may use a newer stable patch release within the
currently selected major/minor line without separate user authorization. Keep
all affected exact pins, lockfiles, documentation, and contract assertions in
sync, and run `make check`. Do not select a prerelease, downgrade, or cross a
minor or major version boundary unless the user explicitly requests it.

`.github/workflows/update-dex-dependencies.yml` and
`scripts/update-dex-dependencies.py` own scheduled Dex Go SDK, Dex Server, Dex
and Dex CLI release updates. Keep their stable-release checks, fixed automation
branch, template-version bump, contract updates, and explicit CI dispatch
aligned. The updater creates or refreshes a pull request; it never merges one.
`make check-template-repository` checks these template-owned workflows and is
required by Template CI. Superverse exports applications without `.github/`;
`make check-contracts` and `make check-static` still validate their complete
application contracts, but must not require or recreate template release CI.

Every pull request advances `templateVersion` in
`.superverse/template.json`. After Template CI passes on `main`, CI publishes
that exact commit as `v<templateVersion>`. Never move or reuse a template tag.
Dex Skills discovers the release and opens its own baseline pull request. Once
the matching Dex Skills release exists, Superverse advances both immutable
release pins in one pull request.

If the user later requests tests, use actual dependency APIs and preserve the
real-dependency policy: no unit tests, mocks, fake providers or intercepted
responses. Missing required services fail that explicit test; do not skip it or
report it as passed. Static contract checks remain separate from test evidence.

When structure, commands, or required tooling changes, update this file,
`.superverse/template.json`, `README.md`, and static contract checks together.
Do not maintain a separate static repository map.

Every `dex-app.yaml` Connector declaration includes the exact released
`modulePath`, in addition to its connection name, Connector ID, version and
authorization method. Obtain the module from the immutable Connector manifest;
do not infer it from the short Connector ID. Declare operations and/or Trigger
bindings. Source checks and emitted release contracts must preserve these same
identities; credentials remain in the configuration store.
