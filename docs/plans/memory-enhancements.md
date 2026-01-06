# CDD Memory System Enhancement Plan

This document consolidates the best enhancements from three proposals (GLM, MiniMax, Claude) into a production-ready implementation plan for CDD's SQLite memory system.

---

## Executive Summary

| Enhancement | Priority | Source | Complexity |
|-------------|----------|--------|------------|
| SQLite + sqlc + goose toolchain | Critical | MiniMax/Claude | Medium |
| CDD-Specific Integration Points | Critical | Claude | Medium |
| Migration Strategy with Rollback | Critical | GLM | Medium |
| Error Handling & Retry Logic | High | GLM | Medium |
| Testing Strategy | High | GLM | Medium |
| Memory Block Prioritization | High | GLM | High |
| Pub/Sub Event Integration | Medium | Claude | Low |
| Feature Flags for Safe Rollout | Medium | GLM | Low |
| Fantasy Library Integration | Medium | Claude | Medium |
| Data Export/Import | Low | GLM | Medium |

---

## Part 1: Toolchain & Schema (Foundation)

### 1.1 Toolchain Selection

**Decision:** Use `sqlc` + `goose` (proven in Crush)

```yaml
# sqlc.yaml
version: "2"
sql:
  - engine: "sqlite"
    schema: "internal/database/migrations"
    queries: "internal/database/sql"
    gen:
      go:
        package: "db"
        out: "internal/database/db"
        emit_prepared_queries: true
        emit_interface: true
```

**Dependencies:**
```go
// go.mod additions
github.com/mattn/go-sqlite3 v1.14.22
github.com/pressly/goose/v3 v3.18.0
```

### 1.2 Database Location

Follow XDG Base Directory spec:

```
~/.local/share/cdd/
  cdd.db                    # Global database

/project/.cdd/              # Optional project-specific
  project.db
```

### 1.3 Complete Schema with Indexes

```sql
-- migrations/20250101000000_initial.sql
-- +goose Up

-- Sessions
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    project_path TEXT,
    model TEXT,
    provider TEXT,
    message_count INTEGER DEFAULT 0,
    prompt_tokens INTEGER DEFAULT 0,
    completion_tokens INTEGER DEFAULT 0,
    cost REAL DEFAULT 0.0,
    created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
);

CREATE INDEX idx_sessions_project ON sessions(project_path);
CREATE INDEX idx_sessions_updated ON sessions(updated_at DESC);

CREATE TRIGGER update_sessions_timestamp
AFTER UPDATE ON sessions
BEGIN
    UPDATE sessions SET updated_at = strftime('%s', 'now') WHERE id = NEW.id;
END;

-- Messages
CREATE TABLE messages (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'system', 'tool')),
    content TEXT NOT NULL,
    reasoning TEXT,
    tool_calls TEXT,        -- JSON array
    tool_results TEXT,      -- JSON array
    model TEXT,
    provider TEXT,
    created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE INDEX idx_messages_session ON messages(session_id);
CREATE INDEX idx_messages_created ON messages(created_at DESC);

-- Memory Blocks
CREATE TABLE memory_blocks (
    id TEXT PRIMARY KEY,
    label TEXT NOT NULL,
    description TEXT,
    content TEXT NOT NULL,
    scope TEXT NOT NULL CHECK (scope IN ('global', 'project')),
    project_path TEXT,
    read_only INTEGER DEFAULT 0,
    character_limit INTEGER,
    priority INTEGER DEFAULT 5,     -- 1-10 scale for loading order
    access_count INTEGER DEFAULT 0,
    last_accessed INTEGER,
    created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    UNIQUE(label, scope, project_path)
);

CREATE INDEX idx_memory_scope ON memory_blocks(scope);
CREATE INDEX idx_memory_project ON memory_blocks(project_path);
CREATE INDEX idx_memory_priority ON memory_blocks(priority DESC);

-- File Versions
CREATE TABLE file_versions (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    path TEXT NOT NULL,
    content TEXT NOT NULL,
    version INTEGER NOT NULL,
    operation TEXT NOT NULL CHECK (operation IN ('read', 'write', 'edit')),
    created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    UNIQUE(session_id, path, version),
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE INDEX idx_files_session ON file_versions(session_id);
CREATE INDEX idx_files_path ON file_versions(path);

-- +goose Down
DROP TABLE IF EXISTS file_versions;
DROP TABLE IF EXISTS memory_blocks;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS sessions;
```

