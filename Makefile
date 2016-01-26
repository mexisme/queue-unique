#!/usr/bin/env make -f

GOOS =
GOARCH =

GO = go

GOBUILD = $(GO) build 
GOTEST = ginkgo
GOTEST_ARGS =

#BUILDFILES = 

ifdef BUILDFILES
default: build

else
default: test

endif

build: $(BUILDFILES)

$(BUILDFILES): deps vet
	$(GO) build $@

test: deps vet
	$(GOTEST) $(GOTEST_ARGS)

watch: deps vet
	$(GOTEST) $@ $(GOTEST_ARGS)

deps:
	$(GO) get -v ./...

vet:
	$(GO) vet -v ./...

clean:
	$(GO) clean
