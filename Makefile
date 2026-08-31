ROOT := $(realpath $(dir $(realpath $(firstword $(MAKEFILE_LIST)))))

.DEFAULT_GOAL := ci

.PHONY: build test validate ci clean

build:
	@bash $(ROOT)/scripts/build

test:
	@bash $(ROOT)/scripts/test

validate:
	@bash $(ROOT)/scripts/validate

ci: build test validate

clean:
	@rm -rf $(ROOT)/bin $(ROOT)/scripts/.version_env