---

## Part 2: CDD-Specific Integration Points

### 2.1 SessionStore Migration

Current CDD uses in-memory `SessionStore`. Migrate to SQLite with interface compatibility:

```go
// internal/agent/session.go - Current interface
type SessionStore interface {
    Create(title string) (*Session, error)
    Get(id string) (*Session, error)
    List() ([]*Session, error)
    Current() *Session
    SetCurrent(id string) error
    AddMessage(sessionID string, msg Message) error
    // ... existing methods
}

// internal/database/repository.go - New implementation
type SQLiteSessionStore struct {
    queries *db.Queries
    cache   map[string]*Session  // Hot cache for current session
    current string
    mu      sync.RWMutex
}

func NewSQLiteSessionStore(queries *db.Queries) *SQLiteSessionStore {
    return &SQLiteSessionStore{
        queries: queries,
        cache:   make(map[string]*Session),
    }
}

// Implements SessionStore interface
func (s *SQLiteSessionStore) Create(title string) (*Session, error) {
    ctx := context.Background()

    dbSession, err := s.queries.CreateSession(ctx, db.CreateSessionParams{
        ID:    uuid.New().String(),
        Title: title,
    })
    if err != nil {
        return nil, fmt.Errorf("create session: %w", err)
    }

    session := sessionFromDB(dbSession)
    s.cache[session.ID] = session
    return session, nil
}
```

### 2.2 Pub/Sub Event Integration

CDD has existing pub/sub broker system. Database changes should emit events:

```go
// internal/database/repository.go
type SQLiteSessionStore struct {
    queries *db.Queries
    broker  *pubsub.Broker[events.SessionEvent]  // Existing CDD broker
    // ...
}

func (s *SQLiteSessionStore) Create(title string) (*Session, error) {
    // ... create session ...

    // Publish to existing pub/sub system
    s.broker.Publish(pubsub.CreatedEvent, events.SessionEvent{
        SessionID: session.ID,
        Title:     session.Title,
    })

    return session, nil
}

// internal/events/database.go - New event types
type DatabaseEvent struct {
    Type      string // "session_created", "message_saved", "memory_updated"
    SessionID string
    Timestamp time.Time
}
```

### 2.3 Fantasy Library Integration

Inject memory blocks into LLM context:

```go
// internal/agent/agent.go - Modify buildHistory()
func (a *DefaultAgent) buildContextWithMemory(session *Session) []fantasy.Message {
    msgs := []fantasy.Message{}

    // 1. Load prioritized memory blocks
    blocks, err := a.memoryRepo.GetActiveBlocks(session.ProjectPath)
    if err != nil {
        log.Warn("Failed to load memory blocks", "error", err)
    } else {
        for _, block := range blocks {
            msgs = append(msgs, fantasy.Message{
                Role: "system",
                Content: fmt.Sprintf("[Memory: %s]\n%s", block.Label, block.Content),
            })
        }
    }

    // 2. Add conversation history (existing logic)
    msgs = append(msgs, a.buildHistory(session)...)

    return msgs
}
```

### 2.4 LRU Cache Coordination

CDD has existing LRUCache for file tracking. Coordinate with SQLite:

```go
// internal/tools/tool.go - Modify file recording
func (e *DefaultExecutor) recordFileAccess(path string, op string) {
    // Existing: Update LRU cache
    e.readTimes.Put(path, time.Now())

    // New: Persist to SQLite (async)
    go func() {
        err := e.fileRepo.RecordAccess(context.Background(), FileAccess{
            SessionID: e.currentSessionID,
            Path:      path,
            Operation: op,
        })
        if err != nil {
            log.Warn("Failed to record file access", "error", err)
        }
    }()
}
```

### 2.5 Configuration Integration

Add database config to existing CDD config:

```go
// internal/config/config.go - Add database section
type Config struct {
    // ... existing fields ...
    Database DatabaseConfig `json:"database"`
}

type DatabaseConfig struct {
    Path     string `json:"path"`      // Default: ~/.local/share/cdd/cdd.db
    WALMode  bool   `json:"wal_mode"`  // Default: true
    MaxConns int    `json:"max_conns"` // Default: 1 (SQLite single-writer)
}

func (c *Config) GetDatabasePath() string {
    if c.Database.Path != "" {
        return expandPath(c.Database.Path)
    }
    return filepath.Join(xdg.DataHome, "cdd", "cdd.db")
}
```

