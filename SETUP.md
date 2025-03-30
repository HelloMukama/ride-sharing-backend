## Quick Start
1. `docker-compose up --build`
2. Access endpoints:
   - API: http://localhost:8080
   - Metrics: http://localhost:8080/metrics
   - Prometheus: http://localhost:9090
   - Grafana: http://localhost:3000
   


# Connect to PostgreSQL
docker exec -it ride-sharing-backend_db_1 psql -U rideuser -d rides

-- Inside psql, run:
INSERT INTO rides (driver_id, rider_id, status, start_location) VALUES 
('driver1', 123, 'available', ST_GeomFromText('POINT(0.3135 32.5811)', 4326)),
('driver2', 456, 'available', ST_GeomFromText('POINT(0.3167 32.5825)', 4326));
\q
