# ==================== BUILDER STAGE ====================
FROM golang:1.24.1 as builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY src/ src/

# Build to absolute path in builder
RUN CGO_ENABLED=0 GOOS=linux go build -o /usr/local/bin/ride-sharing-backend ./...

# ==================== RUNTIME STAGE ====================
FROM alpine:latest

# 1. Install dependencies first (better layer caching)
RUN apk add --no-cache ca-certificates

# 2. Copy binary from standard Unix binary location
COPY --from=builder /usr/local/bin/ride-sharing-backend /usr/local/bin/

# 3. Create app directory and copy env
WORKDIR /app
COPY .env .

EXPOSE 8080

# 4. Use absolute path to binary
CMD ["/usr/local/bin/ride-sharing-backend"]
