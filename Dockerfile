# Build Stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o kumquat ./cmd/kumquat/

# Run Stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/kumquat .
COPY --from=builder /app/config ./config

EXPOSE 8080

CMD ["./kumquat", "server"]

