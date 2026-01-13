# Letta Memory Replication Plan - Local SQLite Implementation

## Executive Summary

This plan describes how to replicate Letta's memory block architecture using a local SQLite database instead of cloud-based storage. The implementation maintains Letta's core concepts (memory blocks, labels, limits, hierarchical organization) while providing full local control, offline capability, and structured querying.

## Core Architecture Comparison

### Letta's Current Approach
- **Storage**: Cloud-based Letta API (`@letta-ai/letta-client`)
- **Memory Blocks**: Stored on Letta servers with block labels and limits
- **Block Types**:
  - Global blocks: `persona`, `human` (shared across all projects)
  - Project blocks: `project`, `skills`, `loaded_skills` (per-project)
- **Access Pattern**: API calls for every read/write operation
- **Context Injection**: Blocks are fetched and injected into LLM prompts at runtime

### Proposed SQLite Approach
- **Storage**: Local SQLite database using `bun:sqlite`
- **Memory Blocks**: Same block structure stored in relational tables
- **Block Types**: Preserved hierarchy (global vs project)
- **Access Pattern**: Direct database queries with optional in-memory caching
- **Context Injection**: Same injection pattern, but data comes from local DB

## Database Schema Design

### Tables

```sql
-- Core memory blocks table
CREATE TABLE memory_blocks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  label TEXT NOT NULL UNIQUE,           -- Block identifier (e.g., "persona", "project")
  value TEXT NOT NULL,                  -- Block content (can be large)
  description TEXT,                      -- Optional description
  limit INTEGER,                         -- Character limit (NULL = unlimited)
  read_only INTEGER DEFAULT 0,            -- Boolean: can agent modify?
  scope TEXT NOT NULL,                   -- 'global' or 'project'
  project_id INTEGER,                    -- NULL for global blocks
  created_at INTEGER NOT NULL,           -- Unix timestamp
  updated_at INTEGER NOT NULL,           -- Unix timestamp
  is_active INTEGER DEFAULT 1,            -- Soft delete flag
  FOREIGN KEY (project_id) REFERENCES projects(id)
);

-- Projects table (tracks memory by working directory)
CREATE TABLE projects (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  path TEXT NOT NULL UNIQUE,             -- Absolute path to project directory
  created_at INTEGER NOT NULL,
  last_accessed_at INTEGER,
  is_active INTEGER DEFAULT 1
);

-- Memory block version history (for rollback/audit)
CREATE TABLE block_history (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  block_id INTEGER NOT NULL,
  previous_value TEXT,
  new_value TEXT NOT NULL,
  changed_at INTEGER NOT NULL,
  change_reason TEXT,                    -- Optional: "user_edit", "agent_update", "auto_cleanup"
  FOREIGN KEY (block_id) REFERENCES memory_blocks(id)
);

-- Memory search embeddings (optional: for semantic search)
CREATE TABLE block_embeddings (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  block_id INTEGER NOT NULL,
  chunk_start INTEGER,                   -- Character position of chunk
  chunk_end INTEGER,
  embedding BLOB,                       -- Vector embedding (from text-embedding-ada-002 or similar)
  created_at INTEGER NOT NULL,
  FOREIGN KEY (block_id) REFERENCES memory_blocks(id)
);

-- Memory operations log (audit trail)
CREATE TABLE memory_operations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  operation_type TEXT NOT NULL,          -- 'create', 'read', 'update', 'delete', 'truncate'
  block_label TEXT,
  block_id INTEGER,
  project_id INTEGER,
  value_length INTEGER,
  operation_result TEXT,                  -- 'success', 'error', etc.
  error_message TEXT,
  timestamp INTEGER NOT NULL,
  FOREIGN KEY (block_id) REFERENCES memory_blocks(id),
  FOREIGN KEY (project_id) REFERENCES projects(id)
);

-- Indexes for performance
CREATE INDEX idx_memory_blocks_label ON memory_blocks(label);
CREATE INDEX idx_memory_blocks_scope ON memory_blocks(scope);
CREATE INDEX idx_memory_blocks_project ON memory_blocks(project_id);
CREATE INDEX idx_block_history_block ON block_history(block_id, changed_at);
CREATE INDEX idx_block_embeddings_block ON block_embeddings(block_id);
CREATE INDEX idx_memory_operations_timestamp ON memory_operations(timestamp);
```

## Implementation Phases

### Phase 1: Core SQLite Infrastructure

