export GO111MODULE=on

# defaults
OS=$(shell go env GOOS)## Target OS
ARCH=$(shell go env GOARCH)## Target architecture
BIN=./bin## Binary location

# env
GOOS=GOOS=$(OS)
GOARCH=GOARCH=$(ARCH)
CGO=CGO_ENABLED=0
ENV=env $(GOOS) $(GOARCH) $(CGO)

# gobuild
GOCMD=go
GOBUILD=$(GOCMD) build

# flags with no arguments
NOARGFLAGS=-trimpath

# output extension
ifeq ($(OS), windows)
    EXT=.exe
else
    EXT=
endif

OUTPUT=$(BIN)/$@_$(OS)_$(ARCH)$(EXT)## Output location

# ld flags
LDFLAGS=-s -w

.PHONY: all full agent raido clean proto_update proto_gen proto_lint help
.DEFAULT_GOAL := help

## agent: Build binary of agent for the specified OS and architecture 
agent:
	$(ENV) $(GOBUILD) $(NOARGFLAGS) -o $(OUTPUT) -ldflags "$(LDFLAGS)" ./cmd/agent/*.go

## raido: Build binary of raido for the specified OS and architecture
raido:
	$(ENV) $(GOBUILD) $(NOARGFLAGS) -o $(OUTPUT) -ldflags "$(LDFLAGS)" ./cmd/raido/*.go

## full: Build binary of agent and raido for the specified OS and architecture 
full: | agent raido

## all: Build binaries for every OS/ARCH pair listed in the Makefile
all: 
	$(MAKE) full OS=windows ARCH=amd64
	$(MAKE) full OS=darwin ARCH=amd64
	$(MAKE) full OS=linux ARCH=amd64

## clean: Remove all binaries
clean:
	rm -vf $(BIN)/*

## proto_update: Update proto files
proto_update:
	buf dep update ./proto

## proto_gen: Generate proto files
proto_gen:
	buf generate ./proto

## proto_lint: Lint proto files
proto_lint:
	buf lint ./proto

# reference:  https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
## help: Print this message
help:
	@echo "Raido Makefile"
	@echo ""
	@echo "Targets:"
	@grep -E '^## [a-zA-Z_-]+: .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ": |## "}; {printf "  %-30s %s\n", $$2, $$3}'
	@echo ""
	@echo "Variables (KEY=DEFAULT):"
	@grep -E '^[a-zA-Z_-]+=.+?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = "## "}; {printf "  %-30s %s\n", $$1, $$2}'