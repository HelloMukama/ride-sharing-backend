# Ride-Sharing Service Backend

**NB: This project has been tested on Ubuntu 20.04 LTS**

## Overview

This project of a backend Go implementation for ride-sharing is designed for scalability and reliability, demonstrating:

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

## Project Structure

```
ride-sharing-backend/
├── src/
│   ├── main.go         # API entry point
│   ├── matching.go     # Optimized ride-matching algorithm
│   ├── api.go          # External API integration
│   ├── auth.go         # JWT authentication & security
│   ├── caching.go      # Redis caching strategy
│   └── config.env      # Environment variables
├── tests/              # Unit, integration, and load tests
├── Dockerfile          # Containerization setup
├── docker-compose.yml  # Local development setup
├── .github/workflows/ci-cd.yml # CI/CD automation
└── README.md           # Documentation
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

Once you get the token, store it in your `.env` file as follows:

```bash
echo "TOKEN=your_token_here" >> .env
source .env
```

Verify that the token is stored:

```bash
echo $TOKEN
```

> **Note:** The token received upon login becomes invalid upon logout.

### Ride Management

#### List Available Drivers (GET /drivers)

```bash
curl -X GET http://localhost:8080/drivers -H "Authorization: Bearer $TOKEN" | jq
```

#### Request Ride (POST /request-ride)

```bash
curl -X POST http://localhost:8080/request-ride -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{"lat":0.3135,"lng":32.5805}' | jq
```

This response contains a `ride_id`. Use it in the next step.

#### Check Ride Status (GET /ride-status/:id)

```bash
ride_id="your_ride_id_here"
curl -X GET "http://localhost:8080/ride-status/$ride_id" -H "Authorization: Bearer $TOKEN" | jq
```

### Logout (POST /auth/logout)

```bash
curl -X POST http://localhost:8080/auth/logout -H "Authorization: Bearer $TOKEN"
```

## WebSocket Testing

Open another terminal. WebSocket notifications will be triggered for drivers available in the system.

1. Request a ride:

```bash
curl -X POST http://localhost:8080/request-ride -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{"lat":0.3135,"lng":32.5805}' | jq
```

2. In another terminal, test WebSocket notifications:

```bash
docker-compose exec app /ws_test_client driver2
```

If `driver2` is in the area, they will receive a ride notification.

> **Note:** If a ride is requested for a driver and they connect later, they will see all pending ride requests.

## Deployment

### Docker Deployment

```bash
docker build -t ride-sharing-backend .
docker tag ride-sharing-backend your-container-registry/ride-sharing
docker push your-container-registry/ride-sharing
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

## Monitoring

### Access Metrics

1. Prometheus: http://localhost:9090
2. Grafana: http://localhost:3000 (default credentials: admin/admin)

### Sample Metrics Collected

- ride_requests_total
- ride_matching_duration_seconds
- active_drivers
- api_response_time_seconds

## Troubleshooting

### Common Issues

#### Redis Connection Problems:

```bash
docker-compose logs redis
```

#### Database Migration Issues:

```bash
docker-compose exec db psql -U rideuser rides -c "SELECT * FROM pg_migrations"
```

#### Authentication Errors:

```bash
curl -v POST "http://localhost:8080/auth/login" -d '{"username":"testuser"}'
```

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

### Explanation of Steps:

1. **docker-compose down** - Stops and removes containers.
2. **docker volume prune -f** - Cleans up unused volumes.
3. **docker-compose build --no-cache** - Rebuilds containers from scratch.
4. **docker-compose up** - Starts the services fresh.

## Future Improvements

1. Implement real-time ride tracking using WebSockets.
2. Add multi-region failover for high availability.
3. Implement dynamic pricing based on demand.
4. Add a driver rating system.
5. Enhance geospatial queries with additional filters.

