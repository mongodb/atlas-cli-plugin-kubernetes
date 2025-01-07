
CLI_SOURCE_FILES?=./cmd/plugin
CLI_BINARY_NAME=binary
CLI_DESTINATION=./bin/$(CLI_BINARY_NAME)

TEST_CMD?=go test
UNIT_TAGS?=unit
COVERAGE?=coverage.out

.PHONY: build
build: ## Generate the binary in ./bin
	@echo "==> Building kubernetes plugin binary"
	go build -o $(CLI_DESTINATION) $(CLI_SOURCE_FILES)

.PHONY: devtools
devtools:  ## Install dev tools
	@echo "==> Installing dev tools..."
	go install github.com/google/addlicense@latest
	go install github.com/golang/mock/mockgen@latest
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin $(GOLANGCI_VERSION)


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