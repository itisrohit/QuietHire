# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Enable Go toolchain auto-download
ENV GOTOOLCHAIN=auto

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN GOTOOLCHAIN=auto CGO_ENABLED=0 GOOS=linux go build -o /app/bin/crawler .

# Run stage
FROM mcr.microsoft.com/playwright:v1.50.0-jammy

# Install ca-certificates
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/bin/crawler .

# Install Playwright browsers
RUN playwright install chromium

CMD ["./crawler"]
