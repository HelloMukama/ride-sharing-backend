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

