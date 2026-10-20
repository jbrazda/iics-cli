BINARY_NAME := iics
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build clean test vet lint fmt install completions demo

build:
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/iics

install:
	go install $(LDFLAGS) ./cmd/iics

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
	go run ./cmd/iics completion bash        > completions/iics.bash
	go run ./cmd/iics completion zsh         > completions/iics.zsh
	go run ./cmd/iics completion fish        > completions/iics.fish
	go run ./cmd/iics completion powershell  > completions/iics.ps1

# Re-record the README demo with the profile and agent named in
# scripts/demo/scenes.json. Needs scripts/demo/masks.local.json and agg
# (https://github.com/asciinema/agg). See scripts/demo/README.md.
DEMO_FONT := JetBrainsMono Nerd Font Mono,FiraCode Nerd Font Mono,DejaVu Sans Mono

demo: build
	PATH="$(CURDIR):$$PATH" python3 -I scripts/demo/record.py
	agg --font-family "$(DEMO_FONT)" --font-size 14 --theme dracula \
		--idle-time-limit 3 --last-frame-duration 4 \
		docs/demo/iics-demo.cast docs/demo/iics-demo.gif

all: fmt vet test clean build completions
