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

Use precise domain names. Do not use the case-insensitive stems `runtime` or
`normaliz` in repository-owned package, directory, file, type, interface,
method, function, field, parameter, variable, constant, schema, configuration,
or resource names. Name the concrete execution role or transformation instead,
such as `TrimWhitespace`, `CanonicalizeURL`, or
`ValidateAndSortSelections`. Generated and third-party code,
framework-mandated identifiers, and migration code or tests that must reference
immutable legacy names are exempt.

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
`make test-unit`, `make test-integration`, `make test-e2e`, `make build`,
`make dev`, and `make check`.

`make bootstrap`, `npm ci`, and `go mod download` may restore dependencies
already declared by the committed manifests and lockfiles. Before adding or
upgrading a project dependency, verify that the standard library and existing
dependencies cannot satisfy explicit requested behavior. Pin the selected
version, update the manifest and lockfile together, explain why it is needed,
and run `make check`. Do not add convenience-only dependencies, perform
unrelated upgrades or audit auto-fixes such as `npm audit fix`, install global
or operating-system packages, or run remote installation scripts.

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
