# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html

GOCOVERDIR?=$(abspath cov)
GOLANGCI_VERSION?=latest

PLUGIN_SOURCE_FILES?=./cmd/plugin
ifeq ($(OS),Windows_NT)
	PLUGIN_BINARY_NAME=atlas-cli-plugin-kubernetes.exe
	E2E_ATLASCLI_BINARY_PATH=../bin/atlas.exe
else
    ATLAS_VERSION?=$(shell git describe --match "atlascli/v*" | cut -d "v" -f 2)
	PLUGIN_BINARY_NAME=atlas-cli-plugin-kubernetes
	E2E_ATLASCLI_BINARY_PATH=../bin/atlas
endif
PLUGIN_BINARY_PATH=./bin/$(PLUGIN_BINARY_NAME)
PURLS_DEP_PATH=build/package
MANIFEST_FILE?=manifest.yml
WIN_MANIFEST_FILE?=manifest.windows.yml

TEST_CMD?=go test
UNIT_TAGS?=unit
COVERAGE?=coverage.out

E2E_PLUGIN_BINARY_PATH=../../$(PLUGIN_BINARY_PATH)
E2E_TAGS?=e2e
E2E_TIMEOUT?=60m
E2E_PARALLEL?=1
E2E_EXTRA_ARGS?=

export E2E_PLUGIN_BINARY_PATH
export E2E_ATLASCLI_BINARY_PATH

.PHONY: setup
setup: deps devtools ## Set up dev env

.PHONY: deps
deps:  ## Download go module dependencies
	@echo "==> Installing go.mod dependencies..."
	go mod download
	go mod tidy

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
	go build -gcflags="all=-N -l" -o ./bin/atlas-cli-plugin-kubernetes ./cmd/plugin

.PHONY: e2e-test
e2e-test: build-debug ## Run E2E tests
# the target assumes the MCLI_* environment variables are exported
	@./scripts/atlas-binary.sh
	@echo "==> Running E2E tests..."
	GOCOVERDIR=$(GOCOVERDIR) $(TEST_CMD) -race -v -p 1 -parallel $(E2E_PARALLEL) -v -timeout $(E2E_TIMEOUT) -tags="$(E2E_TAGS)" ./test/e2e... $(E2E_EXTRA_ARGS)

.PHONY: gen-mocks
gen-mocks: ## Generate mocks
	@echo "==> Generating mocks"
	rm -rf ./internal/mocks
	GOFLAGS=-mod=mod go generate ./internal...

.PHONY: gen-docs
gen-docs: ## Generate docs for atlascli commands
	@echo "==> Generating docs"
	go run -ldflags "$(LINKER_FLAGS)" ./tools/docs/main.go

.PHONY: check-licenses
check-licenses: ## Check licenses
	@echo "==> Running lincense checker..."
	@build/ci/check-licenses.sh

.PHONY: generate-all-manifests
generate-all-manifests: generate-manifest generate-manifest-windows

.PHONY: generate-manifest
generate-manifest: ## Generate the manifest file for non-windows OSes
	@echo "==> Generating non-windows manifest file"
	printenv
	BINARY=$(CLI_BINARY_NAME) envsubst < manifest.template.yml > $(MANIFEST_FILE)

.PHONY: generate-manifest-windows
generate-manifest-windows: ## Generate the manifest file for windows OSes
	@echo "==> Generating windows manifest file"
	printenv
	CLI_BINARY_NAME="${CLI_BINARY_NAME}.exe" MANIFEST_FILE="$(WIN_MANIFEST_FILE)" $(MAKE) generate-manifest

.PHONY: generate-purls
generate-purls: ## Calls the script to generate dependency list for all platforms
	@./scripts/generate-purls.sh

.PHONY: check-purls
check-purls: ## Checks that the dependency list purls.txt matches the current code
	@echo "==> Checking dependency list purls.txt is up to date"
	${MAKE} generate-purls PURLS_FILE=purls_check.txt
	@if ! diff ${PURLS_DEP_PATH}/purls.txt ${PURLS_DEP_PATH}/purls_check.txt > /dev/null; then \
		echo "Dependency list is out of date! Please rerun 'make generate-purls' to update."; \
		rm -f ${PURLS_DEP_PATH}/purls_check.txt; \
		exit 1; \
	fi
	@rm -f ${PURLS_DEP_PATH}/purls_check.txt
	@echo "Dependency list up to date!"

.PHONY: help
.DEFAULT_GOAL := help
help:
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
