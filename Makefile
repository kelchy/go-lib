SHELL := /bin/bash

.PHONY:

DIR = log redis

test-%:
	$(MAKE) GOPATH=$${PWD} test -C $*

test:
	@for f in ${DIR}; do $(MAKE) test-$${f}; done
