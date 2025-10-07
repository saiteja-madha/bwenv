.PHONY: build build-all clean install test

# Build for current platform
build:
	go build -o bwenv

# Build for all platforms
build-all: clean
	mkdir -p dist
	GOOS=darwin GOARCH=amd64 go build -o dist/bwenv-darwin-amd64
	GOOS=darwin GOARCH=arm64 go build -o dist/bwenv-darwin-arm64
	GOOS=linux GOARCH=amd64 go build -o dist/bwenv-linux-amd64
	GOOS=linux GOARCH=arm64 go build -o dist/bwenv-linux-arm64
	GOOS=windows GOARCH=amd64 go build -o dist/bwenv-windows-amd64.exe

# Clean build artifacts
clean:
	rm -rf dist/ bwenv bwenv.exe

# Install to /usr/local/bin
install: build
	cp bwenv /usr/local/bin/

# Run tests
test:
	go test ./...

# Run with args (example: make run ARGS="list myapp")
run: build
	./bwenv $(ARGS)