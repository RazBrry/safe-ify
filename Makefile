BINARY_NAME=safe-ify
BUILD_DIR=./bin
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

.PHONY: build clean test lint

build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/safe-ify/

clean:
	rm -rf $(BUILD_DIR)

test:
	go test ./... -v -race

test-coverage:
	go test ./... -coverprofile=coverage.out -race
	go tool cover -func=coverage.out

lint:
	golangci-lint run ./...
