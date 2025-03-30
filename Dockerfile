# ==================== BUILDER STAGE ====================
FROM golang:1.24.1 as builder

WORKDIR /app

# 1. Copy module files first for better caching
COPY go.mod go.sum ./

# 2. Download dependencies
RUN go mod download

# 3. Copy source code
COPY src/ src/

# 4. Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /ride-sharing-backend ./src

# 5. In the builder stage (after COPY src/)
RUN CGO_ENABLED=0 GOOS=linux go build -o /ride-sharing-backend ./src && \
    go build -o /ws_test_client ./src/client/ws_test_client.go




# ==================== RUNTIME STAGE ====================
FROM alpine:latest

# 1. Install dependencies
RUN apk add --no-cache ca-certificates

# 2. Copy binary
COPY --from=builder /ride-sharing-backend /

# 3. Copy configuration files
COPY .env .
#COPY config.env .  we are using .env in base dir

# 4
COPY --from=builder /ws_test_client /

# 5. Expose and run
EXPOSE 8080
CMD ["/ride-sharing-backend"]