**File Structure:**
```
src/memory/
├── database.ts           # SQLite connection and schema management
├── schema.sql            # Database schema definition
├── migrations/           # Schema version migrations
│   ├── 001_initial.sql
│   ├── 002_add_history.sql
│   └── 003_add_embeddings.sql
└── types.ts             # TypeScript interfaces for memory types
```

**Key Components:**

1. **Database Manager (`database.ts`)**
```typescript
import Database from 'bun:sqlite';

interface DatabaseConfig {
  path?: string;              // Default: ~/.letta-memory/memory.db
  inMemory?: boolean;         // For testing
  enableWAL?: boolean;        // Write-Ahead Logging for performance
}

class MemoryDatabase {
  private db: Database;
  private config: DatabaseConfig;

  constructor(config: DatabaseConfig = {}) {
    const defaultPath = path.join(
      os.homedir(),
      '.letta-memory',
      'memory.db'
    );

    this.config = {
      path: config.path || defaultPath,
      enableWAL: config.enableWAL ?? true,
    };

    // Create directory if needed
    ensureDir(path.dirname(this.config.path!));

    this.db = new Database(this.config.path!);
    this.db.exec('PRAGMA journal_mode = WAL');
    this.db.exec('PRAGMA foreign_keys = ON');

    this.initializeSchema();
    this.runMigrations();
  }

  private initializeSchema(): void {
    // Load and execute schema.sql
  }

  private runMigrations(): void {
    // Track and apply migrations
  }

  // CRUD operations for memory blocks
  createBlock(block: CreateBlock): MemoryBlock { }
  getBlock(label: string, projectId?: number): MemoryBlock | null { }
  updateBlock(label: string, value: string, projectId?: number): void { }
  deleteBlock(label: string, projectId?: number): void { }

  // Project management
  getProjectByPath(path: string): Project | null { }
  createProject(path: string, name?: string): Project { }

  // Query operations
  getAllBlocks(projectId?: number): MemoryBlock[] { }
  getBlocksByScope(scope: 'global' | 'project', projectId?: number): MemoryBlock[] { }

  // History and audit
  getBlockHistory(blockId: number): BlockHistory[] { }
  getRecentOperations(limit?: number): MemoryOperation[] { }
}
```

2. **Type Definitions (`types.ts`)**
```typescript
// Match Letta's CreateBlock structure
export interface CreateBlock {
  label: string;
  value: string;
  description?: string;
  limit?: number;
  read_only?: boolean;
  scope?: 'global' | 'project';
}

export interface MemoryBlock {
  id: number;
  label: string;
  value: string;
  description?: string;
  limit?: number;
  read_only: boolean;
  scope: 'global' | 'project';
  project_id?: number;
  created_at: number;
  updated_at: number;
  is_active: boolean;
}

export interface Project {
  id: number;
  name: string;
  path: string;
  created_at: number;
  last_accessed_at?: number;
  is_active: boolean;
}

export interface BlockHistory {
  id: number;
  block_id: number;
  previous_value?: string;
  new_value: string;
  changed_at: number;
  change_reason?: string;
}
```

### Phase 2: Memory Block API (Letta-Compatible)

**File Structure:**
```
src/memory/
├── blocks.ts            # Main API matching Letta client interface
├── defaults.ts         # Default block initialization
└── prompts/
    ├── persona.mdx
    ├── human.mdx
    ├── project.mdx
    ├── skills.mdx
    └── loaded_skills.mdx
```

**Key Components:**

1. **Blocks API (`blocks.ts`)**
```typescript
// Replicate Letta's client.blocks.* interface
export class MemoryBlocks {
  private db: MemoryDatabase;
  private projectId?: number;

  constructor(projectId?: number) {
    this.db = getDatabase();
    this.projectId = projectId;
  }

  async create(block: CreateBlock): Promise<MemoryBlock> {
    // Validate label uniqueness
    // Apply character limits
    // Create block in database
    // Log operation
    // Return created block
  }

  async retrieve(label: string): Promise<MemoryBlock> {
    // Get block by label (respect project_id)
    // Update last_accessed_at
  }

  async update(label: string, params: UpdateBlockParams): Promise<MemoryBlock> {
    // Check read_only flag
    // Validate character limit
    // Store previous value in history
    // Update block
    // Log operation
  }

  async list(params?: ListBlocksParams): Promise<MemoryBlock[]> {
    // Filter by scope, project_id, etc.
    // Return sorted list
  }

  async delete(label: string): Promise<void> {
    // Soft delete (set is_active = 0)
    // Keep in history
  }

  // Letta-specific operations
  async truncateToLimit(label: string): Promise<void> {
    // Implement character limit enforcement
  }

  async merge(label: string, content: string): Promise<MemoryBlock> {
    // Smart merge for read-only blocks like skills
  }
}
```

