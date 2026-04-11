.PHONY: build test lint clean

build:
	go build -o bin/{{ project_name }} ./cmd/...

test:
	go test ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/
