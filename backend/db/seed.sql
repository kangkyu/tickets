-- Seed initial data for the database

-- Add admin user. test user
INSERT INTO users (email, name, created_at, updated_at) 
VALUES
    ('admin@example.com', 'Admin User', NOW(), NOW()),
    ('test@example.com', 'Test User', NOW(), NOW())
ON CONFLICT (email) DO NOTHING;
