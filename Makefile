.PHONY: build test lint clean cover install-tools install-hooks

BINARY := qlik-script-extractor
GOLANGCI_LINT_VERSION := v2.11.3

build:
	go build -o $(BINARY) .

test:
	go test ./... -v -count=1

cover:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -func=coverage.out

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY) coverage.out

install-tools: ## Note: golangci-lint is pre-installed in the devcontainer; this target is for fresh environments or version updates
	go install github.com/caarlos0/svu@latest
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin $(GOLANGCI_LINT_VERSION)

install-hooks:
	cp scripts/pre-commit .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
