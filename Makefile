VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X bwenv/cmd.Version=$(VERSION) -X bwenv/cmd.Commit=$(COMMIT) -X bwenv/cmd.Date=$(BUILD_DATE)
PREFIX ?= /usr/local
DESTDIR ?=

SHELL_SCRIPTS := install.sh $(wildcard scripts/*.sh)

.PHONY: build build-all clean install test test-race fmt fmt-check vet lint shellcheck compat verify run

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o bwenv .

build-all: clean
	mkdir -p dist
	GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/bwenv-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/bwenv-darwin-arm64 .
	GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/bwenv-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/bwenv-linux-arm64 .
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/bwenv-windows-amd64.exe .

clean:
	rm -rf dist bwenv bwenv.exe

install: build
	install -d "$(DESTDIR)$(PREFIX)/bin"
	install -m 0755 bwenv "$(DESTDIR)$(PREFIX)/bin/bwenv"

test:
	go test ./...

test-race:
	go test -race ./...

fmt:
	gofmt -w .

fmt-check:
	test -z "$$(gofmt -l .)"

vet:
	go vet ./...

lint:
	golangci-lint run

shellcheck:
	shellcheck $(SHELL_SCRIPTS)

compat:
	./scripts/check-bws-compat.sh

verify: fmt-check vet test
	bash -n $(SHELL_SCRIPTS)
	./scripts/check-doc-links.sh
	./scripts/test-installer.sh
	./scripts/test-bws-compat.sh
	./scripts/test-homebrew-formula.sh

run: build
	./bwenv $(ARGS)
