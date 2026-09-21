# Basic Process Template Instructions

This is a complete Superverse `go-react-v1` application. Read
`.superverse/template.json`, `openapi/openapi.yaml`, and the local
`dex-app-builder` skill before changing product behavior. Its pinned upstream
skill loads the sibling `dex-sdk` Core and Go guidance for backend work.

`openapi/openapi.yaml` is the only HTTP contract source. Never edit files below
`internal/api/generated` or `web/src/api/generated` by hand. Change the spec,
run `make generate`, and update server, UI, integration, and E2E coverage in the
same change.

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

Stable commands are `make bootstrap`, `make generate`, `make check-generated`,
`make check-fdg-v2`, `make test-unit`, `make test-integration`,
`make test-e2e`, `make test-mock-e2e`, `make build`, `make dev`, `make mock`,
and `make check`.

`make mock` is the UI approval loop. It starts the Go in-memory mock API and
Vite HMR without Dex. Keep the mock implementation behind `cmd/mock-server`
and `/__mock__`; the production server must return 404 for mock controls.
Mock verification does not replace the real Dex integration and E2E tests.

When structure, commands, or required tooling changes, update this file,
`.superverse/template.json`, `README.md`, and contract tests together. Do not
maintain a separate static repository map.
