FROM golang:1.21-alpine AS build
RUN go install gotest.tools/gotestsum@latest
WORKDIR /app
COPY go.mod go.sum /app/
RUN go mod download
COPY . .
RUN go build -o e2e/tests/hello-http e2e/tests/hello-http/main.go
RUN go build -o pm . && mv pm e2e/pm
CMD ["gotestsum", "--format", "dots-v2", "./e2e/..."]
