RED    := $(shell tput -Txterm setaf 1)
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
BLUE   := $(shell tput -Txterm setaf 4)
VIOLET := $(shell tput -Txterm setaf 5)
CYAN   := $(shell tput -Txterm setaf 6)
WHITE  := $(shell tput -Txterm setaf 7)
RESET  := $(shell tput -Txterm sgr0)

PM := go run cmd/main.go

CURDIR=$(shell pwd)
BINDIR=${CURDIR}/bin

GOLANGCILINTVER=1.53.2
GOLANGCILINTBIN=${BINDIR}/golangci-lint_${GOLANGCILINTVER}

PROTOLINTVER=v0.44.0
PROTOLINTBIN=${BINDIR}/protolint_${PROTOLINTVER}

PROTOCVER=3.15.8
PROTOCBIN=${BINDIR}/protoc_${PROTOCVER}

PROTOCGENGOVER=v1.30.0
PROTOCGENGOBIN=${BINDIR}/protoc-gen-go

PROTOCGENGOGRPCVER=v1.3.0
PROTOCGENGOGRPCBIN=${BINDIR}/protoc-gen-go-grpc


.PHONY: help
help: # show list of all commands
	@awk 'BEGIN {FS = ":.*?# "} { \
	if (/^[/%.a-zA-Z0-9_-]+:.*?#.*$$/) \
		{ printf "  ${YELLOW}%-30s${RESET}${WHTIE}%s${RESET}\n", $$1, $$2} \
	else if (/^## .*$$/) \
		{ printf "${CYAN}%s:${RESET}\n", substr($$1,4)} \
	}' $(MAKEFILE_LIST)

bindir:
	mkdir -p ${BINDIR}

## Development

db: # open database
	go run github.com/antonmedv/fx@latest ~/.pm/db/procs.json

bump: # bump dependencies
	go get -u ./...
	go mod tidy

todo: # check todos
	rg 'TODO' --glob '**/*.go' || echo 'All done!'

install-protoc: bindir
	@test -f ${PROTOCBIN} || \
		(curl -LO https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOCVER}/protoc-${PROTOCVER}-linux-x86_64.zip && \
		unzip protoc-${PROTOCVER}-linux-x86_64.zip -d ${BINDIR} && \
		mv ${BINDIR}/bin/protoc ${PROTOCBIN} && \
		rmdir ${BINDIR}/bin && \
		rm protoc-${PROTOCVER}-linux-x86_64.zip)

install-protoc-gen-go: bindir
	@test -f ${PROTOCGENGOBIN} || \
		(GOBIN=${BINDIR} go install google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOCGENGOVER})

install-protoc-gen-go-grpc: bindir
	@test -f ${PROTOCGENGOGRPCBIN} || \
		(GOBIN=${BINDIR} go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@${PROTOCGENGOGRPCVER})

gen-proto: install-protoc install-protoc-gen-go install-protoc-gen-go-grpc # compile go sources for protobuf
	# service proto
	rm api/*.pb.go || true
	${PROTOCBIN} \
		--plugin=${PROTOCGENGOGRPCBIN} \
		--plugin=${PROTOCGENGOBIN} \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		api/api.proto

## CI

fmt: # run formatters
	@go install mvdan.cc/gofumpt@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	go fmt ./...
	gofumpt -l -w .
	goimports -l -w -local $(shell head -n1 go.mod | cut -d' ' -f2) .
	# go run -mod=mod golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment -fix ./... || \
	# go run -mod=mod golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment -fix ./... || \
	# go run -mod=mod golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment -fix ./... || \
	# go run -mod=mod golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment -fix ./... || \
	# go run -mod=mod golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment -fix ./...
	go mod tidy

install-protolint: bindir
	@test -f ${PROTOLINTBIN} || \
		(GOBIN=${BINDIR} go install github.com/yoheimuta/protolint/cmd/protolint@${PROTOLINTVER} && \
		mv ${BINDIR}/protolint ${PROTOLINTBIN})

lint-proto: install-protolint # run proto linter
	@${PROTOLINTBIN} lint api/api.proto

install-linter: bindir
	@test -f ${GOLANGCILINTBIN} || \
		(wget https://github.com/golangci/golangci-lint/releases/download/v${GOLANGCILINTVER}/golangci-lint-${GOLANGCILINTVER}-linux-amd64.tar.gz -O ${GOLANGCILINTBIN}.tar.gz && \
		tar xvf ${GOLANGCILINTBIN}.tar.gz -C ${BINDIR} && \
		mv ${BINDIR}/golangci-lint-${GOLANGCILINTVER}-linux-amd64/golangci-lint ${GOLANGCILINTBIN} && \
		rm -rf ${BINDRI}/${GOLANGCILINTBIN}.tar.gz ${BINDIR}/golangci-lint-${GOLANGCILINTVER}-linux-amd64)

lint-go: install-linter # run go linter
	@${GOLANGCILINTBIN} run ./...

lint: lint-proto lint-go # run all linters

## Test

test: # run tests
	@go run gotest.tools/gotestsum@latest ./...

test-e2e: # run integration tests
	@go run tests/main.go test all

## Run & test

watch-daemon: # run daemon and restart on file changes
	reflex --start-service --regex='\.go$$' -- ${PM} daemon run

run-task: # TODO: remove # run "long running" task
	${PM} run --name qmen24-$(date +'%H:%M:%S') sleep 10
