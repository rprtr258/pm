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
