PM := "go run cmd/main.go"

.PHONY: help
help: # show list of all commands
	@grep -E '^[a-zA-Z_-]+:.*?# .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?# "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

db: # open database
	go run github.com/antonmedv/fx@latest ~/.pm/db/procs.json

fmt: # run formatters
	@go install mvdan.cc/gofumpt@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	go fmt ./...
	gofumpt -l -w .
	goimports -l -w -local $(shell head -n1 go.mod | cut -d' ' -f2) .
	# go run -mod=mod golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment -fix ./... || \
	# go run -mod=mod golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment -fix ./... || \
	# go run -mod=mod golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment -fix ./... || \
	# go run -mod=mod golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment -fix ./...
	go mod tidy

lint: # run linter
	golangci-lint run ./...

test: # run tests
	@go run gotest.tools/gotestsum@latest ./...

bump: # bump dependencies
	go get -u ./...
	go mod tidy

protoc: # compile go sources for protobuf
	protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		api/api.proto

todo: # check todos
	rg 'TODO' --glob '**/*.go' || echo 'All done!'

daemon-restart: # restart daemon
	{{PM}} daemon stop && {{PM}} daemon start

run-task: # TODO: remove # run "long running" task
	{{PM}} run --name qmen24-$(date +'%H:%M:%S') sleep 10
