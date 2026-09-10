.PHONY: build test lint clean install sec vuln

BINARY_NAME=actup
GO_CMD=go

build:
	$(GO_CMD) build -o $(BINARY_NAME) .

test:
	$(GO_CMD) test -v -race ./...

lint:
	go tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint run ./...

clean:
	rm -f $(BINARY_NAME)
	$(GO_CMD) clean

fmt:
	$(GO_CMD) fmt ./...

mod-tidy:
	$(GO_CMD) mod tidy

sec:
	go tool github.com/securego/gosec/v2/cmd/gosec ./...

vuln:
	go tool golang.org/x/vuln/cmd/govulncheck ./...
