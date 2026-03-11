.PHONY: build test lint clean

BINARY := qlik-script-extractor

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
