.PHONY: build test lint clean install sec vuln

BINARY_NAME=actup
GO_CMD=go

build:
	$(GO_CMD) build -o $(BINARY_NAME) .

test:
	$(GO_CMD) test -v -race ./...

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY_NAME)
	$(GO_CMD) clean

fmt:
	$(GO_CMD) fmt ./...

mod-tidy:
	$(GO_CMD) mod tidy

sec:
	gosec ./...

vuln:
	govulncheck ./...
