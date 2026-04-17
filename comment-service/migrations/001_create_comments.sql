-- ============================================================
-- 001_create_comments.sql
-- Comment service initial schema
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS comments (
    id          TEXT        PRIMARY KEY,
    media_id    TEXT        NOT NULL,
    user_id     TEXT        NOT NULL,
    parent_id   TEXT        REFERENCES comments(id) ON DELETE CASCADE,
    body        TEXT        NOT NULL CHECK (char_length(body) BETWEEN 1 AND 2000),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for the most common query: get all comments for a media item
CREATE INDEX IF NOT EXISTS idx_comments_media_id ON comments(media_id);

-- Index for fetching user's own comments (moderation, profile page)
CREATE INDEX IF NOT EXISTS idx_comments_user_id  ON comments(user_id);

-- Index for thread replies
CREATE INDEX IF NOT EXISTS idx_comments_parent_id ON comments(parent_id) WHERE parent_id IS NOT NULL;
