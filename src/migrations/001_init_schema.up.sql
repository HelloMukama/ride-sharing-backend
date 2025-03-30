CREATE EXTENSION IF NOT EXISTS postgis;

-- Rides table (stores trip information)
CREATE TABLE rides (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id VARCHAR(255) NOT NULL,  -- References drivers.driver_id
    rider_id INTEGER NOT NULL,        -- References your users/app riders
    status VARCHAR(50) NOT NULL CHECK (status IN ('requested', 'accepted', 'in_progress', 'completed', 'cancelled')),
    start_location GEOMETRY(POINT, 4326) NOT NULL,  -- WGS84 coordinates
    end_location GEOMETRY(POINT, 4326),             -- WGS84 coordinates
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Drivers table (stores driver information and availability)
CREATE TABLE drivers (
    driver_id VARCHAR(255) PRIMARY KEY,
    available BOOLEAN NOT NULL DEFAULT true,
    current_location GEOMETRY(POINT, 4326) NOT NULL,  -- WGS84 coordinates
    last_updated TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX idx_rides_status ON rides(status);
CREATE INDEX idx_rides_rider_id ON rides(rider_id);
CREATE INDEX idx_rides_driver_id ON rides(driver_id);
CREATE INDEX idx_drivers_location ON drivers USING GIST(current_location);
CREATE INDEX idx_drivers_available ON drivers(available) WHERE available = true;

-- Insert 15 drivers across Kampala (coordinates approximate)
INSERT INTO drivers (driver_id, available, current_location) VALUES
-- Available drivers
('driver1', true, ST_SetSRID(ST_MakePoint(32.5820, 0.3167), 4326)),  -- Nakasero
('driver2', true, ST_SetSRID(ST_MakePoint(32.5805, 0.3135), 4326)),  -- Kololo
('driver3', true, ST_SetSRID(ST_MakePoint(32.5850, 0.3180), 4326)),  -- Bukoto
('driver4', true, ST_SetSRID(ST_MakePoint(32.5750, 0.3100), 4326)),  -- Old Kampala
('driver5', true, ST_SetSRID(ST_MakePoint(32.5900, 0.3200), 4326)),  -- Ntinda
('driver6', true, ST_SetSRID(ST_MakePoint(32.5650, 0.3050), 4326)),  -- Namirembe
('driver7', true, ST_SetSRID(ST_MakePoint(32.5950, 0.3220), 4326)),  -- Kiwatule
-- Unavailable drivers
('driver8', false, ST_SetSRID(ST_MakePoint(32.5700, 0.3080), 4326)), -- Mengo
('driver9', false, ST_SetSRID(ST_MakePoint(32.5780, 0.3150), 4326)), -- Kamwokya
('driver10', false, ST_SetSRID(ST_MakePoint(32.5880, 0.3190), 4326)),-- Kisaasi
-- Additional drivers
('driver11', true, ST_SetSRID(ST_MakePoint(32.5720, 0.3090), 4326)),-- Rubaga
('driver12', true, ST_SetSRID(ST_MakePoint(32.5830, 0.3170), 4326)),-- Bugolobi
('driver13', false, ST_SetSRID(ST_MakePoint(32.5870, 0.3185), 4326)),-- Bukoto
('driver14', true, ST_SetSRID(ST_MakePoint(32.5790, 0.3140), 4326)),-- Naguru
('driver15', false, ST_SetSRID(ST_MakePoint(32.5920, 0.3210), 4326));-- Kyanja

-- Insert 10 sample rides (mix of completed and active)
INSERT INTO rides (driver_id, rider_id, status, start_location, end_location) VALUES
-- Completed rides
('driver8', 1001, 'completed', 
 ST_SetSRID(ST_MakePoint(32.5820, 0.3167), 4326),  -- Nakasero pickup
 ST_SetSRID(ST_MakePoint(32.5900, 0.3200), 4326)), -- Ntinda dropoff
('driver9', 1002, 'completed',
 ST_SetSRID(ST_MakePoint(32.5805, 0.3135), 4326),  -- Kololo pickup
 ST_SetSRID(ST_MakePoint(32.5750, 0.3100), 4326)), -- Old Kampala dropoff
-- Active rides
('driver10', 1003, 'in_progress',
 ST_SetSRID(ST_MakePoint(32.5850, 0.3180), 4326),  -- Bukoto pickup
 NULL),
('driver3', 1004, 'accepted',
 ST_SetSRID(ST_MakePoint(32.5750, 0.3100), 4326),  -- Old Kampala pickup
 NULL),
-- More sample rides
('driver5', 1005, 'completed',
 ST_SetSRID(ST_MakePoint(32.5900, 0.3200), 4326),
 ST_SetSRID(ST_MakePoint(32.5950, 0.3220), 4326)),
('driver7', 1006, 'cancelled',
 ST_SetSRID(ST_MakePoint(32.5650, 0.3050), 4326),
 NULL),
('driver2', 1007, 'completed',
 ST_SetSRID(ST_MakePoint(32.5950, 0.3220), 4326),
 ST_SetSRID(ST_MakePoint(32.5820, 0.3167), 4326)),
('driver4', 1008, 'requested',
 ST_SetSRID(ST_MakePoint(32.5700, 0.3080), 4326),
 NULL),
('driver1', 1009, 'completed',
 ST_SetSRID(ST_MakePoint(32.5880, 0.3190), 4326),
 ST_SetSRID(ST_MakePoint(32.5780, 0.3150), 4326)),
('driver6', 1010, 'in_progress',
 ST_SetSRID(ST_MakePoint(32.5780, 0.3150), 4326),
 NULL);

CREATE TABLE driver_notifications (
    id SERIAL PRIMARY KEY,
    driver_id VARCHAR(255) NOT NULL REFERENCES drivers(driver_id),
    ride_id UUID NOT NULL REFERENCES rides(id),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_driver ON driver_notifications(driver_id, status);
