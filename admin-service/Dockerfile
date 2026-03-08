FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN go build -o /admin-service ./cmd/main.go

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /admin-service /usr/local/bin/admin-service
COPY config ./config

EXPOSE 9005

CMD ["admin-service"]
