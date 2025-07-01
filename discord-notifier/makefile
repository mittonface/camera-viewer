.PHONY: build run clean deps

# Build the binary
build:
	CGO_ENABLED=1 go build -o discord-notifier main.go

# Run the notifier
run: build
	./discord-notifier

# Download dependencies
deps:
	go mod download
	go mod tidy

# Clean build artifacts
clean:
	rm -f discord-notifier
	rm -f discord-notifier.db

# Build for Linux (useful for deploying to servers)
build-linux:
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o discord-notifier-linux main.go

# Test run (dry run without actually sending notifications)
test:
	go run main.go