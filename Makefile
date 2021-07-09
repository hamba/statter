include github.com/hamba/make/golang

generate:
	@echo "==> Generating"
	@go get -modfile go.tools.mod github.com/a8m/syncmap
	@go generate
	@echo "==> Done"
.PHONY: generate

bench:
	@go test -bench . ./...
.PHONY: bench
