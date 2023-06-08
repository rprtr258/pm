PM := "go run cmd/main.go"

# _help:
#   just --list --unsorted

# open database
db:
	go run github.com/antonmedv/fx@latest ~/.pm/db/procs.json

# run formatters
fmt:
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

# run linter
lint:
	golangci-lint run ./...

# run tests
test:
	go test ./...

# bump dependencies
bump:
	go get -u ./...
	go mod tidy

# compile go sources for protobuf
protoc:
	protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		api/api.proto

# check todos
todo:
	rg 'TODO' --glob '**/*.go' || echo 'All done!'

# restart daemon
daemon-restart:
	{{PM}} daemon stop && {{PM}} daemon start

# TODO: remove
# run "long running" task
run-task:
	{{PM}} run --name qmen24-$(date +'%H:%M:%S') sleep 10
