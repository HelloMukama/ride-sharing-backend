# Ride-Sharing Service Backend

## Overview

This project is a high-performance ride-sharing backend designed for scalability and reliability, demonstrating:

- Advanced Algorithm Design: Efficiently assigning drivers to riders using optimized geospatial queries
- API Integrations: Fetching real-time driver locations from geolocation APIs with caching strategies
- Deployment: Docker deployment with auto-scaling and CI/CD automation
- Security & Performance Best Practices: JWT authentication, rate-limiting, and observability

## Features

- Optimized Ride-Matching Algorithm: Uses geospatial indexing using Redis Geo for efficient distance calculations
- Real-Time Geolocation Fetching: Integrates with OpenStreetMap with rate-limited caching
- Deployment: Containerized with Docker, scalable microservices, and automated CI/CD pipeline
- Secure & Performant: Implements JWT authentication, rate-limiting, and monitoring (Prometheus, Grafana)
- Additional Features: Redis caching, WebSockets for real-time updates, and payment processing

## Project Structure


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



## Installation & Setup

### 1. Clone the Repository

```bash
git clone https://github.com/yourusername/ride-sharing-backend.git
cd ride-sharing-backend


### 2. Set Up Environment Variables

Create a .env file and add API keys & configs:

```bash
PORT=8080
API_KEY=your_api_key_here
REDIS_URL=redis:6379
JWT_SECRET=your_jwt_secret
DATABASE_URL=postgres://rideuser:ride123@db:5432/rides
```

### 3. Run with Docker

```bash
docker-compose up --build
```

### 4. Run Locally (Alternative)

```bash
go run src/main.go
```

## API Endpoints

### Authentication

#### Login (POST /auth/login)

Request:
```bash
curl -X POST "http://localhost:8080/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"testdriver","user_id":123,"role":"driver"}'
```

Response:
```json
{
  "token": "eyJhbGciOi...",
  "expires_in": "12h"
}
```

### Ride Management

#### Request Ride (POST /request-ride)

Request:
```bash
curl -X POST "http://localhost:8080/request-ride" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{"lat":0.3135,"lng":32.5811}'
```

Response:
```json
{
  "success": true,
  "data": {
    "ride_id": "a1b2c3d4",
    "driver_id": "driver1",
    "eta": 8,
    "price": 12.50
  }
}
```

#### List Available Drivers (GET /drivers)

Request:
```bash
curl "http://localhost:8080/drivers?lat=0.3135&lng=32.5811&radius=5" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

Response:
```json
{
  "drivers": [
    {
      "id": "driver1",
      "lat": 0.3135,
      "lng": 32.5810,
      "distance": 0.12,
      "vehicle": "sedan"
    }
  ]
}
```

#### Check Ride Status (GET /ride-status/:id)

Request:
```bash
curl "http://localhost:8080/ride-status/a1b2c3d4" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

Response:
```json
{
  "ride_id": "a1b2c3d4",
  "status": "in_progress",
  "driver_location": {
    "lat": 0.3135,
    "lng": 32.5810
  },
  "eta": 5
}
```

## Deployment

### Docker Deployment

1. Build the container:
```bash
docker build -t ride-sharing-backend .
```

2. Tag and push to container registry:
```bash
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
wrk -t4 -c100 -d60s http://localhost:8080/drivers -H "Authorization: Bearer YOUR_TOKEN"
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

## Future Improvements

1. Implement real-time ride tracking using WebSockets
2. Add multi-region failover for high availability
3. Implement dynamic pricing based on demand
4. Add driver rating system
5. Enhance geospatial queries with additional filters

## Troubleshooting

### Common Issues

1. Redis Connection Problems:
```bash
docker-compose logs redis
```

2. Database Migration Issues:
```bash
docker-compose exec db psql -U rideuser rides -c "SELECT * FROM pg_migrations"
```

3. Authentication Errors:
```bash
curl -v POST "http://localhost:8080/auth/login" -d '{"username":"testuser"}'
```

### Checking Service Health

```bash
curl http://localhost:8080/health
```