2. **Default Blocks (`defaults.ts`)**
```typescript
import { readFile } from 'node:fs/promises';
import { join } from 'node:path';

// Same structure as Letta's MEMORY_PROMPTS
export const MEMORY_PROMPTS: Record<string, string> = {
  'persona.mdx': '',        // Load from disk
  'human.mdx': '',          // Load from disk
  'project.mdx': '',        // Load from disk
  'skills.mdx': '',         // Load from disk
  'loaded_skills.mdx': '',  // Load from disk
};

export const GLOBAL_BLOCK_LABELS = ['persona', 'human'] as const;
export const PROJECT_BLOCK_LABELS = ['project', 'skills', 'loaded_skills'] as const;

export async function initializeDefaultBlocks(db: MemoryDatabase): Promise<void> {
  // Create global blocks if they don't exist
  for (const label of GLOBAL_BLOCK_LABELS) {
    const existing = db.getBlock(label);
    if (!existing) {
      const content = await loadPromptContent(`${label}.mdx`);
      db.createBlock({
        label,
        value: content,
        scope: 'global',
      });
    }
  }
}

async function loadPromptContent(filename: string): Promise<string> {
  const filepath = join(__dirname, 'prompts', filename);
  return await readFile(filepath, 'utf-8');
}
```

### Phase 3: Integration with Agent Context

**File Structure:**
```
src/memory/
├── context.ts           # Agent context management
├── injection.ts         # Block injection into LLM prompts
└── cache.ts            # Optional in-memory caching layer
```

**Key Components:**

1. **Context Management (`context.ts`)**
```typescript
// Replicate Letta's agent/context.ts pattern
interface AgentMemoryContext {
  agentId: string;
  projectId: number;
  skillsDirectory?: string;
  cachedBlocks?: Map<string, MemoryBlock>;
}

class MemoryContext {
  private static contexts = new Map<string, AgentMemoryContext>();

  static setContext(agentId: string, projectId: number): void {
    this.contexts.set(agentId, {
      agentId,
      projectId,
    });
  }

  static getContext(agentId: string): AgentMemoryContext | undefined {
    return this.contexts.get(agentId);
  }

  static getBlocks(agentId: string): MemoryBlocks {
    const ctx = this.getContext(agentId);
    return new MemoryBlocks(ctx?.projectId);
  }
}
```

2. **Prompt Injection (`injection.ts`)**
```typescript
export class BlockInjector {
  private blocks: MemoryBlocks;

  constructor(blocks: MemoryBlocks) {
    this.blocks = blocks;
  }

  async injectIntoPrompt(
    systemPrompt: string,
    mode: 'full' | 'read-only' | 'read-write' = 'full'
  ): Promise<string> {
    const allBlocks = await this.blocks.list();

    let injectedContent = '';

    // Group by read_only status
    const readOnlyBlocks = allBlocks.filter(b => b.read_only);
    const writableBlocks = allBlocks.filter(b => !b.read_only);

    if (mode === 'full' || mode === 'read-only') {
      if (readOnlyBlocks.length > 0) {
        injectedContent += '\n--- MEMORY BLOCKS (READ-ONLY) ---\n';
        for (const block of readOnlyBlocks) {
          injectedContent += `## ${block.label}\n${block.value}\n\n`;
        }
      }
    }

    if (mode === 'full' || mode === 'read-write') {
      if (writableBlocks.length > 0) {
        injectedContent += '--- MEMORY BLOCKS (READ-WRITE) ---\n';
        for (const block of writableBlocks) {
          injectedContent += `## ${block.label}`;
          if (block.description) {
            injectedContent += ` (${block.description})`;
          }
          injectedContent += `\n${block.value}\n\n`;
        }
      }
    }

    return systemPrompt + injectedContent;
  }

  async getBlockSummary(agentId: string): Promise<BlockSummary[]> {
    // Return concise summary for UI display
  }
}
```

### Phase 4: Advanced Features

1. **Memory Search (`search.ts`)**
```typescript
export class MemorySearch {
  private db: MemoryDatabase;

