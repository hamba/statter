# Format all files
fmt:
	@echo "==> Formatting source"
	@golangci-lint fmt ./...
	@echo "==> Done"
.PHONY: fmt

# Tidy the go.mod file
tidy:
	@echo "==> Cleaning go.mod"
	@go mod tidy
	@echo "==> Done"
.PHONY: tidy

# Run all tests
test:
	@go test -cover -race ./...
.PHONY: test

# Lint the project
lint:
	@golangci-lint run ./...
.PHONY: lint

# Generate Go files
generate:
	@echo "==> Generating"
	@go install -modfile go.tools.mod github.com/a8m/syncmap
	@go generate
	@echo "==> Done"
.PHONY: generate

# Run benchmarks
bench:
	@go test -bench . ./...
.PHONY: bench