---

## Part 3: Migration Strategy

### 3.1 Export Existing Data

Before migration, offer to export current in-memory state:

```go
// internal/database/migrate.go
type SessionExport struct {
    Sessions   []Session  `json:"sessions"`
    Messages   []Message  `json:"messages"`
    ExportedAt time.Time  `json:"exported_at"`
    Version    string     `json:"version"`
}

func ExportCurrentState(store *SessionStore) (*SessionExport, error) {
    sessions := store.ListAll()

    export := &SessionExport{
        Sessions:   sessions,
        ExportedAt: time.Now(),
        Version:    version.Version,
    }

    for _, s := range sessions {
        export.Messages = append(export.Messages, s.Messages...)
    }

    return export, nil
}

// CLI command: cdd export --output backup.json
```

### 3.2 Migration Validation

Validate before applying migration:

```go
func ValidateMigration(dbPath string) error {
    // Check if database already exists
    if _, err := os.Stat(dbPath); err == nil {
        // Database exists - check schema version
        db, err := sql.Open("sqlite3", dbPath)
        if err != nil {
            return fmt.Errorf("open database: %w", err)
        }
        defer db.Close()

        version, err := goose.GetDBVersion(db)
        if err != nil {
            return fmt.Errorf("get schema version: %w", err)
        }

        if version > CurrentSchemaVersion {
            return fmt.Errorf("database schema version %d is newer than supported %d",
                version, CurrentSchemaVersion)
        }
    }

    return nil
}
```

### 3.3 Rollback Procedure

If migration fails, restore from backup:

```go
func RollbackMigration(dbPath, backupPath string) error {
    // 1. Close any open connections
    // 2. Remove failed database
    if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
        return fmt.Errorf("remove failed db: %w", err)
    }

    // 3. Restore from backup if exists
    if backupPath != "" {
        if err := copyFile(backupPath, dbPath); err != nil {
            return fmt.Errorf("restore backup: %w", err)
        }
    }

    return nil
}

// Auto-backup before migration
func MigrateWithBackup(dbPath string) error {
    backupPath := dbPath + ".backup." + time.Now().Format("20060102150405")

    if _, err := os.Stat(dbPath); err == nil {
        if err := copyFile(dbPath, backupPath); err != nil {
            return fmt.Errorf("create backup: %w", err)
        }
    }

    if err := runMigrations(dbPath); err != nil {
        RollbackMigration(dbPath, backupPath)
        return fmt.Errorf("migration failed, rolled back: %w", err)
    }

    return nil
}
```

---

## Part 4: Error Handling & Recovery

### 4.1 Database Connection with Retry

```go
// internal/database/connect.go
type DatabaseManager struct {
    db         *sql.DB
    queries    *db.Queries
    maxRetries int
    retryDelay time.Duration
}

func NewDatabaseManager(dbPath string) (*DatabaseManager, error) {
    // Ensure directory exists
    if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
        return nil, fmt.Errorf("create db directory: %w", err)
    }

    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, fmt.Errorf("open database: %w", err)
    }

    // Apply performance pragmas
    pragmas := []string{
        "PRAGMA foreign_keys = ON",
        "PRAGMA journal_mode = WAL",
        "PRAGMA synchronous = NORMAL",
        "PRAGMA cache_size = -8000",  // 8MB cache
        "PRAGMA busy_timeout = 5000", // 5s timeout for locks
    }

    for _, pragma := range pragmas {
        if _, err := db.Exec(pragma); err != nil {
            db.Close()
            return nil, fmt.Errorf("apply pragma %q: %w", pragma, err)
        }
    }

    return &DatabaseManager{
        db:         db,
        queries:    db.New(db),
        maxRetries: 3,
        retryDelay: 100 * time.Millisecond,
    }, nil
}

func (dm *DatabaseManager) ExecWithRetry(ctx context.Context, fn func() error) error {
    var err error
    for i := 0; i < dm.maxRetries; i++ {
        err = fn()
        if err == nil {
            return nil
        }

        if !isRetryableError(err) {
            return err
        }

        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(dm.retryDelay * time.Duration(i+1)):
            // Exponential backoff
        }
    }
    return fmt.Errorf("after %d retries: %w", dm.maxRetries, err)
}

func isRetryableError(err error) bool {
    errStr := err.Error()
    return strings.Contains(errStr, "database is locked") ||
           strings.Contains(errStr, "database is busy") ||
           strings.Contains(errStr, "SQLITE_BUSY")
}
```

