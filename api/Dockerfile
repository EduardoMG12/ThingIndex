FROM golang:1.25.13-alpine AS builder

WORKDIR /build

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/api ./cmd/api

FROM alpine:latest

RUN apk add --no-cache poppler-utils

WORKDIR /app

COPY --from=builder /app/bin/api .

EXPOSE 8080

ENTRYPOINT ["/app/api"]
