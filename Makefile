# Default shell
SHELL := /bin/bash

# General
WORKDIR = $(PWD)

# Go parameters
GOCMD = go
GOTEST = $(GOCMD) test -v --examples

# Coverage
COVERAGE_REPORT = coverage.txt
COVERAGE_PROFILE = profile.out
COVERAGE_MODE = atomic

ifneq ($(origin CI), undefined)
	WORKDIR := $(GOPATH)/src/gopkg.in/src-d/go-git.v4
endif


test:
	cd $(WORKDIR); \
	$(GOTEST) ./...

test-coverage:
	cd $(WORKDIR); \
	echo "" > $(COVERAGE_REPORT); \
	for dir in `find . -name "*.go" | grep -o '.*/' | sort | uniq`; do \
		$(GOTEST) $$dir -coverprofile=$(COVERAGE_PROFILE) -covermode=$(COVERAGE_MODE); \
		if [ $$? != 0 ]; then \
			exit 2; \
		fi; \
		if [ -f $(COVERAGE_PROFILE) ]; then \
			cat $(COVERAGE_PROFILE) >> $(COVERAGE_REPORT); \
			rm $(COVERAGE_PROFILE); \
		fi; \
	done; \
