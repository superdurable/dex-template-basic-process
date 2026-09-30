#!/usr/bin/env python3
"""Validate template-repository automation; never an exported app check."""

from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
REQUIRED = {
    ".github/workflows/update-dex-dependencies.yml": (
        "schedule:", "workflow_dispatch:", "pull-requests: write",
        "automation/update-dex-dependencies", "gh pr create", "gh workflow run ci.yml",
    ),
    ".github/workflows/ci.yml": (
        "workflow_dispatch:", "scripts/check-template-version.py",
        "gh release create", "contents: write",
        ".github/scripts/check-template-maintenance.py",
    ),
}


def main():
    for name, required in REQUIRED.items():
        contents = (ROOT / name).read_text()
        for fragment in required:
            if fragment not in contents:
                raise SystemExit(f"{name} is missing template-maintenance requirement {fragment!r}")
    print("Template repository maintenance checks passed")


if __name__ == "__main__":
    main()
