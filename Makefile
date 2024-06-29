RED    := $(shell tput -Txterm setaf 1)
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
BLUE   := $(shell tput -Txterm setaf 4)
VIOLET := $(shell tput -Txterm setaf 5)
CYAN   := $(shell tput -Txterm setaf 6)
WHITE  := $(shell tput -Txterm setaf 7)
RESET  := $(shell tput -Txterm sgr0)

PM := go run main.go

CURDIR=$(shell pwd)
BINDIR=${CURDIR}/bin

GOLANGCILINTVER=1.58.1
GOLANGCILINTBIN=${BINDIR}/golangci-lint_${GOLANGCILINTVER}

GOCRITICVER=v0.11.4
GOCRITICBIN=${BINDIR}/gocritic_${GOCRITICVER}


.PHONY: help
help: # show list of all commands
	@awk 'BEGIN {FS = ":.*?# "} { \
		if (/^[/%.a-zA-Z0-9_-]+:.*?#.*$$/) \
			{ printf "  ${YELLOW}%-30s${RESET}${WHTIE}%s${RESET}\n", $$1, $$2} \
		else if (/^## .*$$/) \
			{ printf "${CYAN}%s:${RESET}\n", substr($$1,4)} \
		}' $(MAKEFILE_LIST)

bindir:
	@mkdir -p ${BINDIR}



## Development

bump: # bump dependencies
	go get -u ./...
	go mod tidy

todo: # check todos
	rg 'TODO' --glob '**/*.go' || echo 'All done!'

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

run-task: # TODO: remove # run "long running" task
	${PM} run --name qmen24-$(date +'%H:%M:%S') sleep 10


## CI

install-linter: bindir
	@test -f ${GOLANGCILINTBIN} || (\
		wget https://github.com/golangci/golangci-lint/releases/download/v${GOLANGCILINTVER}/golangci-lint-${GOLANGCILINTVER}-linux-amd64.tar.gz -O ${GOLANGCILINTBIN}.tar.gz && \
		tar xvf ${GOLANGCILINTBIN}.tar.gz -C ${BINDIR} && \
		mv ${BINDIR}/golangci-lint-${GOLANGCILINTVER}-linux-amd64/golangci-lint ${GOLANGCILINTBIN} && \
		rm -rf ${BINDIR}/${GOLANGCILINTBIN}.tar.gz ${BINDIR}/golangci-lint-${GOLANGCILINTVER}-linux-amd64 \
	)
	@test -f ${GOCRITICBIN} || (\
		env GOPATH=${BINDIR} go install github.com/go-critic/go-critic/cmd/gocritic@${GOCRITICVER} && \
		mv ${BINDIR}/bin/gocritic ${GOCRITICBIN} && \
		rmdir ${BINDIR}/bin \
	)


# TODO: pin deadcode
lint-go: install-linter # run go linter
	@${GOLANGCILINTBIN} run ./...
	gocritic check -enableAll -disable='rangeValCopy,hugeParam,unnamedResult' ./...
	# deadcode . # false positives

lint: lint-go # run all linters

docs: # generate docs
	jsonnet --string --multi . docs.jsonnet
	go run github.com/eliben/static-server@latest


## Test

test: # run tests
	@go build .
	@go run gotest.tools/gotestsum@latest --format dots-v2 ./...

test-e2e: # run integration tests
	@go run gotest.tools/gotestsum@latest --format dots-v2 ./e2e/...

test-e2e-docker: # run integration tests in docker
	@docker build -t pm-e2e --file e2e/Dockerfile .
	@docker run pm-e2e
