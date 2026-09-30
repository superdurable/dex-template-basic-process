#!/usr/bin/env python3
# Copyright (c) 2026 Super Durable
# SPDX-License-Identifier: MIT

from __future__ import annotations

import json
import importlib.util
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest import mock


SCRIPT = Path(__file__).with_name("generate-release-artifacts.py")
ROOT = SCRIPT.parents[1]
SPEC = importlib.util.spec_from_file_location("release_artifacts", SCRIPT)
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class GenerateReleaseArtifactsTest(unittest.TestCase):
    def test_generates_release_bound_contracts_from_a_real_fdg(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            output = Path(temporary)
            environment = os.environ.copy()
            environment.update({
                "SUPERVERSE_RELEASE_ID": "11111111-1111-4111-8111-111111111111",
                "SUPERVERSE_PROJECT_ID": "22222222-2222-4222-8222-222222222222",
                "SUPERVERSE_SOURCE_COMMIT_SHA": "a" * 40,
                "SUPERVERSE_BUILD_PROFILE_DIGEST": "b" * 64,
            })
            result = subprocess.run(
                ["python3", str(SCRIPT), "--output-directory", str(output)],
                cwd=ROOT,
                env=environment,
                check=False,
                capture_output=True,
                text=True,
            )

            self.assertEqual(result.returncode, 0, result.stderr)
            bundle = json.loads((output / "flow-definitions.json").read_text(encoding="utf-8"))
            self.assertEqual(bundle["schemaVersion"], "superverse.dev/flow-definition-bundle/v1")
            self.assertEqual(bundle["releaseId"], environment["SUPERVERSE_RELEASE_ID"])
            self.assertEqual(bundle["projectId"], environment["SUPERVERSE_PROJECT_ID"])
            self.assertEqual(len(bundle["flowDefinitions"]), 1)
            definition = bundle["flowDefinitions"][0]
            self.assertEqual(definition["flowType"], "process.BasicProcessFlow")
            self.assertTrue(definition["valid"])
            self.assertEqual(definition["graph"]["schemaVersion"], "2.0")
            self.assertRegex(definition["digest"], r"^sha256:[0-9a-f]{64}$")
            connector_contract = json.loads(
                (output / "connector-contract.json").read_text(encoding="utf-8")
            )
            self.assertEqual(connector_contract["connections"], [])
            environment_contract = json.loads(
                (output / "environment-contract.json").read_text(encoding="utf-8")
            )
            self.assertFalse(
                environment_contract["application"]["connectorConfigurationRequired"]
            )
            self.assertEqual(
                (output / "dex-app.yaml").read_bytes(),
                (ROOT / "dex-app.yaml").read_bytes(),
            )

    def test_connector_contract_canonicalizes_explicit_trigger_bindings(self) -> None:
        connection = self.connection()
        connection["triggerBindings"] = [
            {"triggerName": "checkoutSessionUpdated", "bindingName": "second"},
            {"triggerName": "checkoutSessionUpdated", "bindingName": "first"},
        ]
        contract = MODULE.connector_contract({"connectors": [connection]})[0]
        self.assertEqual(contract["modulePath"], connection["modulePath"])
        self.assertEqual([binding["bindingName"] for binding in contract["triggerBindings"]], ["first", "second"])
        self.assertEqual(contract["operations"], ["createACHCheckoutSession", "getCheckoutSession"])
        connection.pop("triggerBindings")
        self.assertEqual(MODULE.connector_contract({"connectors": [connection]})[0]["triggerBindings"], [])
        connection["operations"] = []
        connection["triggerBindings"] = [{"triggerName": "checkoutSessionUpdated", "bindingName": "payments"}]
        self.assertEqual(MODULE.connector_contract({"connectors": [connection]})[0]["operations"], [])

    def test_connector_contract_rejects_unsafe_or_ambiguous_declarations(self) -> None:
        cases = [
            {"modulePath": "github.com/untrusted/stripe"},
            {"modulePath": "github.com/superdurable/dex-connectors-library/connectors/../stripe"},
            {"version": "main"},
            {"operations": [], "triggerBindings": []},
            {"triggerBindings": [{"triggerName": "checkoutSessionUpdated", "bindingName": "payments", "configuration": {"secret": "forbidden"}}]},
            {"triggerBindings": [{"triggerName": "checkoutSessionUpdated", "bindingName": "payments"}] * 2},
        ]
        for patch in cases:
            with self.subTest(patch=patch), self.assertRaises(SystemExit):
                MODULE.connector_contract({"connectors": [{**self.connection(), **patch}]})
        missing_module = self.connection()
        missing_module.pop("modulePath")
        with self.assertRaises(SystemExit):
            MODULE.connector_contract({"connectors": [missing_module]})

    @staticmethod
    def connection() -> dict[str, object]:
        return {
            "connectorId": "stripe", "connectionName": "payments",
            "modulePath": "github.com/superdurable/dex-connectors-library/connectors/stripe",
            "version": "v0.2.2", "authMethodId": "stripe-secret-key-webhook",
            "operations": ["getCheckoutSession", "createACHCheckoutSession", "getCheckoutSession"],
        }

    def test_environment_declarations_are_canonical_and_value_free(self) -> None:
        manifest = {"connectors": [], "application": {"port": 8080, "healthPath": "/healthz", "environment": [
            {"name": "EVENT_TOKEN_SECRET", "secret": True, "required": True, "minLength": 32},
            {"name": "APP_ENV", "enum": ["production", "development"]},
        ]}}
        contract = MODULE.environment_contract(manifest)
        self.assertEqual(contract["environment"], [
            {"enum": ["development", "production"], "minLength": 0, "name": "APP_ENV", "required": False, "secret": False},
            {"enum": [], "minLength": 32, "name": "EVENT_TOKEN_SECRET", "required": True, "secret": True},
        ])
        manifest["application"]["environment"] = []
        self.assertNotIn("environment", MODULE.environment_contract(manifest))
        manifest["application"].pop("environment")
        self.assertNotIn("environment", MODULE.environment_contract(manifest))

    def test_environment_declarations_reject_values_process_control_and_ambiguity(self) -> None:
        cases = [
            [{"name": "TOKEN", "value": "private"}], [{"name": "TOKEN", "default": "private"}],
            [{"name": "TOKEN", "secretRef": {}}], [{"name": "TOKEN"}, {"name": "TOKEN"}],
            [{"name": "TOKEN", "secret": True, "enum": ["private"]}],
            [{"name": "TOKEN", "required": 1}], [{"name": "TOKEN", "minLength": True}],
            [{"name": "TOKEN", "minLength": -1}], [{"name": "TOKEN", "minLength": 32769}],
            [{"name": "TOKEN", "enum": ["a", "a"]}], [{"name": "TOKEN", "enum": ["\x00"]}],
            [{"name": "TOKEN", "enum": ["\ud800"]}], [{"name": "TOKEN", "minLength": 2, "enum": ["a"]}],
            [{"name": "TOKEN", "enum": ["x" * 32769]}], [{"name": "TOKEN", "enum": [1]}],
            [{"name": "TOKEN", "enum": None}], [{"name": "A" * 129}],
            [{"name": "FIELD_" + str(index)} for index in range(129)],
        ]
        for name in ["DEX_PROJECT_ID", "AWS_ACCESS_KEY_ID", "SUPERVERSE_PLATFORM", "LD_PRELOAD", "GODEBUG", "GIT_CONFIG_COUNT", "PATH", "HOME", "PORT", "HOST", "NODE_OPTIONS", "SSL_CERT_FILE", "SSL_CERT_DIR", "PUBLIC_BASE_URL", "HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY"]:
            cases.append([{"name": name}])
        for declarations in cases:
            with self.subTest(declarations=declarations), self.assertRaises(SystemExit):
                MODULE.validated_environment_declarations(declarations)
        # Unicode minima count code points; the independent storage bound counts UTF-8 bytes.
        self.assertEqual(MODULE.validated_environment_declarations([{"name": "LABEL", "minLength": 2, "enum": ["中文"]}])[0]["enum"], ["中文"])

    def test_rejects_a_missing_release_identity(self) -> None:
        with tempfile.TemporaryDirectory() as temporary, mock.patch.dict(
            os.environ,
            {
                "SUPERVERSE_RELEASE_ID": "",
                "SUPERVERSE_PROJECT_ID": "",
                "SUPERVERSE_SOURCE_COMMIT_SHA": "",
                "SUPERVERSE_BUILD_PROFILE_DIGEST": "",
            },
            clear=False,
        ):
            result = subprocess.run(
                ["python3", str(SCRIPT), "--output-directory", temporary],
                cwd=ROOT,
                env=os.environ.copy(),
                check=False,
                capture_output=True,
                text=True,
            )

            self.assertNotEqual(result.returncode, 0)
            self.assertIn("SUPERVERSE_RELEASE_ID is required", result.stderr)


if __name__ == "__main__":
    unittest.main()