  // Keyword search using SQLite FTS5
  async searchKeyword(query: string, projectId?: number): Promise<MemoryBlock[]> {
    // Use SQLite's LIKE or full-text search
    this.db.prepare(`
      SELECT * FROM memory_blocks
      WHERE value LIKE ? AND is_active = 1
    `).all(`%${query}%`);
  }

  // Semantic search using embeddings
  async searchSemantic(query: string, projectId?: number): Promise<MemoryBlock[]> {
    // Compute embedding for query
    // Compare against block_embeddings table
    // Return top K similar blocks
  }

  async searchByLabel(pattern: string, projectId?: number): Promise<MemoryBlock[]> {
    // Regex search on labels
  }
}
```

2. **Memory Compression (`compression.ts`)**
```typescript
export class MemoryCompressor {
  // Summarize old content when approaching limits
  async compressBlock(blockId: number, strategy: 'summarize' | 'truncate' | 'archive'): Promise<void> {
    // Use LLM to summarize
    // Move old content to block_history
    // Keep only summary in main block
  }

  async autoCompressBlocks(projectId: number): Promise<void> {
    // Find blocks approaching 80% of limit
    // Trigger compression
  }
}
```

3. **Memory Export/Import (`import-export.ts`)**
```typescript
export class MemoryExporter {
  async exportToJSON(projectId: number): Promise<ExportedMemory> {
    // Export all blocks, history, and metadata
  }

  async importFromJSON(data: ExportedMemory): Promise<void> {
    // Import blocks with merge strategy
  }

  async exportToLettaFormat(projectId: number): Promise<LettaAgentFile> {
    // Export in Letta's AgentFile format
  }
}
```

### Phase 5: CLI Integration

**File Structure:**
```
src/memory/
└── cli/
    ├── commands/
    │   ├── memory-list.ts
    │   ├── memory-view.ts
    │   ├── memory-edit.ts
    │   ├── memory-history.ts
    │   └── memory-export.ts
    └── ui/
        └── MemoryViewer.tsx    # Replicate Letta's CLI UI
```

**CLI Commands:**

```bash
# List all memory blocks for current project
letta-memory list

# View a specific block
letta-memory view persona

# Edit a block (opens in editor)
letta-memory edit project

# View block history
letta-memory history persona

# Search memory
letta-memory search "authentication"

# Export memory
letta-memory export memory-backup.json

# Import memory
letta-memory import memory-backup.json

# Compress memory blocks
letta-memory compress

# Reset project memory
letta-memory reset
```

## Migration Path from Letta

### Step 1: Export Letta Memory
```typescript
export async function exportLettaBlocks(agentId: string): Promise<CreateBlock[]> {
  const client = await getLettaClient();
  const blocks = await client.agents.blocks.list({ agent_id: agentId });

  return blocks.map(b => ({
    label: b.label,
    value: b.value || '',
    description: b.description,
    limit: b.limit,
    read_only: b.read_only,
  }));
}
```

### Step 2: Import to SQLite
```typescript
export async function importToSQLite(blocks: CreateBlock[], projectId: number): Promise<void> {
  const db = getDatabase();

  for (const block of blocks) {
    await db.createBlock({
      ...block,
      scope: block.label in PROJECT_BLOCK_LABELS ? 'project' : 'global',
      project_id: projectId,
    });
  }
}
```

### Step 3: Full Migration Command
```bash
letta-memory migrate-from-letta --agent-id <id>
```

## Performance Optimizations

1. **In-Memory Cache**
```typescript
class BlockCache {
  private cache = new Map<string, { block: MemoryBlock, timestamp: number }>();
  private ttl = 60000; // 1 minute

  get(label: string): MemoryBlock | null {
    const cached = this.cache.get(label);
    if (cached && Date.now() - cached.timestamp < this.ttl) {
      return cached.block;
    }
    this.cache.delete(label);
    return null;
  }

  set(label: string, block: MemoryBlock): void {
    this.cache.set(label, { block, timestamp: Date.now() });
  }
}
```

2. **Connection Pooling**
```typescript
class DatabasePool {
  private connections: Database[] = [];
  private maxConnections = 5;

  getConnection(): Database {
    // Reuse connections
  }

  releaseConnection(db: Database): void {
    // Return to pool
  }
}
```

3. **Bulk Operations**
```typescript
// Use transactions for bulk updates
db.transaction(() => {
  for (const block of blocks) {
    db.createBlock(block);
  }
})();
```

## Testing Strategy

### Unit Tests
```typescript
import { describe, it, expect } from 'bun:test';

