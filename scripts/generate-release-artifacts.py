#!/usr/bin/env python3
# Copyright (c) 2026 Super Durable
# SPDX-License-Identifier: MIT

"""Generate immutable Superverse release contracts from checked-in application sources."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
from pathlib import Path
import subprocess
import tempfile


ROOT = Path(__file__).resolve().parents[1]
DEX_APP = ROOT / "dex-app.yaml"
DEX_APP_SCHEMA = "superverse.dev/dex-app/v1"
BUNDLE_SCHEMA = "superverse.dev/flow-definition-bundle/v1"
CONNECTOR_CONTRACT_SCHEMA = "superverse.dev/connector-contract/v1"
ENVIRONMENT_CONTRACT_SCHEMA = "superverse.dev/environment-contract/v1"
CONNECTOR_KEYS = {
    "connectionName",
    "connectorId",
    "modulePath",
    "version",
    "authMethodId",
    "operations",
}
OPTIONAL_CONNECTOR_KEYS = {"triggerBindings"}
MODULE_PATH = re.compile(r"^github\.com/superdurable/dex-connectors-library/connectors/[a-z][a-z0-9-]*(?:/[a-z][a-z0-9-]*)*$")
CAPABILITY_NAME = re.compile(r"^[a-z][A-Za-z0-9]+$")
ENVIRONMENT_NAME = re.compile(r"^[A-Z][A-Z0-9_]{0,127}$")
ENVIRONMENT_KEYS = {"name", "required", "secret", "minLength", "enum"}
RESERVED_ENVIRONMENT_PREFIXES = ("SUPERVERSE_", "DEX_", "AWS_", "LD_", "GO", "GIT_")
RESERVED_ENVIRONMENT_NAMES = {
    "PATH", "HOME", "PORT", "HOST", "NODE_OPTIONS", "SSL_CERT_FILE", "SSL_CERT_DIR",
    "PUBLIC_BASE_URL", "HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY",
}
CONNECTOR_VERSION = re.compile(r"^v(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)$")


def main() -> None:
    arguments = parse_arguments()
    release_id = required_environment("SUPERVERSE_RELEASE_ID")
    project_id = required_environment("SUPERVERSE_PROJECT_ID")
    source_commit_sha = required_environment("SUPERVERSE_SOURCE_COMMIT_SHA")
    build_profile_digest = required_environment("SUPERVERSE_BUILD_PROFILE_DIGEST")
    manifest = load_manifest()
    output_directory = arguments.output_directory.resolve()
    output_directory.mkdir(parents=True, exist_ok=True)
    definitions = render_flow_definitions(manifest, arguments.dexcli)
    write_json(output_directory / "flow-definitions.json", {
        "schemaVersion": BUNDLE_SCHEMA,
        "releaseId": release_id,
        "projectId": project_id,
        "sourceCommitSha": source_commit_sha,
        "buildProfileDigest": build_profile_digest,
        "flowDefinitions": definitions,
    })
    write_json(output_directory / "connector-contract.json", {
        "schemaVersion": CONNECTOR_CONTRACT_SCHEMA,
        "releaseId": release_id,
        "projectId": project_id,
        "connections": connector_contract(manifest),
    })
    write_json(output_directory / "environment-contract.json", {
        "schemaVersion": ENVIRONMENT_CONTRACT_SCHEMA,
        "releaseId": release_id,
        "projectId": project_id,
        "application": environment_contract(manifest),
    })
    (output_directory / "dex-app.yaml").write_bytes(DEX_APP.read_bytes())


def parse_arguments() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--output-directory", type=Path, required=True)
    parser.add_argument("--dexcli", default=os.environ.get("DEXCLI", "dexcli"))
    return parser.parse_args()


def required_environment(name: str) -> str:
    value = os.environ.get(name, "").strip()
    if not value:
        raise SystemExit(f"{name} is required")
    return value


def load_manifest() -> dict[str, object]:
    try:
        manifest = json.loads(DEX_APP.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as failure:
        raise SystemExit("dex-app.yaml must contain valid JSON-compatible YAML") from failure
    if not isinstance(manifest, dict) or manifest.get("schemaVersion") != DEX_APP_SCHEMA:
        raise SystemExit("dex-app.yaml has an unsupported schemaVersion")
    flow_definitions = manifest.get("flowDefinitions")
    connectors = manifest.get("connectors")
    application = manifest.get("application")
    if not isinstance(flow_definitions, list) or not flow_definitions:
        raise SystemExit("dex-app.yaml must declare at least one Flow Definition")
    if not isinstance(connectors, list) or not isinstance(application, dict):
        raise SystemExit("dex-app.yaml connectors and application must be declared")
    return manifest


def render_flow_definitions(
    manifest: dict[str, object],
    dexcli: str,
) -> list[dict[str, object]]:
    definitions: list[dict[str, object]] = []
    flow_types: set[str] = set()
    for entry in manifest["flowDefinitions"]:
        if not isinstance(entry, dict) or set(entry) != {"sourcePath"}:
            raise SystemExit("each Flow Definition entry must contain only sourcePath")
        source_path = safe_relative_path(entry.get("sourcePath"), "Flow Definition sourcePath")
        source = ROOT / source_path
        if not source.is_file() or source.suffix != ".go":
            raise SystemExit(f"Flow Definition source does not exist: {source_path}")
        with tempfile.TemporaryDirectory(prefix="release-fdg-") as temporary:
            output_base = Path(temporary) / "flow"
            result = subprocess.run(
                [
                    dexcli,
                    "visualize",
                    source_path,
                    "--schema-version",
                    "2.0",
                    "--json",
                    "--out",
                    str(output_base),
                ],
                cwd=ROOT,
                check=False,
                capture_output=True,
                text=True,
            )
            if result.returncode != 0:
                raise SystemExit(
                    f"FDG rendering failed for {source_path}: "
                    + (result.stderr or result.stdout).strip()
                )
            try:
                graph = json.loads(output_base.with_suffix(".json").read_text(encoding="utf-8"))
            except (OSError, json.JSONDecodeError) as failure:
                raise SystemExit(f"FDG renderer returned invalid JSON for {source_path}") from failure
        if (not isinstance(graph, dict)
                or graph.get("schemaVersion") != "2.0"
                or graph.get("valid") is not True
                or graph.get("diagnostics")):
            raise SystemExit(f"FDG 2.0 graph is invalid for {source_path}")
        flow = graph.get("flow")
        flow_type = flow.get("name") if isinstance(flow, dict) else None
        if not isinstance(flow_type, str) or not flow_type or flow_type in flow_types:
            raise SystemExit("Flow Types must be non-empty and unique")
        flow_types.add(flow_type)
        definitions.append({
            "flowType": flow_type,
            "sourcePath": source_path,
            "schemaVersion": "2.0",
            "digest": "sha256:" + sha256(canonical_json(graph)),
            "valid": True,
            "graph": graph,
        })
    return sorted(definitions, key=lambda definition: definition["flowType"])


def connector_contract(manifest: dict[str, object]) -> list[dict[str, object]]:
    connections: list[dict[str, object]] = []
    connection_names: set[str] = set()
    for entry in manifest["connectors"]:
        if (not isinstance(entry, dict) or not CONNECTOR_KEYS.issubset(entry)
                or not set(entry).issubset(CONNECTOR_KEYS | OPTIONAL_CONNECTOR_KEYS)):
            raise SystemExit("each connector entry must use the exact connector contract fields")
        for key in ("connectionName", "connectorId", "modulePath", "version", "authMethodId"):
            if not isinstance(entry.get(key), str) or not entry[key]:
                raise SystemExit(f"connector {key} must be a non-empty string")
        if not MODULE_PATH.fullmatch(entry["modulePath"]):
            raise SystemExit("connector modulePath must name an exact official connector module")
        if not CONNECTOR_VERSION.fullmatch(entry["version"]):
            raise SystemExit("connector version must be an exact stable v-prefixed release")
        operations = entry.get("operations")
        if (not isinstance(operations, list)
                or any(not isinstance(value, str) or not CAPABILITY_NAME.fullmatch(value) for value in operations)):
            raise SystemExit("connector operations must contain exact capability names")
        trigger_bindings = validated_trigger_bindings(entry.get("triggerBindings", []))
        if not operations and not trigger_bindings:
            raise SystemExit("connector must declare an operation or Trigger binding")
        if entry["connectionName"] in connection_names:
            raise SystemExit("connector connectionName values must be unique")
        connection_names.add(entry["connectionName"])
        connections.append({**entry, "operations": sorted(set(operations)), "triggerBindings": trigger_bindings})
    return sorted(connections, key=lambda connection: connection["connectionName"])


def validated_trigger_bindings(value: object) -> list[dict[str, str]]:
    if not isinstance(value, list):
        raise SystemExit("connector triggerBindings must be a list")
    bindings: list[dict[str, str]] = []
    identities: set[tuple[str, str]] = set()
    for binding in value:
        if not isinstance(binding, dict) or set(binding) != {"triggerName", "bindingName"}:
            raise SystemExit("Trigger bindings contain only triggerName and bindingName")
        trigger_name, binding_name = binding["triggerName"], binding["bindingName"]
        if (not isinstance(trigger_name, str) or not CAPABILITY_NAME.fullmatch(trigger_name)
                or not isinstance(binding_name, str) or not binding_name.strip()
                or binding_name != binding_name.strip() or len(binding_name) > 256
                or any(ord(character) < 32 for character in binding_name)):
            raise SystemExit("Trigger binding identity is invalid")
        identity = (trigger_name, binding_name)
        if identity in identities:
            raise SystemExit("Trigger binding identities must be unique")
        identities.add(identity)
        bindings.append({"triggerName": trigger_name, "bindingName": binding_name})
    return sorted(bindings, key=lambda binding: (binding["triggerName"], binding["bindingName"]))


def environment_contract(manifest: dict[str, object]) -> dict[str, object]:
    application = manifest["application"]
    if not {"port", "healthPath"}.issubset(application) or not set(application).issubset({"port", "healthPath", "environment"}):
        raise SystemExit("dex-app.yaml application must contain port, healthPath, and optional environment declarations")
    port = application.get("port")
    health_path = application.get("healthPath")
    if not isinstance(port, int) or isinstance(port, bool) or port < 1 or port > 65535:
        raise SystemExit("application port must be between 1 and 65535")
    if (not isinstance(health_path, str)
            or not health_path.startswith("/")
            or "?" in health_path
            or "#" in health_path):
        raise SystemExit("application healthPath must be an absolute path")
    contract = {
        "port": port,
        "healthPath": health_path,
        "publicBaseUrlRequired": True,
        "connectorConfigurationRequired": bool(manifest["connectors"]),
    }
    declarations = validated_environment_declarations(application.get("environment", []))
    if declarations:
        contract["environment"] = declarations
    return contract


def validated_environment_declarations(value: object) -> list[dict[str, object]]:
    if not isinstance(value, list) or len(value) > 128:
        raise SystemExit("application environment must contain at most 128 declarations")
    declarations: list[dict[str, object]] = []
    names: set[str] = set()
    for field in value:
        if not isinstance(field, dict) or "name" not in field or not set(field).issubset(ENVIRONMENT_KEYS):
            raise SystemExit("application environment contains only declaration fields; values and defaults are forbidden")
        name = field["name"]
        if (not isinstance(name, str) or not ENVIRONMENT_NAME.fullmatch(name) or name in names
                or name.startswith(RESERVED_ENVIRONMENT_PREFIXES) or name in RESERVED_ENVIRONMENT_NAMES):
            raise SystemExit("application environment name is invalid, reserved, or duplicated")
        names.add(name)
        required, secret = field.get("required", False), field.get("secret", False)
        minimum = field.get("minLength", 0)
        options = field.get("enum", [])
        if (not isinstance(required, bool) or not isinstance(secret, bool)
                or not isinstance(minimum, int) or isinstance(minimum, bool) or minimum < 0 or minimum > 32768
                or not isinstance(options, list) or len(options) > 128 or (secret and options)):
            raise SystemExit("application environment declaration constraints are invalid")
        for option in options:
            if not isinstance(option, str) or "\x00" in option or len(option) < minimum:
                raise SystemExit("application environment enum is invalid")
            try:
                encoded = option.encode("utf-8")
            except UnicodeEncodeError as failure:
                raise SystemExit("application environment enum must be valid UTF-8") from failure
            if len(encoded) > 32768:
                raise SystemExit("application environment enum exceeds the byte limit")
        if len(set(options)) != len(options):
            raise SystemExit("application environment enum values must be unique")
        declarations.append({"enum": sorted(options), "minLength": minimum, "name": name, "required": required, "secret": secret})
    return sorted(declarations, key=lambda field: field["name"])


def safe_relative_path(value: object, label: str) -> str:
    if not isinstance(value, str) or not value:
        raise SystemExit(f"{label} must be a non-empty string")
    path = Path(value)
    if path.is_absolute() or ".." in path.parts or path.as_posix() != value:
        raise SystemExit(f"{label} must be a normalized repository-relative path")
    return value


def canonical_json(value: object) -> bytes:
    return json.dumps(value, ensure_ascii=False, separators=(",", ":")).encode("utf-8")


def sha256(contents: bytes) -> str:
    return hashlib.sha256(contents).hexdigest()


def write_json(path: Path, value: object) -> None:
    path.write_bytes(canonical_json(value) + b"\n")


if __name__ == "__main__":
    main()
