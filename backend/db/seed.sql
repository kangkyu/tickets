-- Seed initial data for the database

-- Add admin user
INSERT INTO users (email, name, created_at, updated_at) 
VALUES ('admin@example.com', 'Admin User', NOW(), NOW())
ON CONFLICT (email) DO NOTHING;

-- Add test user
INSERT INTO users (email, name, created_at, updated_at) 
VALUES ('test@example.com', 'Test User', NOW(), NOW())
ON CONFLICT (email) DO NOTHING;

