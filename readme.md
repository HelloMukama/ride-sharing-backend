# Ride Sharing Backend Service

A high-performance ride-sharing backend designed for scalability and reliability.

## Features

- JWT Authentication
- Redis-based geospatial driver matching
- Dockerized deployment
- CI/CD pipeline

## Running Locally

1. Clone the repository
2. Set up environment variables in `.env`
3. Run `docker-compose up --build`
4. Access the API at `http://localhost:8080`

## API Endpoints

- POST `/auth/login` - Get JWT token
- POST `/request-ride` - Request a ride
- GET `/drivers` - List available drivers



------------------------------


## Running the Application
1. Clone the repository
2. Run: `docker-compose up`
3. Access:
   - App: http://localhost:8080
   - Grafana: http://localhost:3000
   - Prometheus: http://localhost:9090

No additional setup required - all dependencies are containerized.



## Accessing Services
- Application: http://localhost:8080
- PostgreSQL: localhost:15432 (user: rideuser, pass: ride123)
- Redis: localhost:16379
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000 (admin/admin)


what's working

jwt login token issued
invalidation of all existing tokens once a new token is created
