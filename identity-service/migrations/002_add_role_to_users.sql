-- Migration: 002_add_role_to_users
-- Adds a strictly-constrained role column to the users table.
-- Default is 'user' — every new registration starts at the base tier.

ALTER TABLE users
    ADD COLUMN role TEXT NOT NULL DEFAULT 'user';

-- Constraint prevents any invalid tier from being stored.
-- Adding a new role in the future is a single-line schema change.
ALTER TABLE users
    ADD CONSTRAINT users_role_check
    CHECK (role IN ('user', 'premium', 'mod', 'admin'));

COMMENT ON COLUMN users.role IS
    'User access tier. Allowed values: user | premium | mod | admin.';
