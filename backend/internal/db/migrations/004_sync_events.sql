-- Sync events dedup table
-- Stores every client event for idempotent dedup via (user_id, client_event_id) unique key.
CREATE TABLE IF NOT EXISTS sync_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    client_event_id TEXT NOT NULL,
    event_type TEXT NOT NULL CHECK (event_type IN ('review_submitted','output_submitted','task_completed','profile_updated')),
    occurred_at TEXT NOT NULL,
    payload TEXT NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'accepted' CHECK (status IN ('accepted','rejected')),
    reason TEXT,
    server_seq INTEGER,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE (user_id, client_event_id)
);

CREATE INDEX IF NOT EXISTS idx_sync_events_user_seq ON sync_events (user_id, server_seq);
CREATE INDEX IF NOT EXISTS idx_sync_events_created ON sync_events (created_at);

-- Server-side change log for pull sync.
-- Tracks mutations to entities that clients need to know about.
CREATE TABLE IF NOT EXISTS sync_change_log (
    seq INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    op TEXT NOT NULL CHECK (op IN ('upsert','delete')),
    payload TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_sync_change_log_user_seq ON sync_change_log (user_id, seq);
