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
Go naming contract tests enforce this rule in `make test-unit` and `make check`.

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

Project Dex Web owns connector configuration and OAuth. Configure named
connections and declared Trigger bindings before building a Release; deployment
then selects an exact immutable configuration snapshot. The application
Connector SDK resolves current credentials during actual use and refreshes only
when a known expiry has elapsed. No platform credential broker is required.

At startup, local development may supply `DEX_CONNECTOR_CONFIG_FILE`.
A project deployment supplies the following trusted `DEX_PROJECT_*` environment contract:

| Variable | Meaning |
| --- | --- |
| `DEX_PROJECT_ID` | Fixed project identity. |
| `DEX_PROJECT_SCOPE` | `live` or `preview`. |
| `DEX_PROJECT_SESSION_ID` | Required only for Preview. |
| `DEX_PROJECT_CONFIG_KEY` | `projects/<projectID>/live/configuration/head`, or `projects/<projectID>/preview/<sessionID>/configuration/head`. |
| `DEX_PROJECT_CONFIG_VERSION` | Exact immutable object version accepted by Dex Web. |
| `DEX_PROJECT_CONFIG_DIGEST` | `sha256:<hex>` over the exact configuration bytes. |
| `DEX_PROJECT_STORAGE_BUCKET` | Private versioned project bucket. |
| `DEX_PROJECT_STORAGE_PREFIX` | Optional fixed environment prefix. |
| `DEX_PROJECT_STORAGE_KMS_KEY_ARN` | Exact hosted KMS key ARN. |
| `PUBLIC_BASE_URL` | The actual application Preview or Live URL. |

AWS region and identity come from the standard AWS environment/credential chain.
Isolated local fixtures additionally set `DEX_PROJECT_ALLOW_LOCAL_STORAGE=true`
and `DEX_PROJECT_STORAGE_ENDPOINT` to a local MinIO endpoint. Hosted deployments
omit both. Scope, key, version, digest format, and storage boundary are validated
before the Worker starts. Local and project configuration cannot be combined;
obsolete platform broker environment variables are rejected.

This template declares no connectors or application environment fields, but all
project deployments use one shared application loader. At the beginning of
`worker.New`, `internal/projectconfiguration.Load(false)` calls
`projectconfig.LoadFromEnvironment`, resolves the accepted application
environment and applies it before creating any Dex Client, Worker, service or
goroutine. The shared loader verifies the exact object version, digest and
scope; it never reads a mutable configuration head or refreshes a credential at
startup. Missing or invalid accepted objects fail startup.

The Connector SDK dependency is needed for this actual bootstrap, including
connector-free apps with ordinary values and secrets. The tracked `sdkgo
v0.14.2` is a real published baseline, but it does not yet contain these additive
APIs. Local development uses an ignored workspace containing the official SDK
worktree. Production builds and the required standalone `make check` remain
blocked until the authorized upstream release and exact dependency promotion;
no unpublished version or committed local replacement is presented as released.
Use `projectconfig/provider` for typed connector credential resolution, preserving
complete renewal material. Never put credentials in Flow state, logs or browser
responses.

Applications may declare ordinary and secret startup strings in
`application.environment`. Declarations contain names and constraints only:

```json
{
  "application": {
    "port": 8080,
    "healthPath": "/healthz",
    "environment": [
      {"name": "APP_ENV", "required": true, "enum": ["development", "production"]},
      {"name": "EVENT_TOKEN_SECRET", "required": true, "secret": true, "minLength": 32}
    ]
  }
}
```

This is an extension example; the checked-in template has no such fields.
Project Dex Web collects values before a Release exists. Ordinary values enter
its immutable configuration snapshot; secret values remain in encrypted scoped
objects referenced by exact key, version and digest. The application consumes
only the accepted versions. Replacing a secret affects a newly accepted
configuration; it does not silently rotate a running application. Removing
Live preserves these objects, while final project/scope deletion cleans them.

Names are unique uppercase ASCII identifiers, at most 128 characters. There are
at most 128 fields. `required` and `secret` default to false; `minLength` defaults
to zero and counts Unicode code points. Values are limited to 32768 UTF-8 bytes,
with no NUL characters. `enum` is an optional unique string list and is forbidden
for secret fields. Missing required values fail validation; an explicitly empty
string is permitted only when its other constraints allow it. Values, defaults,
and secret references never belong in the manifest or generated Release
artifacts. Infrastructure and process-control variables—including `DEX_*`,
`AWS_*`, `SUPERVERSE_*`, `PUBLIC_BASE_URL`, proxy overrides and loader/toolchain
settings—are reserved. The generator canonicalizes every nonempty declaration
list by name, sorts enums, and emits all five declaration fields. An absent or
empty list leaves the existing empty environment contract unchanged.

The template's project startup boundary already uses the shared loader, including
when there are no connectors. Apps adding clients retain the same startup order:
resolve all values and call `environment.Apply()` before reading application
settings, constructing clients, starting Workers or launching goroutines. Never
read application settings in `init` functions or package variable initializers:


```go
func loadProjectStartup(ctx context.Context) (*projectconfig.LoadedProject, error) {
    loaded, err := projectconfig.LoadFromEnvironment(ctx)
    if err != nil {
        return nil, err
    }
    environment, err := loaded.ResolveApplicationEnvironment(ctx)
    if err != nil {
        return nil, err
    }
    if err := environment.Apply(); err != nil {
        return nil, err
    }
    return loaded, nil
}
```

Use `loaded.Configuration` for immutable connection/Trigger/operation settings
and `loaded.Connections` for current credential resolution. Neither resolved
application values nor credentials may enter Flow state, logs or browser DTOs.
Missing or invalid project configuration must fail startup rather than falling
back to development credentials. Preserve a deliberate, separate local
configuration path for development. Production pinning of these additive APIs
waits for their authorized SDK release; do not invent an unreleased version.

Every connector declaration includes `modulePath`, the exact released `version`,
`connectionName`, `connectorId`, selected `authMethodId`, `operations`, and
optional `triggerBindings` containing only `triggerName`/`bindingName` pairs.
`modulePath` names an official connector-library module; the application does
not derive it from a connector ID. The generator always emits `triggerBindings`
(including an empty list), sorted by trigger and binding name. At least one
operation or Trigger binding is required. Configuration values and credentials
remain outside `dex-app.yaml` and Release artifacts.

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
