BINARY_NAME := iics
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build clean test vet lint fmt install completions

build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

install:
	go install $(LDFLAGS) .

clean:
	rm -f $(BINARY_NAME)

test:
	go test -v ./...

vet:
	go vet ./...

fmt:
	gofmt -s -w .

lint: vet
	@which golangci-lint > /dev/null 2>&1 || echo "golangci-lint not installed"
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run ./... || true

completions:
	go run . completion bash        > completions/iics.bash
	go run . completion zsh         > completions/iics.zsh
	go run . completion fish        > completions/iics.fish
	go run . completion powershell  > completions/iics.ps1

all: fmt vet test build completions
