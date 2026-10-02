// Copyright (c) 2026 Super Durable
// SPDX-License-Identifier: MIT

// Package connectorconfiguration selects the official Connector SDK configuration
// store before the application creates clients, Workers or goroutines.
package connectorconfiguration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/superdurable/dex-connectors-library/sdkgo/localconfig"
	"github.com/superdurable/dex-connectors-library/sdkgo/projectconfig"
)

// Configuration contains exactly one configuration authority. Neither store
// eagerly loads provider credentials. Connections resolve them at actual use.
type Configuration struct {
	Local   *localconfig.Store
	Project *projectconfig.LoadedProject
}

// Load verifies the immutable project snapshot and applies its pinned application
// environment before startup. Standalone development may use the SDK local store.
// The caller owns the deadline; errors prevent Worker startup.
func Load(ctx context.Context, requiresConnections bool) (Configuration, error) {
	projectEnvironment := false
	for _, entry := range os.Environ() {
		name, value, _ := strings.Cut(entry, "=")
		if value == "" {
			continue
		}
		if strings.HasPrefix(name, "SUPERVERSE_CONNECTOR_") {
			return Configuration{}, errors.New("obsolete connector configuration: use the official DEX_PROJECT_* deployment contract")
		}
		projectEnvironment = projectEnvironment || strings.HasPrefix(name, "DEX_PROJECT_")
	}
	localFile := strings.TrimSpace(os.Getenv("DEX_CONNECTOR_CONFIG_FILE"))
	if projectEnvironment {
		if localFile != "" {
			return Configuration{}, errors.New("local and project connector configuration cannot be combined")
		}
		project, err := projectconfig.LoadFromEnvironment(ctx)
		if err != nil {
			return Configuration{}, fmt.Errorf("load accepted project configuration: %w", err)
		}
		environment, err := project.ResolveApplicationEnvironment(ctx)
		if err != nil {
			return Configuration{}, fmt.Errorf("resolve application environment: %w", err)
		}
		if err := environment.Apply(); err != nil {
			return Configuration{}, fmt.Errorf("apply application environment: %w", err)
		}
		return Configuration{Project: project}, nil
	}
	if localFile == "" && !requiresConnections {
		return Configuration{}, nil
	}
	local, err := localconfig.LoadFromEnvironment()
	if err != nil {
		return Configuration{}, fmt.Errorf("load local connector configuration: %w", err)
	}
	return Configuration{Local: local}, nil
}
