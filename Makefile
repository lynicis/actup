.PHONY: build test lint clean install sec vuln

BINARY_NAME=actup

build:
	go build -o $(BINARY_NAME) .

install:
	go install .

test:
	go test -v -race ./...

lint:
	go tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint run ./...

clean:
	rm -f $(BINARY_NAME)
	go clean

fmt:
	go tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint fmt ./...

mod-tidy:
	go mod tidy

sec:
	go tool github.com/securego/gosec/v2/cmd/gosec ./...

vuln:
	go tool golang.org/x/vuln/cmd/govulncheck ./...
