# Auto generated binary variables helper managed by https://github.com/bwplotka/bingo v0.4.0. DO NOT EDIT.
# All tools are designed to be build inside $GOBIN.
BINGO_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
GOPATH ?= $(shell go env GOPATH)
GOBIN  ?= $(firstword $(subst :, ,${GOPATH}))/bin
GO     ?= $(shell which go)

# Below generated variables ensure that every time a tool under each variable is invoked, the correct version
# will be used; reinstalling only if needed.
# For example for bingo variable:
#
# In your main Makefile (for non array binaries):
#
#include .bingo/Variables.mk # Assuming -dir was set to .bingo .
#
#command: $(BINGO)
#	@echo "Running bingo"
#	@$(BINGO) <flags/args..>
#
BINGO := $(GOBIN)/bingo-v0.4.0
$(BINGO): $(BINGO_DIR)/bingo.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/bingo-v0.4.0"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=bingo.mod -o=$(GOBIN)/bingo-v0.4.0 "github.com/bwplotka/bingo"

PROTOC_GEN_BUF_BREAKING := $(GOBIN)/protoc-gen-buf-breaking-v0.41.0
$(PROTOC_GEN_BUF_BREAKING): $(BINGO_DIR)/protoc-gen-buf-breaking.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/protoc-gen-buf-breaking-v0.41.0"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=protoc-gen-buf-breaking.mod -o=$(GOBIN)/protoc-gen-buf-breaking-v0.41.0 "github.com/bufbuild/buf/cmd/protoc-gen-buf-breaking"

PROTOC_GEN_BUF_LINT := $(GOBIN)/protoc-gen-buf-lint-v0.41.0
$(PROTOC_GEN_BUF_LINT): $(BINGO_DIR)/protoc-gen-buf-lint.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/protoc-gen-buf-lint-v0.41.0"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=protoc-gen-buf-lint.mod -o=$(GOBIN)/protoc-gen-buf-lint-v0.41.0 "github.com/bufbuild/buf/cmd/protoc-gen-buf-lint"

PROTOC_GEN_GO := $(GOBIN)/protoc-gen-go-v1.26.0
$(PROTOC_GEN_GO): $(BINGO_DIR)/protoc-gen-go.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/protoc-gen-go-v1.26.0"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=protoc-gen-go.mod -o=$(GOBIN)/protoc-gen-go-v1.26.0 "google.golang.org/protobuf/cmd/protoc-gen-go"

