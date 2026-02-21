FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/crawler_app ./cmd/crawler

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/crawler_app .
COPY .env .env

RUN mkdir -p /app/output_pdfs

CMD ["/app/crawler_app"]
