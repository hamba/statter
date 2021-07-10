include github.com/hamba/make/golang

# Generate Go files
generate:
	@echo "==> Generating"
	@go get -modfile go.tools.mod github.com/a8m/syncmap
	@go generate
	@echo "==> Done"
.PHONY: generate

# Run benchmarks
bench:
	@go test -bench . ./...
.PHONY: bench
