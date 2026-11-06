SHELL := /bin/bash

.PHONY:

lint:
ifndef SUB
	@echo "SUB not defined defaulting to consumer"
	$(MAKE) lint -C consumer
else
	$(MAKE) lint -C ${SUB}
endif

test:
ifndef ${SUB}
	@echo "SUB not defined defaulting to consumer"
	$(MAKE) test -C consumer
else
	$(MAKE) test -C ${SUB}
endif
