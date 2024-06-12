FROM golang:1.21-alpine AS build
WORKDIR /app
COPY go.mod go.sum /app/
RUN go mod download
COPY . .
RUN go build -o /pm ./tests/main.go
RUN go build -o hello-http tests/hello-http/main.go
ENTRYPOINT ["/pm", "test"]
CMD ["all"]
