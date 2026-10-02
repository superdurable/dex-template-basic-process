#!/usr/bin/env bash
set -euo pipefail

# One source of truth for both authoring checks and immutable release artifacts.
# Validate every source declared by dex-app.yaml, including added or renamed Flows.
exec python3 ./scripts/generate-release-artifacts.py --check-only
