IMAGE := "kuberhealthy/network-connection-check"
TAG := "latest"

# Build the network connection check container locally.
build:
	podman build -f Containerfile -t {{IMAGE}}:{{TAG}} .

# Run the unit tests for the network connection check.
test:
	go test ./...

# Build the network connection check binary locally.
binary:
	go build -o bin/network-connection-check ./cmd/network-connection-check
