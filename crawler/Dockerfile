FROM golang:alpine AS builder

WORKDIR /build

# Copy the shared telemetry module first
COPY telemetry/ ./telemetry/

# Copy the crawler module
COPY crawler/go.mod crawler/go.sum ./crawler/
WORKDIR /build/crawler
RUN go mod download

WORKDIR /build
COPY crawler/ ./crawler/

WORKDIR /build/crawler
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/crawler_app ./cmd/crawler

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/crawler_app .

RUN mkdir -p /app/output_pdfs

CMD ["/app/crawler_app"]
