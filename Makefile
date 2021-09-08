include .bingo/Variables.mk

.DEFAULT_GOAL=build

GOLANGCI_LINT=go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.42.1
GOVVV=go run github.com/ahmetb/govvv@v0.3.0 
BUF=go run github.com/bufbuild/buf/cmd/buf@v0.41.0

HEAD_SHORT ?= $(shell git rev-parse --short HEAD)

BIN_BUILD_FLAGS?=CGO_ENABLED=0
BIN_VERSION?="git"
GOVVV_FLAGS=$(shell $(GOVVV) -flags -version $(BIN_VERSION) -pkg $(shell go list ./buildinfo))


lint: 
	$(GOLANGCI_LINT) run
.PHONYY: lint


build: 
	$(BIN_BUILD_FLAGS) go build -ldflags="${GOVVV_FLAGS}" .
.PHONY: build

install: 
	$(BIN_BUILD_FLAGS) go install -ldflags="${GOVVV_FLAGS}" .
.PHONY: install

protos: $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC) clean-protos
	$(BUF) generate --template '{"version":"v1beta1","plugins":[{"name":"go","out":"gen","opt":"paths=source_relative","path":$(PROTOC_GEN_GO)},{"name":"go-grpc","out":"gen","opt":"paths=source_relative","path":$(PROTOC_GEN_GO_GRPC)}]}'
.PHONY: protos

clean-protos:
	find . -type f -name '*.pb.go' -delete
	find . -type f -name '*pb_test.go' -delete
.PHONY: clean-protos