describe('MemoryBlocks', () => {
  it('should create a new block', async () => {
    const db = new MemoryDatabase({ inMemory: true });
    const blocks = new MemoryBlocks();

    const block = await blocks.create({
      label: 'test',
      value: 'content',
    });

    expect(block.id).toBeDefined();
    expect(block.label).toBe('test');
  });

  it('should enforce character limits', async () => {
    const blocks = new MemoryBlocks();

    const block = await blocks.create({
      label: 'test',
      value: 'x'.repeat(1000),
      limit: 500,
    });

    expect(block.value.length).toBeLessThanOrEqual(500);
  });

  it('should prevent modification of read-only blocks', async () => {
    const blocks = new MemoryBlocks();

    await expect(blocks.update('persona', { value: 'modified' }))
      .rejects.toThrow('read-only');
  });
});
```

### Integration Tests
```typescript
describe('Full Memory Lifecycle', () => {
  it('should create, read, update, and delete blocks', async () => {
    const db = new MemoryDatabase({ inMemory: true });
    const blocks = new MemoryBlocks();
    const injector = new BlockInjector(blocks);

    // Create project
    const project = db.createProject('/test/path');

    // Create blocks
    await blocks.create({ label: 'test', value: 'content' });

    // Inject into prompt
    const prompt = await injector.injectIntoPrompt('System prompt');
    expect(prompt).toContain('test');

    // Update block
    await blocks.update('test', { value: 'updated' });

    // Verify history
    const history = db.getBlockHistory(blockId);
    expect(history.length).toBe(1);

    // Delete
    await blocks.delete('test');
  });
});
```

## Security Considerations

1. **File Permissions**
   - Database file: `0600` (read/write by owner only)
   - `.letta-memory` directory: `0700`

2. **SQL Injection Prevention**
   - Always use parameterized queries
   - Validate labels and block values

3. **Path Traversal Protection**
   - Sanitize project paths before insertion
   - Use `path.resolve()` for absolute paths

4. **Block Value Validation**
   - Enforce maximum size limits
   - Sanitize Markdown content

## Future Enhancements

1. **Multi-Model Support**
   - Store blocks for different agents in same DB
   - Use `agent_id` as partition key

2. **Memory Analytics**
   - Track which blocks are accessed most frequently
   - Suggest rarely-used blocks for archival

3. **Collaborative Memory**
   - SQLite WAL for concurrent access
   - Conflict resolution for team projects

4. **Memory Sync**
   - Optional cloud backup sync
   - Version control integration (git)

## Rollout Plan

### Week 1: Core Infrastructure
- [ ] Database schema design
- [ ] SQLite connection manager
- [ ] Basic CRUD operations
- [ ] Unit tests

### Week 2: Memory API
- [ ] MemoryBlocks class (Letta-compatible)
- [ ] Default block initialization
- [ ] Character limit enforcement
- [ ] Integration tests

### Week 3: Context & Injection
- [ ] MemoryContext manager
- [ ] BlockInjector for prompt building
- [ ] Caching layer
- [ ] CLI commands

### Week 4: Advanced Features
- [ ] Block history and rollback
- [ ] Memory search (keyword + semantic)
- [ ] Export/import functionality
- [ ] Letta migration tool

### Week 5: Polish & Documentation
- [ ] Performance optimizations
- [ ] Error handling and edge cases
- [ ] Documentation and examples
- [ ] User guide

## Success Metrics

1. **Performance**
   - Block read: < 10ms (cached), < 50ms (database)
   - Block write: < 100ms
   - List all blocks: < 100ms

2. **Functionality Parity**
   - All Letta block operations supported
   - Same block structure and metadata
   - Compatible with Letta's prompt injection format

3. **Reliability**
   - ACID transactions for all writes
   - No data loss in crash scenarios
   - History rollback works correctly

4. **Usability**
   - Intuitive CLI commands
   - Clear error messages
   - Migration from Letta works seamlessly

## Conclusion

This plan implements a fully local, SQLite-based memory system that replicates Letta's memory block architecture while providing additional benefits:
- **Offline capability**: No network required
- **Performance**: Direct database access, no API latency
- **Data ownership**: Full control over memory data
- **Advanced features**: History, search, compression, analytics
- **Migration support**: Smooth transition from Letta cloud

The implementation follows Letta's design principles while leveraging SQLite's strengths for relational data storage and efficient querying.
