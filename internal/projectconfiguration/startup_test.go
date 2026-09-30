// Copyright (c) 2026 Super Durable
// SPDX-License-Identifier: MIT

package projectconfiguration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/superdurable/dex-connectors-library/sdkgo/projectconfig"
)

func TestProjectStartupLoadsAndAppliesExactEnvironmentWithoutConnectors(t *testing.T) {
	for _, mode := range []string{"live", "preview", "snapshot digest mismatch", "secret digest mismatch", "secret version mismatch", "secret unavailable"} {
		t.Run(mode, func(t *testing.T) {
			clearEnvironment(t)
			t.Setenv("APP_ENV", "before-startup")
			t.Setenv("SIGNING_SECRET", "before-startup")
			scope := projectconfig.Scope{ProjectID: "a7k2", Kind: "live"}
			if mode == "preview" {
				scope.Kind, scope.SessionID = "preview", "s8m4"
			}
			prefix, err := scope.Prefix()
			if err != nil {
				t.Fatal(err)
			}
			secretKey := prefix + "/app-secrets/SIGNING_SECRET/" + strings.Repeat("a", 32)
			secret := strings.Repeat("test-private-signing-value-", 2)
			secretBody, err := json.Marshal(map[string]any{"scope": scope, "name": "SIGNING_SECRET", "value": secret})
			if err != nil {
				t.Fatal(err)
			}
			secretDigest := startupDigest(secretBody)
			if mode == "secret digest mismatch" {
				secretDigest = "sha256:" + strings.Repeat("b", 64)
			}
			ordinary := "production"
			document := projectconfig.Configuration{SchemaVersion: projectconfig.ConfigurationSchemaVersion, Scope: scope, Revision: 7,
				Connections: []projectconfig.ConnectionConfiguration{}, TriggerBindings: []projectconfig.TriggerConfiguration{}, OperationConfigurations: []projectconfig.OperationConfiguration{},
				Environment: map[string]projectconfig.EnvironmentValue{
					"APP_ENV":        {Value: &ordinary},
					"SIGNING_SECRET": {SecretRef: &projectconfig.ApplicationSecretRef{Key: secretKey, Version: "secret-version-1", Digest: secretDigest}},
				}}
			contents, err := json.Marshal(document)
			if err != nil {
				t.Fatal(err)
			}
			var snapshotReads, secretReads atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Error("startup must not mutate storage or dispatch provider operations")
					http.Error(w, "unexpected write", http.StatusForbidden)
					return
				}
				switch {
				case r.URL.Path == "/project-private" && r.URL.Query().Has("versioning"):
					w.Header().Set("Content-Type", "application/xml")
					fmt.Fprint(w, `<VersioningConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Status>Enabled</Status></VersioningConfiguration>`)
				case r.URL.Path == "/project-private/test/"+prefix+"/configuration/head" && r.URL.Query().Get("versionId") == "configuration-version-7":
					snapshotReads.Add(1)
					w.Header().Set("X-Amz-Version-Id", "configuration-version-7")
					w.Header().Set("ETag", `"snapshot-etag"`)
					w.Write(contents)
				case r.URL.Path == "/project-private/test/"+secretKey && r.URL.Query().Get("versionId") == "secret-version-1":
					secretReads.Add(1)
					if mode == "secret unavailable" {
						http.Error(w, "not found", http.StatusNotFound)
						return
					}
					version := "secret-version-1"
					if mode == "secret version mismatch" {
						version = "secret-version-2"
					}
					w.Header().Set("X-Amz-Version-Id", version)
					w.Header().Set("ETag", `"secret-etag"`)
					w.Write(secretBody)
				default:
					t.Errorf("unexpected scope or mutable object read: %s", r.URL)
					http.Error(w, "unexpected read", http.StatusForbidden)
				}
			}))
			defer server.Close()
			digest := startupDigest(contents)
			if mode == "snapshot digest mismatch" {
				digest = "sha256:" + strings.Repeat("c", 64)
			}
			for name, value := range map[string]string{
				"DEX_PROJECT_ID": scope.ProjectID, "DEX_PROJECT_SCOPE": scope.Kind, "DEX_PROJECT_SESSION_ID": scope.SessionID,
				"DEX_PROJECT_CONFIG_KEY": prefix + "/configuration/head", "DEX_PROJECT_CONFIG_VERSION": "configuration-version-7", "DEX_PROJECT_CONFIG_DIGEST": digest,
				"DEX_PROJECT_STORAGE_BUCKET": "project-private", "DEX_PROJECT_STORAGE_PREFIX": "test/", "DEX_PROJECT_STORAGE_ENDPOINT": server.URL, "DEX_PROJECT_ALLOW_LOCAL_STORAGE": "true",
				"PUBLIC_BASE_URL": "https://example.test", "AWS_ACCESS_KEY_ID": "fixture-access", "AWS_SECRET_ACCESS_KEY": "fixture-secret", "AWS_SESSION_TOKEN": "", "AWS_REGION": "us-east-1", "AWS_EC2_METADATA_DISABLED": "true",
			} {
				t.Setenv(name, value)
			}
			configuration, err := Load(false)
			if mode == "live" || mode == "preview" {
				if err != nil || configuration.Mode != ModeProject {
					t.Fatalf("connector-free startup failed: %v", err)
				}
				if os.Getenv("APP_ENV") != ordinary || os.Getenv("SIGNING_SECRET") != secret || snapshotReads.Load() != 1 || secretReads.Load() != 1 {
					t.Fatal("accepted ordinary and private values were not resolved before startup returned")
				}
			} else {
				if err == nil {
					t.Fatal("invalid accepted configuration started application")
				}
				if strings.Contains(err.Error(), secret) || os.Getenv("APP_ENV") != "before-startup" || os.Getenv("SIGNING_SECRET") != "before-startup" {
					t.Fatal("failed startup leaked a secret or partially applied settings")
				}
			}
		})
	}
}

func startupDigest(value []byte) string {
	digest := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(digest[:])
}
