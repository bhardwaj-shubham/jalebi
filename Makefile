.PHONY: run build test fmt clean

run:
	go run ./cmd/jalebi

build:
	go build -o bin/jalebi ./cmd/jalebi

test:
	go test ./...

fmt:
	go fmt ./...

clean:
	rm -rf bin/
