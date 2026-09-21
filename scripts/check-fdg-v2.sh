#!/usr/bin/env bash
set -euo pipefail

dexcli_binary="${DEXCLI:-dexcli}"
output_base="$(mktemp "${TMPDIR:-/tmp}/basic-process-fdg-v2.XXXXXX")"
output_json="${output_base}.json"
cleanup() {
  rm -f -- "${output_base}" "${output_json}"
}
trap cleanup EXIT

"${dexcli_binary}" visualize internal/process/flow.go \
  --schema-version 2.0 \
  --json \
  --out "${output_base}"

python3 - "${output_json}" <<'PY'
import json
import sys

path = sys.argv[1]
document = json.load(open(path))
if document.get("valid") is not True:
    diagnostics = json.dumps(document.get("diagnostics", []), indent=2)
    raise SystemExit(f"FDG 2.0 graph is invalid:\n{diagnostics}")
if document.get("diagnostics"):
    raise SystemExit(f"FDG 2.0 graph has diagnostics: {document['diagnostics']}")
print("validated FDG 2.0 graph")
PY
