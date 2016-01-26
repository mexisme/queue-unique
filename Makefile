#!/usr/bin/env make -f

GOOS =
GOARCH =

GO = go

GOBUILD = $(GO) build 
GOTEST = ginkgo
GOTEST_ARGS =

GODEPS = github.com/onsi/ginkgo/ginkgo github.com/onsi/gomega

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
	$(GO) get -v $(GODEPS)

vet:
	$(GO) vet -v ./...

clean:
	$(GO) clean
