FROM golang:1.23 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o shell ./cmd/shell

FROM debian:bullseye-slim

WORKDIR /app

COPY --from=builder /app/shell .

ENTRYPOINT ["/app/shell"]

EXPOSE 80
