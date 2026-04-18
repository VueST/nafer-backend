-- Migration: 002_add_role_to_users
-- Adds a strictly-constrained role column to the users table.
-- Default is 'user' — every new registration starts at the base tier.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'user';

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'users_role_check') THEN
        ALTER TABLE users
            ADD CONSTRAINT users_role_check
            CHECK (role IN ('user', 'premium', 'mod', 'admin'));
    END IF;
END $$;

COMMENT ON COLUMN users.role IS
    'User access tier. Allowed values: user | premium | mod | admin.';
