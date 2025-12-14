# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Enable Go toolchain auto-download for newer Go versions
ENV GOTOOLCHAIN=auto

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN GOTOOLCHAIN=auto CGO_ENABLED=0 GOOS=linux go build -o /app/bin/api ./cmd/api

# Run stage
FROM alpine:3.21

RUN apk --no-cache add ca-certificates=20241010-r0

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/bin/api .

EXPOSE 3000

CMD ["./api"]
