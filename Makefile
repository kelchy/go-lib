SHELL := /bin/bash

.PHONY:

DIR = log redis http

test-%:
	$(MAKE) GOPATH=$${PWD} test -C $* SUB=${SUB}

test:
	@for f in ${DIR}; do $(MAKE) test-$${f}; done
