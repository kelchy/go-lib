SHELL := /bin/bash

.PHONY:

# this will install binary in ${GOPATH}
$(GOPATH)/bin/golint:
	go install golang.org/x/lint/golint@v0.0.0-20201208152925-83fdc39ff7b5

lint: $(GOPATH)/bin/golint
	${GOPATH}/bin/golint -set_exit_status ./...

test: lint
	$(MAKE) test -C client
	$(MAKE) test -C server

ifndef ${SUB}
	@echo "SUB not defined defaulting to client"
	$(MAKE) test -C client
else
	$(MAKE) test -C ${SUB}
endif
