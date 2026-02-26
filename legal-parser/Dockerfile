FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/legal-parser ./cmd/parser

# Final lightweight image
FROM alpine:latest

WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /app/bin/legal-parser .

# Run the binary
ENTRYPOINT ["/app/legal-parser"]
