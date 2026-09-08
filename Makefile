# Config
BINARY_NAME=vaultify

.PHONY: help
help:
	@echo "Usage:"
	@echo "  make <goal>"
	@echo ""
	@echo "Goals:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: all test clean

test: ## Launch all tests
	go clean -testcache
	go test -v ./...
# go tool cover -html=coverage.out -o coverage.html

benchmark:
	go clean -testcache
	go test -v ./test -bench=. -benchmem

clean: ## Delete binary build of service
	go clean
	rm -f $(BINARY_NAME)

lint: ## Lauch linter
	@if ! command -v ./bin/golangci-lint >/dev/null 2>&1; then \
	echo "golangci-lint is not installed... wait a sec"; \
	$(MAKE) install-tools; \
	fi

	./bin/golangci-lint run --config=.golangci.yaml \
	--max-issues-per-linter=1000 \
	--max-same-issues=1000 \
	./...

pprof: ## Launch pprof
	go tool pprof http://localhost:8080/debug/pprof/profile

imports: ## Format imports
	@if ! command -v goimports-reviser >/dev/null 2>&1; then \
		echo "goimports-reviser не установлен. Устанавливаем..."; \
		$(MAKE) install-tools; \
	fi
	./bin/goimports-reviser -use-cache -recursive .

install-tools: ## Install revising & other deps
	$(info Installing binary dependencies...)
	mkdir -p ./bin
	GOBIN=$(PWD)/bin go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
 # GOBIN=$(PWD)/bin go install github.com/incu6us/goimports-reviser/v3@latest
	GOBIN=$(PWD)/bin go install github.com/vektra/mockery/v2@latest
	GOBIN=$(PWD)/bin go install github.com/go-swagger/go-swagger/cmd/swagger@v0.33.1
	GOBIN=$(PWD)/bin go install github.com/mailru/easyjson/easyjson@latest

goimports:
	./bin/goimports-reviser --company-prefixes github.com  -use-cache -recursive .;

generate-mocks: ## Generate mocks only
	./bin/mockery --all --recursive --dir=./internal --output=./mocks --case=underscore
	@echo "Finished generating mocks"
