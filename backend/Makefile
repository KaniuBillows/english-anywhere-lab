.PHONY: build run test migrate clean

BINARY_API=bin/api
BINARY_WORKER=bin/worker

build:
	go build -o $(BINARY_API) ./cmd/api
	go build -o $(BINARY_WORKER) ./cmd/worker

run: build
	./$(BINARY_API)

test:
	go test ./... -v -count=1

test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf bin/ coverage.out coverage.html

tidy:
	go mod tidy
