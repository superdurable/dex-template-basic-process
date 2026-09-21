# Local mock server

Use the mock server to validate UI interactions before connecting the frontend
to a real Dex Worker. It implements the application's current OpenAPI contract
with in-memory Go state and never starts Dex.

## Start

```bash
make bootstrap
make mock
```

Open <http://127.0.0.1:8080>. Vite provides hot module replacement and proxies
`/api` plus `/__mock__` to the loopback-only Go mock API.

| Variable | Default | Purpose |
| --- | --- | --- |
| `MOCK_WEB_HOST` | `0.0.0.0` | Vite bind host |
| `MOCK_WEB_PORT` | `8080` | Browser port |
| `MOCK_API_HOST` | `127.0.0.1` | Mock API bind host |
| `MOCK_API_PORT` | `18081` | Mock API port |

## Lifecycle

A new process automatically moves from `started` through `validated` to
`waiting_for_approval`. Approval moves through `approved` and `executing` to
`completed`. The short delays make loading and progress visible without
changing the real fifteen-minute Dex reminder timer.

The server keeps state across browser refreshes. Data exists only in memory and
is discarded by Reset or server restart.

## Mock Controls

The controls appear only when Vite starts with `VITE_MOCK_MODE=true`:

| Control | Behavior |
| --- | --- |
| Fail next Start | The next create request returns a one-time 503. |
| Advance | Stops automatic progression for that Flow and advances one state. |
| Emit reminder | Increments the reminder count while approval is pending. |
| Fail next Refresh | Pauses polling on a one-time 503 and exposes Retry. |
| Fail next Approval | The next approval returns a one-time 503. |
| Reset | Clears all server and browser Flow state. |

The mock-only HTTP surface is `GET /__mock__/control?flowId=...` and
`POST /__mock__/control`. The production server returns 404 for this prefix.

## Verification boundary

Run `make test-mock-e2e` for the mock interaction suite. Before handoff, always
run `make check`; only the real Dex integration and E2E suites prove durable
waits, RPC behavior, Worker replacement, Timer handling, and terminal state.
