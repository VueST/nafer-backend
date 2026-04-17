CREATE TYPE media_status AS ENUM ('pending', 'uploaded', 'failed');

CREATE TABLE IF NOT EXISTS media (
    id           TEXT PRIMARY KEY,
    owner_id     TEXT NOT NULL,
    filename     TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes   BIGINT NOT NULL DEFAULT 0,
    storage_key  TEXT NOT NULL,
    status       media_status NOT NULL DEFAULT 'pending',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_media_owner_id  ON media(owner_id);
CREATE INDEX IF NOT EXISTS idx_media_status    ON media(status);
