// Copyright (c) 2026 Super Durable
// SPDX-License-Identifier: MIT

package projectconfiguration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmptyTemplateNeedsNoConnectorStore(t *testing.T) {
	clearEnvironment(t)
	configuration, err := Load(false)
	if err != nil || configuration.Mode != ModeLocal {
		t.Fatalf("empty template: %v", err)
	}
}
func TestProjectScopeAndImmutableReference(t *testing.T) {
	for _, scope := range []string{"live", "preview"} {
		t.Run(scope, func(t *testing.T) {
			configureProject(t, scope)
			configuration, err := validateProjectReference()
			if err != nil {
				t.Fatal(err)
			}
			if configuration.Mode != ModeProject || configuration.ProjectID != "a7k2" || configuration.PublicBaseURL != "https://example.test" {
				t.Fatalf("wrong project references: %+v", configuration)
			}
		})
	}
}
func TestProjectBoundaryFailsClosed(t *testing.T) {
	for _, test := range []struct{ name, key, value string }{
		{"wrong project", "DEX_PROJECT_ID", "other"}, {"missing preview owner", "DEX_PROJECT_SCOPE", "preview"}, {"extra Live owner", "DEX_PROJECT_SESSION_ID", "session"},
		{"missing version", "DEX_PROJECT_CONFIG_VERSION", ""}, {"mutable null version", "DEX_PROJECT_CONFIG_VERSION", "null"}, {"unprefixed digest", "DEX_PROJECT_CONFIG_DIGEST", strings.Repeat("a", 64)},
		{"unscoped key", "DEX_PROJECT_CONFIG_KEY", "somewhere/configuration/head"}, {"unsafe prefix", "DEX_PROJECT_STORAGE_PREFIX", "../other"},
		{"missing KMS", "DEX_PROJECT_STORAGE_KMS_KEY_ARN", ""}, {"KMS alias", "DEX_PROJECT_STORAGE_KMS_KEY_ARN", "alias/project"},
		{"unexpected endpoint", "DEX_PROJECT_STORAGE_ENDPOINT", "http://127.0.0.1:29000"}, {"invalid URL", "PUBLIC_BASE_URL", "https://user:secret@example.test"},
		{"mixed local mode", "DEX_CONNECTOR_CONFIG_FILE", "/tmp/unsafe"}, {"old broker", "SUPERVERSE_CONNECTOR_BROKER_URL", "https://obsolete"},
	} {
		t.Run(test.name, func(t *testing.T) {
			configureProject(t, "live")
			t.Setenv(test.key, test.value)
			if _, err := Load(false); err == nil {
				t.Fatal("expected invalid boundary rejection")
			}
		})
	}
}
func TestExplicitLocalStorage(t *testing.T) {
	configureProject(t, "preview")
	t.Setenv("DEX_PROJECT_ALLOW_LOCAL_STORAGE", "true")
	t.Setenv("DEX_PROJECT_STORAGE_KMS_KEY_ARN", "")
	t.Setenv("DEX_PROJECT_STORAGE_ENDPOINT", "http://project-minio.sv2-project-a7k2.svc.cluster.local:9000")
	if _, err := validateProjectReference(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEX_PROJECT_STORAGE_ENDPOINT", "http://example.com")
	if _, err := validateProjectReference(); err == nil {
		t.Fatal("public endpoint cannot become local storage")
	}
}
func TestLocalDeveloperConfiguration(t *testing.T) {
	clearEnvironment(t)
	path := filepath.Join(t.TempDir(), "connections.json")
	if err := os.WriteFile(path, []byte(`{"connections":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEX_CONNECTOR_CONFIG_FILE", path)
	configuration, err := Load(true)
	if err != nil || configuration.ConfigurationFile != path {
		t.Fatalf("local configuration: %v", err)
	}
}
func configureProject(t *testing.T, scope string) {
	t.Helper()
	clearEnvironment(t)
	prefix := "projects/a7k2/" + scope
	session := ""
	if scope == "preview" {
		session = "s8m4"
		prefix += "/" + session
	}
	for key, value := range map[string]string{"DEX_PROJECT_ID": "a7k2", "DEX_PROJECT_SCOPE": scope, "DEX_PROJECT_SESSION_ID": session, "DEX_PROJECT_CONFIG_KEY": prefix + "/configuration/head", "DEX_PROJECT_CONFIG_VERSION": "immutable-version", "DEX_PROJECT_CONFIG_DIGEST": "sha256:" + strings.Repeat("a", 64), "DEX_PROJECT_STORAGE_BUCKET": "project-private", "DEX_PROJECT_STORAGE_PREFIX": "test/", "DEX_PROJECT_STORAGE_KMS_KEY_ARN": "arn:aws:kms:us-east-1:123456789012:key/example", "PUBLIC_BASE_URL": "https://example.test/"} {
		t.Setenv(key, value)
	}
}
func clearEnvironment(t *testing.T) {
	t.Helper()
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "DEX_PROJECT_") {
			t.Setenv(key, "")
		}
	}
	for _, key := range []string{"DEX_CONNECTOR_CONFIG_FILE", "SUPERVERSE_CONNECTOR_CONFIG_FILE", "SUPERVERSE_CONNECTOR_CONFIG_DIGEST", "SUPERVERSE_CONNECTOR_BROKER_URL", "SUPERVERSE_CONNECTOR_WORKLOAD_CREDENTIAL_FILE", "PUBLIC_BASE_URL"} {
		t.Setenv(key, "")
	}
}
