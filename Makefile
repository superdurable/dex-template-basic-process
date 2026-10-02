SHELL := /usr/bin/env bash

.PHONY: bootstrap generate check-fdg-v2 check-contracts check-template-repository superverse-release-artifacts build dev check-static check
bootstrap:
	go mod download
	go -C tools/openapi mod download
	npm --prefix web ci
generate:
	./scripts/generate-openapi.sh
check-fdg-v2:
	./scripts/check-fdg-v2.sh
superverse-release-artifacts:
	@test -n "$(SUPERVERSE_RELEASE_ARTIFACT_DIR)" || { echo "SUPERVERSE_RELEASE_ARTIFACT_DIR is required" >&2; exit 1; }
	python3 ./scripts/generate-release-artifacts.py --output-directory "$(SUPERVERSE_RELEASE_ARTIFACT_DIR)"
check-contracts:
	go run ./tools/checkcontract
check-template-repository:
	go run ./tools/checkcontract --repository-automation
build: generate
	npm --prefix web run build
	go build -o bin/basic-process ./cmd/server
dev: generate
	./scripts/with-dex.sh bash -c 'npm --prefix web run build && go run ./cmd/server'
check-static: generate check-fdg-v2 check-contracts
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './upstream-dex/*'))" || { gofmt -d $$(gofmt -l $$(find . -name '*.go' -not -path './upstream-dex/*')); exit 1; }
	go mod tidy -diff
	go vet ./...
	npm --prefix web run build
	go build -o bin/basic-process ./cmd/server
check: bootstrap
	$(MAKE) check-static
