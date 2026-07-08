.PHONY: test build run clean

# Default target
all: test build

# Run all tests
test:
	go test ./... -v -count=1

# Build binary
build:
	go build -o convallaria ./cmd/convallaria/

# Build for Windows
build-windows:
	GOOS=windows GOARCH=amd64 go build -o convallaria.exe ./cmd/convallaria/

# Build for Linux
build-linux:
	GOOS=linux GOARCH=amd64 go build -o convallaria ./cmd/convallaria/

# Build for macOS
build-darwin:
	GOOS=darwin GOARCH=amd64 go build -o convallaria ./cmd/convallaria/

# Build all platforms
build-all: build-windows build-linux build-darwin

# Run server (mock mode, no API key needed)
run:
	go run ./cmd/convallaria/ -port 8080

# Clean build artifacts
clean:
	rm -f convallaria convallaria.exe *.db *.db-shm *.db-wal