.PHONY: tidy test lint vet cover build-local build-linux-amd64 build-linux-arm64 clean setup

tidy:
	go mod tidy
	go fmt ./...

test:
	go test -v -race ./...

lint:
	@echo "=> Running golangci-lint..."
	golangci-lint run ./...

vet:
	go vet ./...

cover:
	go test -cover ./...

build-local:
	mkdir -p bin
	go build -o bin/example ./cmd/example

build-linux-amd64:
	mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -o bin/example-linux-amd64 ./cmd/example

build-linux-arm64:
	mkdir -p bin
	GOOS=linux GOARCH=arm64 go build -o bin/example-linux-arm64 ./cmd/example

clean:
	rm -rf bin/

setup:
	@echo "=> Configuring local git hooks..."
	git config core.hooksPath .githooks
	chmod +x .githooks/*
	@echo "✅ Pre-commit hooks installed."
