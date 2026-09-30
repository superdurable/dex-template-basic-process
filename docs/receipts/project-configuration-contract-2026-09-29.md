# Project configuration contract verification — 2026-09-29

Base commit: `22675514900881b5436166a453944058055cd7c5`.
Isolated worktree: `/private/tmp/sv2-template-project-config`.
Template version: `1.8.1` → `1.9.0`; no release/tag was created.

## Implemented behavior

Replaced the platform broker environment boundary with
`internal/projectconfiguration`, which validates trusted `DEX_PROJECT_*`
project/Live/Preview scope, exact object key/version, prefixed SHA-256 digest,
versioned storage location, hosted KMS key ARN, explicit local endpoint opt-in,
and the actual public URL. The template still declares zero connectors. The
latest approved startup now uses the shared SDK for every project deployment,
including connector-free configuration and private application environment.
Local developer configuration remains separate.
Obsolete platform connector/broker variables fail closed.

All project startups call the shared Connector SDK loader to fetch and verify
the exact snapshot, resolve the exact application secret versions, and apply
settings before clients, Workers or goroutines are created. Apps adding
connectors use its typed provider adapter to resolve current credentials. The
tracked `sdkgo v0.14.2` is a real published baseline, but the additive package is
not yet in that release. Local tests use the official SDK worktree; production
checks are blocked pending an authorized release and exact pin update. This
work does not invent an unpublished dependency pin.

The application-manifest generator now requires exact official `modulePath` and
released version, accepts static Trigger binding identities, rejects duplicate
or value-bearing bindings, and always emits a canonical sorted `triggerBindings`
array. Trigger-only connections may use `operations: []`; both lists cannot be
empty. The checked-in empty connector set remains unchanged.

## Generic application environment follow-up

Added optional `application.environment` declarations for application-owned
startup strings. The generator validates unique safe names, required/secret
classification, Unicode minimum lengths, bounded UTF-8 enum strings, and
reserved process/infrastructure/proxy names. It rejects values, defaults,
secret references, duplicate enum values and secret enums. Nonempty lists are
sorted and contain every canonical declaration field; absent/empty lists are
omitted to preserve the existing empty contract.

The checked-in template has no environment declarations. Its startup nevertheless
uses `projectconfig.LoadFromEnvironment`, then
`loaded.ResolveApplicationEnvironment(ctx)` and `environment.Apply()` before
constructing clients or starting goroutines. The dependency is used by actual
startup; no platform configuration file or broker is involved. Application
settings must never be read from package variable initializers or `init`.

The earlier declaration-only revision passed complete `make check`, recorded in
`/private/tmp/sv2-template-environment-check.log`. The subsequent unified-loader
revision changes dependency requirements; that earlier result is not validation
of the final source. Current verification is recorded below.

## Dex Flow Changes

None. The existing BasicProcess Flow, Steps, Attributes, Channel, Timer, RPCs,
input/output, and retention behavior remain unchanged. Strict FDG 2.0 rendering
completed with `valid: true` and no diagnostics.

## Database Schema Changes

None. No database, credential store, projection, or business owner was added.
Project Dex Web owns mutable configuration; the application consumes an
immutable deployment reference.

## UI/UX

No product UI, OpenAPI operation, or generated-client contract changed. The
existing approval/reminder browser journey remains the template regression.
Connector/OAuth configuration belongs to project Dex Web.

## Tests

The final required `make check` was attempted with the isolated workspace and
failed at `go mod tidy -diff`: published `sdkgo v0.14.2` does not contain
`projectconfig`. The check was not skipped or weakened. Log:
`/private/tmp/sv2-template-unified-loader-make-check.log`.

The explicit ignored workspace `.local/projectconfig.work` selects this template,
its existing OpenAPI tools module and the official local SDK worktree. With that
workspace, the following commands passed:

```sh
GOWORK=/private/tmp/sv2-template-project-config/.local/projectconfig.work go test -race ./internal/projectconfiguration ./internal/templatecontract
GOWORK=/private/tmp/sv2-template-project-config/.local/projectconfig.work go vet ./...
GOWORK=/private/tmp/sv2-template-project-config/.local/projectconfig.work make test-unit test-integration test-e2e build
```

These cover all Go tests, thirteen Python tests, five UI tests, real isolated Dex
integration, one production browser journey (2.7 seconds; 3.1-second runner), and
frontend/Go production builds. Strict FDG 2.0 and formatting passed before the
required module check stopped. The local check log is
`/private/tmp/sv2-template-unified-loader-local-check.log`.

New startup regressions exercise the actual shared SDK against a versioned S3
HTTP fixture. Live and Preview connector-free snapshots apply ordinary values
and exact private secret versions before startup returns. Snapshot digest,
secret digest, secret version and missing-secret failures leave all process
settings unchanged and never expose secret values in errors. The fixture
rejects every mutable read, wrong-scope read and write/provider dispatch.
These fixtures do not replace the upstream real MinIO tests or full Kind E2E.

The installed CLI was `dexcli v0.14.2`, a permitted stable patch of the template's
`cli-v0.14.1` baseline; committed CLI/Server baselines and Go SDK `v0.13.1` pins
were preserved. The fixture script created separate temporary Dex stacks and
cleaned them after successful integration/E2E runs. Generated OpenAPI outputs,
web assets, binaries, and node_modules remain ignored.

Dependency bootstrap reported existing npm audit findings; no dependency
upgrades or audit auto-fixes were performed in this contract change.

## Documentation

Updated `AGENTS.md`, `README.md`, `.superverse/template.json`, and template
contract assertions together. The README documents every standard environment
input, exact Preview/Live scope keys, the unified application loader, and release
requirements for Connector SDK consumers.

## Remaining acceptance

`blocked=true` for this template's required `make check` until the shared SDK
package is published and its exact dependency pin is updated. This is not evidence
of the complete Superverse Local Kind project/Agent/Preview/Release/Live journeys,
real provider OAuth, hosted KMS/IAM, or AWS deployment. Upstream publication and
consumer pin updates were not performed.
