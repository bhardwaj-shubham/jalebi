.PHONY: run build test fmt clean

VERSION ?= dev
LDFLAGS := -X github.com/bhardwaj-shubham/jalebi/internal/cli.version=$(VERSION)

run:
	go run ./cmd/jalebi

build:
	go build -ldflags="$(LDFLAGS)" -o bin/jalebi-linux-amd64 ./cmd/jalebi

test:
	go test ./...

fmt:
	go fmt ./...

clean:
	rm -rf bin/
