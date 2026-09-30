// Copyright (c) 2026 Super Durable
// SPDX-License-Identifier: MIT

// Package projectconfiguration loads accepted project configuration before application startup.
package projectconfiguration

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/superdurable/dex-connectors-library/sdkgo/projectconfig"
)

const maximumConfigurationBytes = 32 << 20

// Mode identifies the active configuration authority.
type Mode string

const (
	// ModeLocal selects a developer-owned Dex configuration file.
	ModeLocal Mode = "local"
	// ModeProject selects project Dex Web's immutable configuration and shared credential store.
	ModeProject Mode = "project"
)

// Configuration contains only safe deployment references; it never contains provider or AWS credentials.
type Configuration struct {
	Mode                 Mode
	ConfigurationFile    string
	ProjectID            string
	Scope                string
	SessionID            string
	ObjectKey            string
	ObjectVersion        string
	ObjectDigest         string
	StorageBucket        string
	StoragePrefix        string
	KMSKeyARN            string
	LocalStorageEndpoint string
	PublicBaseURL        string
}

var identityPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,127}$`)
var digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
var kmsKeyPattern = regexp.MustCompile(`^arn:[a-z0-9-]+:kms:[a-z0-9-]+:[0-9]{12}:key/[a-zA-Z0-9-]+$`)

// Load loads the exact project snapshot and applies its accepted application
// environment, including exact private secret versions, before constructing any
// clients or starting application goroutines. Local development remains explicit.
func Load(requiresProjectConfiguration bool) (Configuration, error) {
	for _, name := range []string{"SUPERVERSE_CONNECTOR_CONFIG_FILE", "SUPERVERSE_CONNECTOR_CONFIG_DIGEST", "SUPERVERSE_CONNECTOR_BROKER_URL", "SUPERVERSE_CONNECTOR_WORKLOAD_CREDENTIAL_FILE"} {
		if os.Getenv(name) != "" {
			return Configuration{}, fmt.Errorf("obsolete platform connector configuration is unsupported; use DEX_PROJECT_* scoped storage")
		}
	}
	localFile := os.Getenv("DEX_CONNECTOR_CONFIG_FILE")
	if hasProjectEnvironment() {
		if localFile != "" {
			return Configuration{}, fmt.Errorf("local and project connector configuration cannot be combined")
		}
		return loadProject()
	}
	if localFile == "" {
		if requiresProjectConfiguration {
			return Configuration{}, fmt.Errorf("DEX_CONNECTOR_CONFIG_FILE or DEX_PROJECT_* configuration is required")
		}
		return Configuration{Mode: ModeLocal}, nil
	}
	if err := validateLocalConfigurationFile(localFile); err != nil {
		return Configuration{}, err
	}
	return Configuration{Mode: ModeLocal, ConfigurationFile: localFile}, nil
}

func loadProject() (Configuration, error) {
	configuration, err := validateProjectReference()
	if err != nil {
		return Configuration{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	loaded, err := projectconfig.LoadFromEnvironment(ctx)
	if err != nil {
		return Configuration{}, fmt.Errorf("load accepted project configuration: %w", err)
	}
	environment, err := loaded.ResolveApplicationEnvironment(ctx)
	if err != nil {
		return Configuration{}, fmt.Errorf("resolve accepted application environment: %w", err)
	}
	if err = environment.Apply(); err != nil {
		return Configuration{}, err
	}
	return configuration, nil
}

func validateProjectReference() (Configuration, error) {
	configuration := Configuration{
		Mode: ModeProject, ProjectID: os.Getenv("DEX_PROJECT_ID"), Scope: os.Getenv("DEX_PROJECT_SCOPE"), SessionID: os.Getenv("DEX_PROJECT_SESSION_ID"),
		ObjectKey: os.Getenv("DEX_PROJECT_CONFIG_KEY"), ObjectVersion: os.Getenv("DEX_PROJECT_CONFIG_VERSION"), ObjectDigest: os.Getenv("DEX_PROJECT_CONFIG_DIGEST"),
		StorageBucket: os.Getenv("DEX_PROJECT_STORAGE_BUCKET"), StoragePrefix: os.Getenv("DEX_PROJECT_STORAGE_PREFIX"), KMSKeyARN: os.Getenv("DEX_PROJECT_STORAGE_KMS_KEY_ARN"), LocalStorageEndpoint: os.Getenv("DEX_PROJECT_STORAGE_ENDPOINT"),
	}
	if !identityPattern.MatchString(configuration.ProjectID) || (configuration.Scope != "live" && configuration.Scope != "preview") || (configuration.Scope == "live" && configuration.SessionID != "") || (configuration.Scope == "preview" && !identityPattern.MatchString(configuration.SessionID)) {
		return Configuration{}, fmt.Errorf("DEX_PROJECT_* scope is invalid")
	}
	expectedKey := "projects/" + configuration.ProjectID + "/" + configuration.Scope
	if configuration.Scope == "preview" {
		expectedKey += "/" + configuration.SessionID
	}
	expectedKey += "/configuration/head"
	if configuration.ObjectKey != expectedKey {
		return Configuration{}, fmt.Errorf("DEX_PROJECT_CONFIG_KEY differs from the configured project scope")
	}
	if configuration.ObjectVersion == "" || configuration.ObjectVersion == "null" || strings.ContainsAny(configuration.ObjectVersion, "\x00\r\n") {
		return Configuration{}, fmt.Errorf("DEX_PROJECT_CONFIG_VERSION must identify an exact immutable version")
	}
	if !digestPattern.MatchString(configuration.ObjectDigest) {
		return Configuration{}, fmt.Errorf("DEX_PROJECT_CONFIG_DIGEST must be sha256: followed by lowercase hexadecimal")
	}
	if configuration.StorageBucket == "" || strings.ContainsAny(configuration.StorageBucket, "/\\\x00\r\n") {
		return Configuration{}, fmt.Errorf("DEX_PROJECT_STORAGE_BUCKET is invalid")
	}
	if configuration.StoragePrefix != "" && !isSafePrefix(configuration.StoragePrefix) {
		return Configuration{}, fmt.Errorf("DEX_PROJECT_STORAGE_PREFIX is invalid")
	}
	allowLocal := os.Getenv("DEX_PROJECT_ALLOW_LOCAL_STORAGE")
	switch allowLocal {
	case "", "false":
		if configuration.LocalStorageEndpoint != "" || !kmsKeyPattern.MatchString(configuration.KMSKeyARN) {
			return Configuration{}, fmt.Errorf("hosted project storage requires an exact KMS key ARN and the default AWS endpoint")
		}
	case "true":
		if !isLocalStorageEndpoint(configuration.LocalStorageEndpoint) || (configuration.KMSKeyARN != "" && !kmsKeyPattern.MatchString(configuration.KMSKeyARN)) {
			return Configuration{}, fmt.Errorf("local project storage requires an explicit isolated endpoint")
		}
	default:
		return Configuration{}, fmt.Errorf("DEX_PROJECT_ALLOW_LOCAL_STORAGE must be true or false")
	}
	publicURL, err := validatedPublicURL(os.Getenv("PUBLIC_BASE_URL"))
	if err != nil {
		return Configuration{}, err
	}
	configuration.PublicBaseURL = publicURL
	return configuration, nil
}

func hasProjectEnvironment() bool {
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, "DEX_PROJECT_") {
			_, value, _ := strings.Cut(entry, "=")
			if value != "" {
				return true
			}
		}
	}
	return false
}
func isSafePrefix(prefix string) bool {
	if len(prefix) > 1024 || strings.HasPrefix(prefix, "/") || strings.ContainsAny(prefix, "\\\x00\r\n") {
		return false
	}
	for _, segment := range strings.Split(strings.TrimSuffix(prefix, "/"), "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}
func isLocalStorageEndpoint(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return false
	}
	hostname := parsed.Hostname()
	address := net.ParseIP(hostname)
	return hostname == "localhost" || hostname == "host.docker.internal" || strings.HasSuffix(hostname, ".svc.cluster.local") || (address != nil && (address.IsLoopback() || address.IsPrivate()))
}
func validatedPublicURL(value string) (string, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return "", fmt.Errorf("PUBLIC_BASE_URL must be an absolute HTTP(S) URL without credentials, query, or fragment")
	}
	return strings.TrimRight(value, "/"), nil
}
func validateLocalConfigurationFile(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("DEX_CONNECTOR_CONFIG_FILE must be an absolute path")
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("local connector configuration is unavailable")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > maximumConfigurationBytes {
		return fmt.Errorf("local connector configuration is not a bounded regular file")
	}
	size, err := io.Copy(io.Discard, io.LimitReader(file, maximumConfigurationBytes+1))
	if err != nil {
		return err
	}
	if size != info.Size() {
		return fmt.Errorf("local connector configuration changed while reading")
	}
	return nil
}
