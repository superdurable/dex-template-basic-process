// Copyright (c) 2026 Super Durable
// SPDX-License-Identifier: MIT

package processhost

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/superdurable/dex-template-basic-process/internal/connectorconfiguration"
	"github.com/superdurable/dex-template-basic-process/internal/process"
	"github.com/superdurable/dex/blob-cache-go/blobcache"
	"github.com/superdurable/dex/sdk-go/dex"
)

type Host struct {
	Processes *process.Service
	worker    *dex.Worker
	client    *dex.Client
	cache     *blobcache.Cache
}

func New(logger *slog.Logger) (*Host, error) {
	if _, err := connectorconfiguration.Load(false); err != nil {
		return nil, fmt.Errorf("validate connector configuration: %w", err)
	}
	registry, err := dex.NewRegistry([]dex.Flow{process.BasicProcess})
	if err != nil {
		return nil, fmt.Errorf("register Basic Process Flow: %w", err)
	}
	cache, err := blobcache.New(&blobcache.Config{Dir: environment("DEX_BLOB_CACHE_DIR", filepath.Join(os.TempDir(), "basic-process-blobs")), MaxBytes: 256 << 20, Logger: logger})
	if err != nil {
		return nil, fmt.Errorf("create Dex blob cache: %w", err)
	}
	flowServiceAddress := environment("DEX_FLOW_SERVICE_ADDRESS", "127.0.0.1:8801")
	worker, err := dex.NewWorker(registry, cache, dex.WorkerOptions{BindAddress: environment("DEX_WORKER_BIND_ADDRESS", "127.0.0.1:8811"), WorkerTarget: dex.WorkerTarget{Address: environment("DEX_WORKER_TARGET", "127.0.0.1:8811")}, FlowServiceAddress: flowServiceAddress, Logger: logger})
	if err != nil {
		_ = cache.Close()
		return nil, fmt.Errorf("create Dex Worker: %w", err)
	}
	client, err := dex.NewClient(registry, cache, dex.ClientOptions{FlowServiceAddress: flowServiceAddress, WorkerTarget: worker.WorkerTarget(), Logger: logger})
	if err != nil {
		_ = worker.Stop(context.Background())
		_ = cache.Close()
		return nil, fmt.Errorf("create Dex Client: %w", err)
	}
	return &Host{Processes: process.NewService(client), worker: worker, client: client, cache: cache}, nil
}

func (host *Host) StartWorker() <-chan error {
	result := make(chan error, 1)
	go func() { result <- host.worker.Start() }()
	return result
}

func (host *Host) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return errors.Join(host.worker.Stop(ctx), host.client.Close(), host.cache.Close())
}

func environment(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
