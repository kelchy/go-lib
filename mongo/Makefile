SHELL := /bin/bash

.PHONY:

# this will install binary in ${GOPATH}
${GOPATH}/bin/golangci-lint:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

lint: ${GOPATH}/bin/golangci-lint
	${GOPATH}/bin/golangci-lint run ./...

test: lint
	go test -coverprofile coverage.out .
	go tool cover -func=coverage.out
