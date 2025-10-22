FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o httpssh .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

RUN update-ca-certificates

WORKDIR /app

COPY --from=builder /app/httpssh .
COPY config.json /etc/httpssh/config.json

CMD ["./httpssh", "-config", "/etc/httpssh/config.json"]
