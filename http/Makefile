SHELL := /bin/bash

.PHONY:

lint:
ifndef SUB
	@echo "SUB not defined defaulting to all"
	$(MAKE) lint -C client
	$(MAKE) lint -C server
else
	$(MAKE) lint -C ${SUB}
endif

test: lint
ifndef SUB
	@echo "SUB not defined defaulting to all"
	$(MAKE) test -C client
	$(MAKE) test -C server
else
	$(MAKE) test -C ${SUB}
endif
