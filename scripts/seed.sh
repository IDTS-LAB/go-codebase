#!/bin/bash
set -e

echo "Seeding database..."

psql "${DATABASE_URL:-postgres://postgres:postgres@localhost:5432/go_codebase?sslmode=disable}" <<SQL
-- Default roles
INSERT INTO roles (name, description) VALUES
    ('admin', 'System administrator with full access'),
    ('user', 'Regular user with basic access'),
    ('moderator', 'Content moderator')
ON CONFLICT (name) DO NOTHING;

-- Default permissions
INSERT INTO permissions (name, description, resource, action) VALUES
    ('users.read', 'Read users', 'users', 'read'),
    ('users.write', 'Write users', 'users', 'write'),
    ('users.delete', 'Delete users', 'users', 'delete'),
    ('roles.read', 'Read roles', 'roles', 'read'),
    ('roles.write', 'Write roles', 'roles', 'write'),
    ('permissions.read', 'Read permissions', 'permissions', 'read'),
    ('permissions.write', 'Write permissions', 'permissions', 'write'),
    ('todos.read', 'Read todos', 'todos', 'read'),
    ('todos.write', 'Write todos', 'todos', 'write'),
    ('todos.delete', 'Delete todos', 'todos', 'delete')
ON CONFLICT (name) DO NOTHING;

-- Assign all permissions to admin role
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;

-- Assign basic permissions to user role
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'user' AND p.name IN ('todos.read', 'todos.write')
ON CONFLICT DO NOTHING;
SQL

echo "Seed complete!"
