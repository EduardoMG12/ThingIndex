FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy the shared telemetry module first
COPY telemetry/ ./telemetry/

# Copy the legal-parser module
COPY legal-parser/go.mod legal-parser/go.sum ./legal-parser/
WORKDIR /build/legal-parser
RUN go mod download

WORKDIR /build
COPY legal-parser/ ./legal-parser/

WORKDIR /build/legal-parser
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/legal-parser ./cmd/parser

# Final lightweight image
FROM alpine:latest

WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /app/bin/legal-parser .

# Run the binary
ENTRYPOINT ["/app/legal-parser"]
