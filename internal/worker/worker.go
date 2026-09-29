// Copyright (c) 2026 Super Durable
// SPDX-License-Identifier: MIT

package worker

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

type Worker struct {
	Processes *process.Service
	dexWorker *dex.Worker
	client    *dex.Client
	cache     *blobcache.Cache
}

func New(logger *slog.Logger) (*Worker, error) {
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
	dexWorker, err := dex.NewWorker(registry, cache, dex.WorkerOptions{BindAddress: environment("DEX_WORKER_BIND_ADDRESS", "127.0.0.1:8811"), WorkerTarget: dex.WorkerTarget{Address: environment("DEX_WORKER_TARGET", "127.0.0.1:8811")}, FlowServiceAddress: flowServiceAddress, Logger: logger})
	if err != nil {
		_ = cache.Close()
		return nil, fmt.Errorf("create Dex Worker: %w", err)
	}
	client, err := dex.NewClient(registry, cache, dex.ClientOptions{FlowServiceAddress: flowServiceAddress, WorkerTarget: dexWorker.WorkerTarget(), Logger: logger})
	if err != nil {
		_ = dexWorker.Stop(context.Background())
		_ = cache.Close()
		return nil, fmt.Errorf("create Dex Client: %w", err)
	}
	return &Worker{Processes: process.NewService(client), dexWorker: dexWorker, client: client, cache: cache}, nil
}

func (worker *Worker) Start() <-chan error {
	result := make(chan error, 1)
	go func() { result <- worker.dexWorker.Start() }()
	return result
}

func (worker *Worker) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return errors.Join(worker.dexWorker.Stop(ctx), worker.client.Close(), worker.cache.Close())
}

func environment(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
