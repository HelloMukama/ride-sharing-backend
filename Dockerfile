# ==================== BUILDER STAGE ====================
FROM golang:1.24.1 as builder

WORKDIR /app

# 1. Copy module files first for better caching
COPY go.mod go.sum ./

# 2. Download dependencies
RUN go mod download

# 3. Copy source code
COPY src/ src/

# 4. Build both main app and test client in one layer
RUN CGO_ENABLED=0 GOOS=linux go build -o /ride-sharing-backend ./src && \
    CGO_ENABLED=0 GOOS=linux go build -o /ws_test_client ./src/client/ws_test_client.go

# ==================== RUNTIME STAGE ====================
FROM alpine:latest

# 1. Install dependencies (add curl for healthchecks)
RUN apk add --no-cache ca-certificates curl

# 2. Copy binaries
COPY --from=builder /ride-sharing-backend /ws_test_client /

# 3. Copy configuration files
COPY .env .

# 4. Expose and run
EXPOSE 8080
CMD ["/ride-sharing-backend"]
