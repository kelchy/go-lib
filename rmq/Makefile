SHELL := /bin/bash

.PHONY:

test:
ifndef ${SUB}
	@echo "SUB not defined defaulting to consumer"
	$(MAKE) test -C consumer
else
	$(MAKE) test -C ${SUB}
endif
