# Ride-Sharing Service Backend

**NB: This project has been tested on Ubuntu 20.04 LTS**

## Overview

This project is a backend Go implementation for ride-sharing, designed for scalability and reliability, demonstrating:

- **Advanced Algorithm Design**: Efficiently assigns drivers to riders using optimized geospatial queries.
- **API Integrations**: Fetches real-time driver locations from geolocation APIs with caching strategies.
- **Deployment**: Docker deployment with auto-scaling and CI/CD automation.
- **Security & Performance Best Practices**: Implements JWT authentication, rate-limiting, and observability.

## Features

- **Optimized Ride-Matching Algorithm**: Uses geospatial indexing with Redis Geo for efficient distance calculations.
- **Real-Time Geolocation Fetching**: Integrates with OpenStreetMap with rate-limited caching.
- **Deployment**: Containerized with Docker, scalable microservices, and automated CI/CD pipeline.
- **Security & Performance**: Implements JWT authentication, rate-limiting, and monitoring with Prometheus and Grafana.
- **Additional Features**: Redis caching, WebSockets for real-time updates, and payment processing.

## Quick Start

1. Start the backend service with Docker:
   ```bash
   docker-compose up --build
   ```
2. Access endpoints:
   - API: [http://localhost:8080](http://localhost:8080)
   - Metrics: [http://localhost:8080/metrics](http://localhost:8080/metrics)
   - Prometheus: [http://localhost:9090](http://localhost:9090)
   - Grafana: [http://localhost:3000](http://localhost:3000)

## Monitoring and Metrics

### Prometheus
- Accessible at [http://localhost:9090](http://localhost:9090)
- Prometheus collects and stores time-series data.
- You can query metrics like `ride_requests_total` and `api_response_time_seconds`.

### Grafana
- Accessible at [http://localhost:3000](http://localhost:3000) (default credentials: admin/admin)
- Grafana provides dashboards for visualizing performance metrics.
- Expect graphs on ride-matching duration, active drivers, and system load.

## Connecting to PostgreSQL

If you need direct database access for debugging or testing, you can connect to PostgreSQL inside the Docker container:
```bash
docker exec -it ride-sharing-backend_db_1 psql -U rideuser -d rides
```

## Project Structure

```
ride-sharing-backend/
├── docker-compose.yml
├── Dockerfile
├── go.mod
├── go.sum
├── LICENSE
├── monitoring
│   ├── grafana-dashboard.json
│   └── prometheus.yml
├── readme.md
├── src
│   ├── api.go
│   ├── auth.go
│   ├── caching.go
│   ├── client
│   │   └── ws_test_client.go
│   ├── config.env
│   ├── database.go
│   ├── init.go
│   ├── main.go
│   ├── matching.go
│   ├── migrations
│   │   └── 001_init_schema.up.sql
│   ├── notifications.go
│   ├── payments.go
│   └── testutils.go
├── tests
│   ├── auth_test.go
│   ├── drivers_test.go
│   ├── load_test.go
│   ├── ride_test.go
│   └── ws_test.go
```

## Installation & Setup

### 1. Clone the Repository

```bash
git clone https://github.com/hellomukama/ride-sharing-backend.git
cd ride-sharing-backend
```

### 2. Set Up Environment Variables

Copy the contents of `.env.example` into a new `.env` file and update values where necessary:

```bash
cp .env.example .env
```

### 3. Run with Docker

```bash
docker-compose up --build
```

## API Endpoints

### Authentication

#### Login (POST /auth/login)
```bash
curl -s -X POST http://localhost:8080/auth/login -H "Content-Type: application/json" -d '{"username":"testuser","user_id":123,"role":"rider"}' | jq
```

### Ride Management

#### List Available Drivers (GET /drivers)
```bash
curl -X GET http://localhost:8080/drivers -H "Authorization: Bearer $TOKEN" | jq
```

#### Request Ride (POST /request-ride)
```bash
curl -X POST http://localhost:8080/request-ride -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{"lat":0.3135,"lng":32.5805}' | jq
```

### Optimized Ride-Matching Algorithm (Redis Geo)

You can test Redis Geo indexing manually:

```bash
docker-compose exec redis redis-cli
```

#### Sample Redis Commands:
```bash
127.0.0.1:6379> GEOADD drivers 32.5811 0.3135 "driver1"
(integer) 1
127.0.0.1:6379> GEORADIUS drivers 32.5811 0.3135 5 km
1) "driver1"
127.0.0.1:6379> GEOADD drivers 32.5811 0.3135 "driver3"
(integer) 1
127.0.0.1:6379> GEORADIUS drivers 32.5811 0.3135 5 km
1) "driver1"
2) "driver3"
```

Use `GEORADIUS` to find drivers within a given distance.

## Deployment

### Docker Deployment

```bash
# Build the Docker image
docker build -t ride-sharing-backend .

# Tag it for Docker Hub
docker tag ride-sharing-backend hellomukama/ride-sharing-backend:latest

# Push it to Docker Hub
docker push hellomukama/ride-sharing-backend:latest
```

## Testing

### Running Unit Tests
```bash
go test ./tests/...
```

### Load Testing Example
```bash
wrk -t4 -c100 -d60s http://localhost:8080/drivers -H "Authorization: Bearer $TOKEN"
```

## Troubleshooting

### Checking Service Health
```bash
curl http://localhost:8080/health
```

### Docker Issues
If you encounter issues running the service with Docker, try the following:
```bash
docker-compose down
docker volume prune -f
docker-compose build --no-cache
docker-compose up
```

## Future Improvements
1. Implement real-time ride tracking using WebSockets.
2. Add multi-region failover for high availability.
3. Implement dynamic pricing based on demand.
4. Add a driver rating system.
5. Enhance geospatial queries with additional filters.

