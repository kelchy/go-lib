SHELL := /bin/bash

.PHONY:
# this will install binary in ${GOPATH}
$(GOPATH)/bin/golint:
	go install golang.org/x/lint/golint@v0.0.0-20201208152925-83fdc39ff7b5

lint: $(GOPATH)/bin/golint
	${GOPATH}/bin/golint -set_exit_status ./...

test: lint
	go test -coverprofile coverage.out ./...
	go tool cover -func=coverage.out
