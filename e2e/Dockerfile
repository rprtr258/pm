FROM golang:1.23.0-alpine AS build
WORKDIR /app
RUN mkdir /.coverdata
COPY go.mod go.sum /app/
RUN go mod download
COPY . .
RUN go build -o e2e/tests/hello-http e2e/tests/hello-http/main.go
RUN go build -cover -o e2e/pm .
ENV GOCOVERDIR=/.coverdata
RUN go test ./e2e/...
CMD ["go", "tool", "covdata", "percent", "-i=/.coverdata"]
