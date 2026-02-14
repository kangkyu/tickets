-- Seed initial data for the database

-- Add admin user. test user
-- Password for both: password123
INSERT INTO users (email, name, password_hash, created_at, updated_at)
VALUES
    ('admin@example.com', 'Admin User', '$2a$10$OYVHPAPKof1NgjWqPV9ZueWcD4lKog6csSHxtkZWscRbuxEtAg3.y', NOW(), NOW()),
    ('test@example.com', 'Test User', '$2a$10$OYVHPAPKof1NgjWqPV9ZueWcD4lKog6csSHxtkZWscRbuxEtAg3.y', NOW(), NOW())
ON CONFLICT (email) DO NOTHING;