### 4.2 Integrity Check & Recovery

```go
func (dm *DatabaseManager) CheckIntegrity() error {
    var result string
    err := dm.db.QueryRow("PRAGMA integrity_check").Scan(&result)
    if err != nil {
        return fmt.Errorf("integrity check: %w", err)
    }

    if result != "ok" {
        log.Warn("Database integrity check failed", "result", result)
        return fmt.Errorf("integrity check failed: %s", result)
    }

    return nil
}

func (dm *DatabaseManager) Vacuum() error {
    _, err := dm.db.Exec("VACUUM")
    return err
}
```

### 4.3 Graceful Degradation

Fall back to in-memory if SQLite unavailable:

```go
// internal/database/store.go
func NewSessionStore(dbPath string) (SessionStore, error) {
    dm, err := NewDatabaseManager(dbPath)
    if err != nil {
        log.Warn("SQLite unavailable, using in-memory storage", "error", err)
        return NewInMemorySessionStore(), nil
    }

    // Run migrations
    if err := dm.Migrate(); err != nil {
        log.Warn("Migration failed, using in-memory storage", "error", err)
        dm.Close()
        return NewInMemorySessionStore(), nil
    }

    return NewSQLiteSessionStore(dm), nil
}
```

---

## Part 5: Memory Block Prioritization

### 5.1 Priority-Based Loading

Load memory blocks intelligently when context is limited:

```go
// internal/database/memory.go
type MemoryBlock struct {
    ID           string
    Label        string
    Content      string
    Priority     int       // 1-10 scale
    AccessCount  int
    LastAccessed time.Time
    Size         int       // character count
}

func (r *MemoryRepository) GetPrioritizedBlocks(projectPath string, maxTokens int) ([]MemoryBlock, error) {
    blocks, err := r.queries.ListMemoryBlocks(context.Background(), projectPath)
    if err != nil {
        return nil, err
    }

    // Score and sort blocks
    scored := make([]scoredBlock, len(blocks))
    for i, b := range blocks {
        scored[i] = scoredBlock{
            block: memoryBlockFromDB(b),
            score: calculateBlockScore(b),
        }
    }

    sort.Slice(scored, func(i, j int) bool {
        return scored[i].score > scored[j].score
    })

    // Select blocks within token budget
    var selected []MemoryBlock
    currentTokens := 0

    for _, sb := range scored {
        blockTokens := len(sb.block.Content) / 4  // Rough token estimate
        if currentTokens + blockTokens > maxTokens {
            continue
        }
        selected = append(selected, sb.block)
        currentTokens += blockTokens
    }

    return selected, nil
}

func calculateBlockScore(block db.MemoryBlock) float64 {
    // Factors: explicit priority, recency, access frequency

    // Priority weight (1-10 → 0.1-1.0)
    priorityScore := float64(block.Priority) / 10.0

    // Recency score (exponential decay over 7 days)
    hoursSinceAccess := time.Since(time.Unix(block.LastAccessed, 0)).Hours()
    recencyScore := math.Exp(-hoursSinceAccess / (24.0 * 7.0))

    // Access frequency score (capped at 1.0)
    accessScore := math.Min(float64(block.AccessCount)/10.0, 1.0)

    // Critical blocks always loaded
    if block.Label == "persona" || block.Label == "project" {
        return 100.0 + priorityScore
    }

    return priorityScore*0.5 + recencyScore*0.3 + accessScore*0.2
}
```

### 5.2 Default Memory Blocks

```go
var DefaultBlocks = []MemoryBlock{
    {
        Label:       "persona",
        Description: "Agent behavioral preferences and adaptations",
        Priority:    10,
        Scope:       "global",
    },
    {
        Label:       "user",
        Description: "Information about the user and communication style",
        Priority:    9,
        Scope:       "global",
    },
    {
        Label:       "project",
        Description: "Project-specific conventions, commands, and architecture",
        Priority:    10,
        Scope:       "project",
    },
    {
        Label:       "preferences",
        Description: "Tool preferences, ignored paths, and workflow settings",
        Priority:    7,
        Scope:       "project",
    },
}
```

