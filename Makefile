.PHONY: build test lint clean install

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

install:
	$(GO_CMD) install .

fmt:
	$(GO_CMD) fmt ./...

mod-tidy:
	$(GO_CMD) mod tidy
