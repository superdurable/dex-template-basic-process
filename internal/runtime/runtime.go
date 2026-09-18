package runtime

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/superdurable/dex-template-basic-process/internal/process"
	"github.com/superdurable/dex/blob-cache-go/blobcache"
	"github.com/superdurable/dex/sdk-go/dex"
)

type Runtime struct {
	Processes *process.Service
	worker    *dex.Worker
	client    *dex.Client
	cache     *blobcache.Cache
}

func New(logger *slog.Logger) (*Runtime, error) {
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
	return &Runtime{Processes: process.NewService(client), worker: worker, client: client, cache: cache}, nil
}

func (runtime *Runtime) StartWorker() <-chan error {
	result := make(chan error, 1)
	go func() { result <- runtime.worker.Start() }()
	return result
}

func (runtime *Runtime) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return errors.Join(runtime.worker.Stop(ctx), runtime.client.Close(), runtime.cache.Close())
}

func environment(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
