# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html

GOCOVERDIR?=$(abspath cov)

PLUGIN_SOURCE_FILES?=./cmd/plugin
ifeq ($(OS),Windows_NT)
	PLUGIN_BINARY_NAME=binary.exe
	E2E_ATLASCLI_BINARY_PATH=../bin/atlas.exe
else
    ATLAS_VERSION?=$(shell git describe --match "atlascli/v*" | cut -d "v" -f 2)
	PLUGIN_BINARY_NAME=binary
	E2E_ATLASCLI_BINARY_PATH=../bin/atlas
endif
PLUGIN_BINARY_PATH=./bin/$(PLUGIN_BINARY_NAME)

TEST_CMD?=go test
UNIT_TAGS?=unit
COVERAGE?=coverage.out
GOCOVERDIR?=$(abspath cov)
TEST_CMD?=go test
UNIT_TAGS?=unit
E2E_TAGS?=e2e
E2E_TIMEOUT?=60m
E2E_PARALLEL?=1
E2E_EXTRA_ARGS?=

.PHONY: deps
deps:  ## Download go module dependencies
	@echo "==> Installing go.mod dependencies..."
	go mod download
	go mod tidy

E2E_PLUGIN_BINARY_PATH=../../$(PLUGIN_BINARY_PATH)
E2E_TAGS?=e2e
E2E_TIMEOUT?=60m
E2E_PARALLEL?=1
E2E_EXTRA_ARGS?=

export E2E_PLUGIN_BINARY_PATH
export E2E_ATLASCLI_BINARY_PATH

.PHONY: build
build: ## Generate the binary in ./bin
	@echo "==> Building kubernetes plugin binary"
	go build -o $(PLUGIN_BINARY_PATH) $(PLUGIN_SOURCE_FILES)

.PHONY: devtools
devtools:  ## Install dev tools
	@echo "==> Installing dev tools..."
	go install github.com/google/addlicense@latest
	go install github.com/golang/mock/mockgen@latest
	go install mvdan.cc/sh/v3/cmd/shfmt@latest
	go install golang.org/x/tools/cmd/goimports@latest
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin $(GOLANGCI_VERSION)

.PHONY: fmt
fmt: ## Format changed go
	@scripts/fmt.sh

.PHONY: lint
lint: ## Run linter
	golangci-lint run

.PHONY: unit-test
unit-test: ## Run unit-tests
	@echo "==> Running unit tests..."
	$(TEST_CMD) --tags="$(UNIT_TAGS)" -race -cover -coverprofile $(COVERAGE) -count=1 ./...

.PHONY: fuzz-normalizer-test
fuzz-normalizer-test: ## Run fuzz test
	@echo "==> Running fuzz test..."
	$(TEST_CMD) -fuzz=Fuzz -fuzztime 50s --tags="$(UNIT_TAGS)" -race ./internal/kubernetes/operator/resources

.PHONY: build-debug
build-debug: ## Generate a binary in ./bin for debugging plugin
	@echo "==> Building kubernetes plugin binary for debugging"
	go build -gcflags="all=-N -l" -o ./bin/binary ./cmd/plugin

.PHONY: e2e-test
e2e-test: build-debug ## Run E2E tests
# the target assumes the MCLI_* environment variables are exported
	@./scripts/atlas-binary.sh
	@echo "==> Running E2E tests..."
	GOCOVERDIR=$(GOCOVERDIR) go test -v -p 1 -parallel 1 -v -timeout 60m -tags="e2e" ./test/e2e... $(E2E_EXTRA_ARGS)

.PHONY: gen-mocks
gen-mocks: ## Generate mocks
	@echo "==> Generating mocks"
	rm -rf ./internal/mocks
	go generate ./internal...

.PHONY: gen-docs
gen-docs: ## Generate docs for atlascli commands
	@echo "==> Generating docs"
	go run -ldflags "$(LINKER_FLAGS)" ./tools/docs/main.go

.PHONY: check-licenses
check-licenses: ## Check licenses
	@echo "==> Running lincense checker..."
	@build/ci/check-licenses.sh

.PHONY: help
.DEFAULT_GOAL := help
help:
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'