---

## Part 6: Feature Flags for Safe Rollout

### 6.1 Feature Flag System

```go
// internal/config/features.go
type FeatureFlags struct {
    SQLitePersistence bool `json:"sqlite_persistence"`
    MemoryBlocks      bool `json:"memory_blocks"`
    FileVersioning    bool `json:"file_versioning"`
    SessionSwitching  bool `json:"session_switching"`
}

func LoadFeatureFlags() FeatureFlags {
    flags := FeatureFlags{
        SQLitePersistence: getEnvBool("CDD_FEATURE_SQLITE", true),
        MemoryBlocks:      getEnvBool("CDD_FEATURE_MEMORY", false),
        FileVersioning:    getEnvBool("CDD_FEATURE_VERSIONING", false),
        SessionSwitching:  getEnvBool("CDD_FEATURE_SESSIONS", false),
    }

    return flags
}

func getEnvBool(key string, defaultVal bool) bool {
    val := os.Getenv(key)
    if val == "" {
        return defaultVal
    }
    return val == "true" || val == "1"
}
```

### 6.2 Gradual Rollout Plan

```
Phase 1 (Week 1-2): SQLitePersistence = true (default on)
  - Sessions and messages persisted
  - Fallback to in-memory if fails

Phase 2 (Week 3-4): FileVersioning = true
  - File changes tracked
  - /undo command available

Phase 3 (Week 5-6): MemoryBlocks = true
  - Memory system active
  - /remember, /memory commands

Phase 4 (Week 7-8): SessionSwitching = true
  - /sessions, /switch commands
  - Full feature set
```

---

## Part 7: Testing Strategy

### 7.1 Test Scenarios Matrix

```go
// internal/database/db_test.go
var testScenarios = []struct {
    name     string
    test     func(*testing.T, *DatabaseManager)
    category string
}{
    // Session CRUD
    {"CreateSession", testCreateSession, "session"},
    {"GetSession", testGetSession, "session"},
    {"ListSessions", testListSessions, "session"},
    {"DeleteSession", testDeleteSession, "session"},
    {"DeleteSessionCascade", testDeleteSessionCascade, "session"},

    // Message CRUD
    {"AddMessage", testAddMessage, "message"},
    {"GetSessionMessages", testGetSessionMessages, "message"},
    {"MessageOrdering", testMessageOrdering, "message"},

    // Memory Blocks
    {"CreateMemoryBlock", testCreateMemoryBlock, "memory"},
    {"UpdateMemoryBlock", testUpdateMemoryBlock, "memory"},
    {"MemoryBlockUnique", testMemoryBlockUnique, "memory"},
    {"MemoryPrioritization", testMemoryPrioritization, "memory"},

    // File Versions
    {"CreateFileVersion", testCreateFileVersion, "file"},
    {"FileVersionHistory", testFileVersionHistory, "file"},
    {"FileVersionUnique", testFileVersionUnique, "file"},

    // Error Handling
    {"RetryOnLocked", testRetryOnLocked, "error"},
    {"IntegrityCheck", testIntegrityCheck, "error"},
    {"GracefulDegradation", testGracefulDegradation, "error"},

    // Migration
    {"MigrationUp", testMigrationUp, "migration"},
    {"MigrationDown", testMigrationDown, "migration"},
    {"MigrationRollback", testMigrationRollback, "migration"},
}

func TestDatabase(t *testing.T) {
    for _, tc := range testScenarios {
        t.Run(tc.name, func(t *testing.T) {
            dm := setupTestDB(t)
            defer cleanupTestDB(t, dm)
            tc.test(t, dm)
        })
    }
}
```

### 7.2 Example Tests

