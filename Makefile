SHELL := /bin/bash

# `make` should build the binary by default.
.DEFAULT_GOAL := build

.PHONY: build tu tu-help help tools fmt fmt-check test vet verify-release ci clean install all

BIN_DIR := $(CURDIR)/bin
BIN := $(BIN_DIR)/tu
CMD := ./cmd/tu

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

TOOLS_DIR := $(CURDIR)/.tools
GOFUMPT := $(TOOLS_DIR)/gofumpt
GOIMPORTS := $(TOOLS_DIR)/goimports

build:
	@mkdir -p $(BIN_DIR)
	@go build -ldflags "$(LDFLAGS)" -o $(BIN) $(CMD)

tu: build
	@if [ -z "$(ARGS)" ]; then \
		$(BIN) --help; \
	else \
		$(BIN) $(ARGS); \
	fi

tu-help: build
	@$(BIN) --help

help: tu-help

tools:
	@mkdir -p $(TOOLS_DIR)
	@GOBIN=$(TOOLS_DIR) go install mvdan.cc/gofumpt@v0.7.0
	@GOBIN=$(TOOLS_DIR) go install golang.org/x/tools/cmd/goimports@v0.28.0

fmt: tools
	@$(GOIMPORTS) -local github.com/richclement/tu -w .
	@$(GOFUMPT) -w .

fmt-check: tools
	@before="$$(mktemp)"; after="$$(mktemp)"; \
	git diff --binary -- '*.go' go.mod go.sum > "$$before"; \
	$(GOIMPORTS) -local github.com/richclement/tu -w .; \
	$(GOFUMPT) -w .; \
	git diff --binary -- '*.go' go.mod go.sum > "$$after"; \
	if ! cmp -s "$$before" "$$after"; then \
		git diff -- '*.go' go.mod go.sum; \
		rm -f "$$before" "$$after"; \
		exit 1; \
	fi; \
	rm -f "$$before" "$$after"

test:
	@go test ./...

vet:
	@go vet ./...

verify-release:
	@mkdir -p dist
	@go build -ldflags "$(LDFLAGS)" -o ./dist/tu $(CMD)
	@go run ./tools/releaseverify --binary ./dist/tu --version $(VERSION)

ci: fmt-check test vet

clean:
	rm -rf bin/
	rm -rf dist/
	rm -rf .tools/

install:
	go install -ldflags "$(LDFLAGS)" $(CMD)

all: fmt test vet build
