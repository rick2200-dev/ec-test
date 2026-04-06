-- Create service-scoped schemas
CREATE SCHEMA IF NOT EXISTS auth_svc;
CREATE SCHEMA IF NOT EXISTS catalog_svc;
CREATE SCHEMA IF NOT EXISTS inventory_svc;
CREATE SCHEMA IF NOT EXISTS order_svc;

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
