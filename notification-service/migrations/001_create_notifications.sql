-- ============================================================
-- 001_create_notifications.sql
-- Notification service initial schema
-- ============================================================

CREATE TABLE IF NOT EXISTS notifications (
    id          TEXT        PRIMARY KEY,
    user_id     TEXT        NOT NULL,
    actor_id    TEXT        NOT NULL,
    type        TEXT        NOT NULL,
    resource_id TEXT        NOT NULL DEFAULT '',
    message     TEXT        NOT NULL,
    is_read     BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Most frequent query: get all notifications for a user, newest first
CREATE INDEX IF NOT EXISTS idx_notifications_user_id    ON notifications(user_id, created_at DESC);

-- Fast unread count query
CREATE INDEX IF NOT EXISTS idx_notifications_unread     ON notifications(user_id, is_read) WHERE is_read = FALSE;
