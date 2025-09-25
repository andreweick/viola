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