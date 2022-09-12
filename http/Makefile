SHELL := /bin/bash

.PHONY:

test:
ifndef ${SUB}
	@echo "SUB not defined defaulting to client"
	$(MAKE) test -C client
else
	$(MAKE) test -C ${SUB}
endif
