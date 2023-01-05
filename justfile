PM := "go run cmd/main.go"

@_help:
  just --list --unsorted

# bump dependencies
@bump:
  go get -u ./...
  go mod tidy

# compile go sources for protobuf
@protoc:
  protoc \
    --go_out=. \
    --go_opt=paths=source_relative \
    --go-grpc_out=. \
    --go-grpc_opt=paths=source_relative \
    api/api.proto

# check todos
@todo:
  rg 'TODO' --glob '**/*.go' || echo 'All done!'

# restart daemon
@daemon-restart:
  {{PM}} daemon stop && {{PM}} daemon start

# list db keys
@db-keys:
  bbolt keys ~/.pm/pm.db main

# TODO: remove
# run "long running" task
run-task:
  {{PM}} run --name qmen24-$(date +'%H:%M:%S') sleep 10
