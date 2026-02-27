FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy the shared telemetry module first
COPY telemetry/ ./telemetry/

# Copy the populator module
COPY populator/go.mod populator/go.sum ./populator/
WORKDIR /build/populator
RUN go mod download

WORKDIR /build
COPY populator/ ./populator/

WORKDIR /build/populator
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/populator ./cmd/populator

# Final lightweight image
FROM alpine:latest

WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /app/bin/populator .

# Run the binary
ENTRYPOINT ["/app/populator"]
