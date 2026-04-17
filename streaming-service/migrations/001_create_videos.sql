-- ============================================================
-- 001_create_videos.sql
-- Streaming service initial schema
-- ============================================================

CREATE TYPE video_status AS ENUM ('pending', 'processing', 'ready', 'failed');

CREATE TABLE IF NOT EXISTS videos (
    id            TEXT         PRIMARY KEY,
    uploader_id   TEXT         NOT NULL,
    title         TEXT         NOT NULL DEFAULT '',
    description   TEXT         NOT NULL DEFAULT '',
    source_path   TEXT         NOT NULL,          -- MinIO object key (original file)
    hls_path      TEXT         NOT NULL DEFAULT '', -- MinIO key to master.m3u8
    thumbnail_url TEXT         NOT NULL DEFAULT '',
    duration_sec  INTEGER      NOT NULL DEFAULT 0,
    status        video_status NOT NULL DEFAULT 'pending',
    error_msg     TEXT         NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Filter by uploader (profile page, my videos)
CREATE INDEX IF NOT EXISTS idx_videos_uploader_id ON videos(uploader_id);

-- Filter by status (find all pending/failed for retry)
CREATE INDEX IF NOT EXISTS idx_videos_status      ON videos(status);

-- Time-based listing
CREATE INDEX IF NOT EXISTS idx_videos_created_at  ON videos(created_at DESC);