```go
func testCreateSession(t *testing.T, dm *DatabaseManager) {
    repo := NewSQLiteSessionStore(dm)

    session, err := repo.Create("Test Session")
    require.NoError(t, err)
    require.NotEmpty(t, session.ID)
    require.Equal(t, "Test Session", session.Title)
}

func testDeleteSessionCascade(t *testing.T, dm *DatabaseManager) {
    repo := NewSQLiteSessionStore(dm)

    // Create session with messages
    session, _ := repo.Create("Test")
    repo.AddMessage(session.ID, Message{Role: "user", Content: "Hello"})
    repo.AddMessage(session.ID, Message{Role: "assistant", Content: "Hi"})

    // Delete session
    err := repo.Delete(session.ID)
    require.NoError(t, err)

    // Verify messages deleted (CASCADE)
    msgs, err := repo.GetMessages(session.ID)
    require.NoError(t, err)
    require.Empty(t, msgs)
}

func testMemoryPrioritization(t *testing.T, dm *DatabaseManager) {
    repo := NewMemoryRepository(dm)

    // Create blocks with different priorities
    repo.Create(MemoryBlock{Label: "low", Priority: 2, Content: strings.Repeat("x", 1000)})
    repo.Create(MemoryBlock{Label: "high", Priority: 9, Content: strings.Repeat("x", 1000)})
    repo.Create(MemoryBlock{Label: "medium", Priority: 5, Content: strings.Repeat("x", 1000)})

    // Get prioritized (limited budget)
    blocks, err := repo.GetPrioritizedBlocks("", 600)  // Only ~2 blocks fit
    require.NoError(t, err)
    require.Len(t, blocks, 2)
    require.Equal(t, "high", blocks[0].Label)
    require.Equal(t, "medium", blocks[1].Label)
}

func testRetryOnLocked(t *testing.T, dm *DatabaseManager) {
    // Simulate locked database
    lockCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()

    // This should retry and eventually succeed or timeout gracefully
    err := dm.ExecWithRetry(lockCtx, func() error {
        return errors.New("database is locked")
    })

    require.Error(t, err)
    require.Contains(t, err.Error(), "database is locked")
}
```

### 7.3 Performance Benchmarks

```go
func BenchmarkSessionCreate(b *testing.B) {
    dm := setupBenchDB(b)
    repo := NewSQLiteSessionStore(dm)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        repo.Create(fmt.Sprintf("Session %d", i))
    }
}

func BenchmarkMessageAdd(b *testing.B) {
    dm := setupBenchDB(b)
    repo := NewSQLiteSessionStore(dm)
    session, _ := repo.Create("Bench Session")

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        repo.AddMessage(session.ID, Message{
            Role:    "user",
            Content: "Test message content",
        })
    }
}

func BenchmarkMemorySearch(b *testing.B) {
    dm := setupBenchDB(b)
    repo := NewMemoryRepository(dm)

    // Seed with 100 memory blocks
    for i := 0; i < 100; i++ {
        repo.Create(MemoryBlock{
            Label:   fmt.Sprintf("block_%d", i),
            Content: fmt.Sprintf("Content for block %d with searchable text", i),
        })
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        repo.Search("searchable")
    }
}
```

---

## Part 8: Data Export/Import

### 8.1 Export Format

```go
// internal/database/export.go
type ExportData struct {
    Version      string         `json:"version"`
    ExportedAt   time.Time      `json:"exported_at"`
    Sessions     []Session      `json:"sessions"`
    Messages     []Message      `json:"messages"`
    MemoryBlocks []MemoryBlock  `json:"memory_blocks"`
    FileVersions []FileVersion  `json:"file_versions,omitempty"`
}

func ExportAll(dm *DatabaseManager, includeFiles bool) (*ExportData, error) {
    ctx := context.Background()

    sessions, err := dm.queries.ListAllSessions(ctx)
    if err != nil {
        return nil, err
    }

    var messages []Message
    for _, s := range sessions {
        msgs, _ := dm.queries.GetSessionMessages(ctx, s.ID)
        messages = append(messages, messagesFromDB(msgs)...)
    }

    blocks, _ := dm.queries.ListAllMemoryBlocks(ctx)

    export := &ExportData{
        Version:      version.Version,
        ExportedAt:   time.Now(),
        Sessions:     sessionsFromDB(sessions),
        Messages:     messages,
        MemoryBlocks: memoryBlocksFromDB(blocks),
    }

    if includeFiles {
        files, _ := dm.queries.ListAllFileVersions(ctx)
        export.FileVersions = fileVersionsFromDB(files)
    }

    return export, nil
}

// CLI: cdd export --output backup.json
// CLI: cdd export --output backup.json --include-files
```

### 8.2 Import with Merge Strategy

