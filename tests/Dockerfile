FROM golang:1.19-alpine AS build
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o /pm ./cmd/main.go
CMD ["/pm", "daemon", "run"]
