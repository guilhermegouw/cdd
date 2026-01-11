-- +goose Up

-- Sessions table
CREATE TABLE sessions (
    id                 TEXT PRIMARY KEY,
    title              TEXT NOT NULL DEFAULT '',
    message_count      INTEGER NOT NULL DEFAULT 0,
    summary_message_id TEXT,
    created_at         INTEGER NOT NULL,
    updated_at         INTEGER NOT NULL
);

CREATE INDEX idx_sessions_updated ON sessions(updated_at DESC);

-- Messages table
CREATE TABLE messages (
    id         TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    role       TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'system', 'tool')),
    parts      TEXT NOT NULL DEFAULT '[]',
    model      TEXT,
    provider   TEXT,
    is_summary INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE INDEX idx_messages_session ON messages(session_id);
CREATE INDEX idx_messages_created ON messages(session_id, created_at);

-- +goose Down
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS sessions;
