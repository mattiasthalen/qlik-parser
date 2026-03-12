.PHONY: build test lint clean cover install-tools install-hooks next-version release release-patch release-minor release-major

BINARY := qlik-script-extractor
GOLANGCI_LINT_VERSION := v2.11.3
SVU_VERSION := v3.4.0

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
	go install github.com/caarlos0/svu@$(SVU_VERSION)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin $(GOLANGCI_LINT_VERSION)

install-hooks:
	cp scripts/pre-commit $(shell git rev-parse --git-common-dir)/hooks/pre-commit
	chmod +x $(shell git rev-parse --git-common-dir)/hooks/pre-commit

next-version:
	svu next

release:
	git tag $(shell svu next)
	git push --tags

release-patch:
	git tag $(shell svu patch)
	git push --tags

release-minor:
	git tag $(shell svu minor)
	git push --tags

release-major:
	git tag $(shell svu major)
	git push --tags
