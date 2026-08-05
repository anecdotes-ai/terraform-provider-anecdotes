# Copyright (c) Anecdotes AI
# SPDX-License-Identifier: MPL-2.0

HOSTNAME=registry.terraform.io
NAMESPACE=anecdotes-ai
NAME=anecdotes
BINARY=terraform-provider-${NAME}
VERSION=1.0.0
OS_ARCH=$(shell go env GOOS)_$(shell go env GOARCH)

default: install

.PHONY: build
build:
	go build -buildvcs=false -o ${BINARY}

.PHONY: install
install: build
	mkdir -p ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}
	mv ${BINARY} ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}

.PHONY: test
test:
	go test -race -v -cover -timeout=120s -parallel=4 ./...

.PHONY: testacc
testacc:
	TF_ACC=1 go test -v -cover -timeout 120m ./...

.PHONY: lint
lint:
	golangci-lint run

.PHONY: fmt
fmt:
	gofmt -s -w .
	terraform fmt -recursive examples/

.PHONY: docs
docs:
	go generate ./...

.PHONY: clean
clean:
	rm -f ${BINARY}
	rm -rf ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}

.PHONY: deps
deps:
	go mod download
	go mod tidy

.PHONY: release
release:
	goreleaser release --clean

.PHONY: release-snapshot
release-snapshot:
	goreleaser release --snapshot --clean