```go
type ImportOptions struct {
    Strategy string // "overwrite", "merge", "skip"
    DryRun   bool
}

func Import(dm *DatabaseManager, data *ExportData, opts ImportOptions) error {
    if opts.DryRun {
        return validateImport(data)
    }

    tx, err := dm.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    for _, session := range data.Sessions {
        existing, _ := dm.queries.GetSession(context.Background(), session.ID)

        switch opts.Strategy {
        case "overwrite":
            dm.queries.DeleteSession(context.Background(), session.ID)
            dm.queries.CreateSession(context.Background(), sessionToParams(session))
        case "merge":
            if existing.ID == "" {
                dm.queries.CreateSession(context.Background(), sessionToParams(session))
            }
            // Keep existing, skip
        case "skip":
            if existing.ID != "" {
                continue
            }
            dm.queries.CreateSession(context.Background(), sessionToParams(session))
        }
    }

    // Similar logic for messages, memory blocks...

    return tx.Commit()
}

// CLI: cdd import backup.json --strategy merge
// CLI: cdd import backup.json --strategy overwrite --dry-run
```

---

## Part 9: Implementation Roadmap

### Phase 1: Database Foundation (Week 1-2)

**Deliverables:**
- [ ] `internal/database/` package structure
- [ ] SQLite connection with pragmas
- [ ] Goose migrations setup
- [ ] Session/Message CRUD operations
- [ ] Basic tests passing

**Files to create:**
```
internal/database/
  connect.go        # Connection management
  migrate.go        # Migration runner
  repository.go     # SessionStore implementation
  migrations/
    20250101000000_initial.sql
  sql/
    sessions.sql    # sqlc queries
    messages.sql
  db/
    db.go           # Generated by sqlc
    models.go
    sessions.sql.go
    messages.sql.go
```

### Phase 2: Migration & Safety (Week 3)

**Deliverables:**
- [ ] Export current state command
- [ ] Migration validation
- [ ] Rollback procedure
- [ ] Graceful degradation
- [ ] Feature flag: `CDD_FEATURE_SQLITE=true`

### Phase 3: File Versioning (Week 4)

**Deliverables:**
- [ ] File versions table
- [ ] Integration with Edit/Write tools
- [ ] `/undo` command
- [ ] `/diff` command
- [ ] Feature flag: `CDD_FEATURE_VERSIONING=true`

### Phase 4: Memory System (Week 5-6)

**Deliverables:**
- [ ] Memory blocks table
- [ ] Priority-based loading
- [ ] `/memory` command
- [ ] `/remember` command
- [ ] Fantasy integration
- [ ] Feature flag: `CDD_FEATURE_MEMORY=true`

### Phase 5: Session Management (Week 7)

**Deliverables:**
- [ ] `/sessions` command
- [ ] `/switch` command
- [ ] `/new` command
- [ ] TUI session picker
- [ ] Feature flag: `CDD_FEATURE_SESSIONS=true`

### Phase 6: Polish & Export (Week 8)

**Deliverables:**
- [ ] `cdd export` command
- [ ] `cdd import` command
- [ ] Performance benchmarks passing
- [ ] Documentation complete
- [ ] All feature flags default to `true`

---

## Part 10: Success Criteria

### Phase 1 Success
- [ ] SQLite database creates successfully on startup
- [ ] Sessions persist across restarts
- [ ] Messages persist with sessions
- [ ] Migrations apply cleanly
- [ ] Tests pass with >80% coverage

### Phase 2 Success
- [ ] Export command produces valid JSON
- [ ] Migration validates before applying
- [ ] Rollback restores previous state
- [ ] Graceful fallback to in-memory works

### Phase 3 Success
- [ ] File versions tracked on edit/write
- [ ] `/undo` reverts last change
- [ ] `/diff` shows file changes
- [ ] No performance regression

### Phase 4 Success
- [ ] Memory blocks persist across sessions
- [ ] Priority loading respects token limits
- [ ] `/remember` updates blocks
- [ ] Memory injected into LLM context

### Phase 5 Success
- [ ] Multiple sessions can exist
- [ ] Session switching preserves state
- [ ] TUI shows session list
- [ ] Current session indicator works

### Overall Success
- [ ] All features work together
- [ ] Query performance <100ms
- [ ] Database size reasonable (<100MB typical)
- [ ] No data loss during normal operation
- [ ] Clean upgrade path from in-memory
