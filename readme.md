# Ride-Sharing Service Backend

A high-performance ride-sharing backend demonstrating scalable architecture with real-time capabilities.

## Features Implemented

**JWT Authentication**  
- Token generation/login endpoint
- Token validation middleware  
- Automatic token invalidation on logout  

**Driver-Rider Matching**  
- Geospatial queries for nearest drivers  
- Redis caching of driver locations  
- Real-time WebSocket notifications  

**Observability**  
- Prometheus metrics endpoint  
- Grafana dashboard integration  

**Deployment**  
- Docker containerization  
- PostgreSQL with PostGIS  
- Redis for caching  

## Installation & Setup

```bash
# 1. Clone repository
git clone https://github.com/yourusername/ride-sharing-backend.git
cd ride-sharing-backend

# 2. Setup environment
cp .env.example .env
# Edit .env with your configuration

# 3. Start services
docker-compose up --build

test out with the endpoints

