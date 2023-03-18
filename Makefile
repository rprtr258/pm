PM := "go run cmd/main.go"

# _help:
#   just --list --unsorted

# open database
db:
	@go install github.com/antonmedv/fx@latest
	fx ~/.pm/pm.db

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
