# ==========================================
# Multi-Stage Build untuk ASM Golang App
# ==========================================

# 1. Builder Stage
FROM golang:1.26-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Compile binary statis Linux
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o asm .

# ------------------------------------------
# 2. Final Minimal Runner Stage
FROM alpine:3.20 AS runner

# Install runtime utilities & set timezone
RUN apk add --no-cache ca-certificates tzdata curl && \
    cp /usr/share/zoneinfo/Asia/Jakarta /etc/localtime && \
    echo "Asia/Jakarta" > /etc/timezone

WORKDIR /app

# Copy binary dari stage builder
COPY --from=builder /app/asm /app/asm

# Copy static web assets
COPY web/ /app/web/

# Set environment defaults
ENV PORT=8080 \
    GIN_MODE=release \
    TZ=Asia/Jakarta

EXPOSE 8080

# Healthcheck endpoint
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/ || exit 1

ENTRYPOINT ["/app/asm"]
