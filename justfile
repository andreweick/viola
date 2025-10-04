# When a recipe isn't found here, search up the directory tree for it
# Stops at the first justfile without 'set fallback' (usually project root)
set fallback

# Build the viola CLI
build:
    go build -o bin/viola ./cmd/viola

# Run viola with arguments
run *ARGS: build
    ./bin/viola {{ ARGS }}

# Run tests
test:
    go test -v ./...

# Download dependencies
deps:
    go mod download
    go mod tidy

# Clean build artifacts
clean:
    rm -rf bin/

# Install locally
install: build
    cp bin/viola ~/.local/bin/viola

# Show help
help:
    @just --list
