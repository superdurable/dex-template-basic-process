# Basic Process Template Instructions

This is a complete Dex `go-react-v1` application. Read
`.superverse/template.json` and `openapi/openapi.yaml`, then load the installed
`dex-app-builder` skill through the current coding-agent host before changing
product behavior. Superverse Coding Sandbox preinstalls a pinned Dex Skills
release; external developers install the released Dex plugin in their coding
agent. Never assume a fixed skill path. This repository must not vendor, clone,
or initialize a project-local copy.

During product adaptation, first confirm whether the process needs a custom UI.
If it does not, use Dex Web for process management and retain only a
non-business Hello World page plus the Go/OpenAPI/React generation skeleton.
Remove process-management routes, components, fixtures, and related tests.
Keep only explicitly required integration ingress such as a trigger webhook.
If a custom UI is required, confirm a static wireframe before connecting the
generated client to the real Go and Dex backend.

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
repository-owned names. `internal/worker` owns the Dex Worker, Client,
and blob cache lifecycle. The Go naming contract in `internal/templatecontract`
runs with `make test-unit` and `make check`; do not bypass it or add exceptions
for new repository-owned names.

The Dex Flow has stable Step, Attribute, Channel, Timer, and RPC identities.
Keep external effects in `Execute`; `WaitForApproval.WaitFor` only declares the
approval Channel and reminder Timer. Register every durable primitive in the
Flow persistence schema. Preserve open-Flow compatibility unless the user
explicitly requests a migration.

The Flow is a Dex Web v2 / FDG 2.0 definition. Keep its indexed Attributes,
`GetDexSummary`, `GetDexDisplay`, Action RPCs, directives, input structs, and
Dex control flow in `internal/process/flow.go`. Every Step has exactly one
group and explanation. Run `make check-fdg-v2`; never fall back to rendering
schema v1.

After each edit batch, run the narrowest relevant Make target. Before calling
`commit_and_push`, run `make check` successfully and include it in verification.
If `make check` fails or cannot run, report `blocked=true`. Do not weaken, skip,
or delete a failing check.

Stable commands are `make bootstrap`, `make generate`, `make check-fdg-v2`,
`make superverse-release-artifacts`, `make test-unit`,
`make test-integration`, `make test-e2e`, `make build`, `make dev`, and
`make check`.

`dex-app.yaml` declares every application Flow source and exact connector
connection. Update it whenever either set changes. Never put endpoint values,
tokens, keys, refresh tokens, webhook secrets, or other configuration material
in this manifest or the generated Release artifacts.

Every connector declaration includes the exact official `modulePath`, released
`version`, `connectionName`, `connectorId`, selected `authMethodId`, `operations`,
and optional `triggerBindings` pairs containing `triggerName` and `bindingName`.
Canonical contracts always emit a sorted `triggerBindings` list, including when
empty. A connection must declare at least one operation or Trigger binding.
Project Dex Web owns mutable configuration and OAuth before Release creation.

Project deployments use the trusted `DEX_PROJECT_*` scope, versioned storage,
and immutable configuration-reference contract documented in `README.md`.
`internal/projectconfiguration.Load(false)` invokes the shared Connector SDK
`projectconfig.LoadFromEnvironment` and resolves/applies the accepted application
environment even when the template has no connectors. Keep this startup path
before all application settings reads, client construction, Worker creation, and
goroutines. Never read application environment in `init` functions or package
variable initializers. Use `projectconfig/provider` for typed connector
credentials when adding connectors. The shared loader verifies the exact
snapshot version/digest and scoped secret versions; do not duplicate it with a
platform-downloaded JSON file. These additive APIs are not yet released: local
verification uses an ignored workspace against the official SDK worktree;
production build/pin promotion remains blocked until an authorized release.
Never invent a released pin. Credential refresh occurs only during actual expired use. Do not
restore a platform broker, periodic refresh, or credentials in Flow state.

`application.environment` declares application-owned startup strings using only
`name`, `required`, `secret`, `minLength` and `enum`; never add values, defaults or
secret references to the manifest. Names are unique uppercase ASCII, at most
128 characters and 128 fields. `required`/`secret` default false, `minLength`
defaults zero and counts Unicode code points; values are bounded to 32768 UTF-8
bytes. Secret enums are forbidden. Canonical nonempty declarations include all
five fields, sort names/enums, and preserve an absent/empty list's existing
contract. Reject platform/SDK/AWS, process-loader, proxy and toolchain overrides.
All project deployments, including connector-free applications, use
`loaded.ResolveApplicationEnvironment(ctx)` and `environment.Apply()` before
constructing clients or starting any goroutines. Resolve only exact accepted
secret versions from the same Live/Preview scope; do not copy plaintext values
into Flow state or generated artifacts. Startup errors must not trigger fallback
to development credentials. DeleteLive preserves accepted values; final scope
removal deletes them. Reauthorization, credential refresh, and application
secret replacement are distinct operations with their own existing owners.

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

Every pull request advances `templateVersion` in
`.superverse/template.json`. After Template CI passes on `main`, CI publishes
that exact commit as `v<templateVersion>`. Never move or reuse a template tag.
Dex Skills discovers the release and opens its own baseline pull request. Once
the matching Dex Skills release exists, Superverse advances both immutable
release pins in one pull request.

Use Vitest mocks of the generated client for isolated loading, failure, and
terminal UI states. When a browser-only edge case cannot be reached
economically, use test-local Playwright request interception. Do not add an
application mock server, a second business state machine, or user-visible Mock
Controls. Mock evidence never replaces real Dex integration and E2E tests.

When structure, commands, or required tooling changes, update this file,
`.superverse/template.json`, `README.md`, and contract tests together. Do not
maintain a separate static repository map.
