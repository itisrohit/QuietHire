# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Enable Go toolchain auto-download
ENV GOTOOLCHAIN=auto

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build the application (note: main.go is in root, not cmd/)
RUN GOTOOLCHAIN=auto CGO_ENABLED=0 GOOS=linux go build -o /app/bin/proxy-manager ./main.go

# Run stage
FROM alpine:3.21

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/bin/proxy-manager .

EXPOSE 8002

CMD ["./proxy-manager"]
