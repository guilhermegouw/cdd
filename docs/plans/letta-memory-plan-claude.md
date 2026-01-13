# Letta Memory Plan - Local SQLite Implementation

**Plan Version:** 1.0  
**Created:** 2024-12-30  
**Goal:** Replicate Letta Code's memory architecture using a fully local SQLite backend

---

## Executive Summary

This plan outlines the implementation of `letta-memory`, a local-first memory management system that replicates Letta Code's block-based memory architecture. The key difference: all memory storage, retrieval, and management happens locally via SQLite, eliminating dependency on external Letta servers while preserving the sophisticated memory semantics Letta pioneered.

### Core Philosophy

| Letta Cloud Approach | Local SQLite Approach |
|---------------------|----------------------|
| Remote API for all operations | Local database operations |
| Server-side memory blocks | SQLite-backed blocks with ACID |
| Network dependency | Fully offline-capable |
| Cloud scalability | Local performance optimization |
| Sleeptime agents (remote) | Background consolidation tasks |

---

## 1. Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                      letta-memory CLI                           │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │   Agent      │  │   Memory     │  │   Background         │  │
│  │   Manager    │  │   Engine     │  │   Consolidator       │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
│         │                │                     │                │
│         └────────────────┼─────────────────────┘                │
│                          ▼                                      │
│              ┌────────────────────────┐                        │
│              │   SQLite Database      │                        │
│              │   (letta-memory.db)    │                        │
│              │                        │                        │
│              │  ┌─────────────────┐   │                        │
│              │  │ agents          │   │                        │
│              │  │ blocks          │   │                        │
│              │  │ messages        │   │                        │
│              │  │ memory_events   │   │                        │
│              │  │ skills          │   │                        │
│              │  └─────────────────┘   │                        │
│              └────────────────────────┘                        │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. Database Schema

### 2.1 Core Tables

```sql
-- Main database file: ~/.letta/letta-memory.db

-- ============================================
-- AGENTS TABLE
-- ============================================
CREATE TABLE IF NOT EXISTS agents (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    model TEXT NOT NULL,
    embedding_model TEXT DEFAULT 'local/all-MiniLM-L6-v2',
    system_prompt TEXT,
    context_window_limit INTEGER DEFAULT 200000,
    parallel_tool_calls INTEGER DEFAULT 1,
    enable_sleeptime INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    metadata JSON
);

-- ============================================
-- MEMORY BLOCKS TABLE
-- ============================================
CREATE TABLE IF NOT EXISTS memory_blocks (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    label TEXT NOT NULL,
    value TEXT NOT NULL,
    description TEXT,
    limit_bytes INTEGER,
    read_only INTEGER DEFAULT 0,
    provenance TEXT DEFAULT 'new', -- 'new', 'global', 'project'
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE,
    UNIQUE(agent_id, label)
);

-- Index for fast label lookups
CREATE INDEX IF NOT EXISTS idx_blocks_agent_label 
ON memory_blocks(agent_id, label);

-- ============================================
-- MESSAGES TABLE (Conversation History)
-- ============================================
CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    role TEXT NOT NULL, -- 'user', 'assistant', 'system'
    content TEXT NOT NULL,
    token_count INTEGER,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_messages_agent_time 
ON messages(agent_id, created_at DESC);

-- ============================================
-- MEMORY EVENTS TABLE (Audit Trail)
-- ============================================
CREATE TABLE IF NOT EXISTS memory_events (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    event_type TEXT NOT NULL, -- 'block_created', 'block_updated', 'block_deleted', 'consolidation'
    block_label TEXT,
    old_value TEXT,
    new_value TEXT,
    triggered_by TEXT, -- 'user', 'agent', 'system', 'consolidator'
    created_at INTEGER NOT NULL,
    FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_events_agent_time 
ON memory_events(agent_id, created_at DESC);

-- ============================================
-- SKILLS TABLE
-- ============================================
CREATE TABLE IF NOT EXISTS skills (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    content TEXT NOT NULL,
    source_path TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE,
    UNIQUE(agent_id, name)
);

-- ============================================
-- CONSOLIDATION LOG TABLE
-- ============================================
CREATE TABLE IF NOT EXISTS consolidation_sessions (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    started_at INTEGER NOT NULL,
    completed_at INTEGER,
    blocks_analyzed INTEGER DEFAULT 0,
    blocks_modified INTEGER DEFAULT 0,
    status TEXT DEFAULT 'running', -- 'running', 'completed', 'failed'
    summary TEXT,
    FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE
);

-- ============================================
-- PROJECT METADATA TABLE
-- ============================================
CREATE TABLE IF NOT EXISTS project_metadata (
    path TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    last_agent_id TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    metadata JSON
);
```

### 2.2 Block Label Enum

```sql
-- Block labels are constrained to known values
CREATE TABLE IF NOT EXISTS block_labels (
    label TEXT PRIMARY KEY,
    block_type TEXT NOT NULL, -- 'global', 'project', 'shared'
    description TEXT,
    default_limit INTEGER,
    read_only DEFAULT 0
);

INSERT OR IGNORE INTO block_labels (label, block_type, description, default_limit, read_only) VALUES
    ('persona', 'global', 'Agent persona and personality', 8000, 0),
    ('human', 'global', 'Human context and preferences', 4000, 0),
    ('project', 'project', 'Project-specific context', 16000, 0),
    ('skills', 'project', 'Available skills catalog', 32000, 1),
    ('loaded_skills', 'project', 'Currently loaded skills', 32000, 1),
    ('style', 'project', 'Communication style preferences', 2000, 0);
```

---

## 3. Core Modules

### 3.1 Module Structure

```
src/
├── db/
│   ├── schema.ts          # Database initialization
│   ├── connection.ts      # Connection pool & singleton
│   ├── migrations.ts      # Versioned schema migrations
│   └── backup.ts          # Database backup/restore
├── agent/
│   ├── manager.ts         # Agent CRUD operations
│   ├── context.ts         # Global agent context (Symbol-based)
│   └── lifecycle.ts       # Agent lifecycle hooks
├── memory/
│   ├── blocks.ts          # Block CRUD operations
│   ├── search.ts          # Semantic/keyword search
│   ├── consolidation.ts   # Memory consolidation engine
│   └── limits.ts          # Block size limit enforcement
├── skills/
│   ├── manager.ts         # Skill discovery & management
│   └── loader.ts          # Skill loading/unloading
├── consolidator/
│   ├── scheduler.ts       # Periodic consolidation jobs
│   ├── analyzer.ts        # Memory usage analysis
│   └── optimizer.ts       # Memory optimization suggestions
├── context/
│   └── session.ts         # CLI session context
└── cli/
    ├── commands/          # CLI command implementations
    └── tools/             # Tool definitions
```

---

## 4. Implementation Phases

### Phase 1: Foundation (Week 1)

#### Goal: Core database and agent management

**Deliverables:**

1. **Database Connection Module** (`src/db/connection.ts`)
   ```typescript
   import Database from 'bun:sqlite';
   
   class DatabaseManager {
     private static instance: DatabaseManager;
     private db: Database;
     
     private constructor(dbPath: string) {
       this.db = new Database(dbPath);
       this.initializeSchema();
     }
     
     static getInstance(): DatabaseManager { /* singleton */ }
     static getDatabase(): Database { /* direct access */ }
   }
   ```

2. **Schema Initialization** (`src/db/schema.ts`)
   - Execute all CREATE TABLE statements
   - Insert block label enum values
   - Create indexes for performance

3. **Agent Manager** (`src/agent/manager.ts`)
   - `createAgent()` - Create agent with default blocks
   - `getAgent()` - Retrieve agent by ID
   - `listAgents()` - List all agents
   - `updateAgent()` - Update agent config
   - `deleteAgent()` - Delete agent and cascade

4. **Memory Blocks CRUD** (`src/memory/blocks.ts`)
   - `createBlock()` - Create labeled block
   - `getBlock()` - Retrieve block by label
   - `updateBlock()` - Update block content
   - `deleteBlock()` - Delete block
   - `listBlocks()` - List all blocks for agent

**Tests:**
- Unit tests for each CRUD operation
- Integration test for agent creation flow
- Schema validation tests

---

### Phase 2: Memory Semantics (Week 2)

#### Goal: Implement Letta's sophisticated memory features

**Deliverables:**

1. **Block Size Limits** (`src/memory/limits.ts`)
   ```typescript
   interface LimitConfig {
     softLimit: number;  // Warning threshold
     hardLimit: number;  // Maximum allowed
   }
   
   function enforceBlockLimits(
     block: MemoryBlock, 
     config: LimitConfig
   ): EnforcementResult {
     const byteCount = new Blob([block.value]).size;
     
     if (byteCount > config.hardLimit) {
       return {
         action: 'truncate',
         originalSize: byteCount,
         truncatedSize: config.hardLimit,
         reason: 'Block exceeds hard limit'
       };
     }
     
     if (byteCount > config.softLimit) {
       return {
         action: 'warn',
         originalSize: byteCount,
         truncatedSize: byteCount,
         reason: 'Block approaching limit'
       };
     }
     
     return { action: 'none' };
   }
   ```

2. **Memory Search** (`src/memory/search.ts`)
   - Keyword-based search across blocks
   - Optional semantic search (local embeddings via Transformers.js)
   - Weighted scoring by block label

3. **Memory Events Audit** (`src/memory/events.ts`)
   - Track all block modifications
   - Enable rollback to previous states
   - Provide change history

4. **Read-Only Blocks** (`src/memory/readonly.ts`)
   - Mark blocks as read-only (skills, loaded_skills)
   - Prevent direct modification via memory tools
   - Allow modification via dedicated tools only

**Tests:**
- Limit enforcement edge cases
- Search result accuracy
- Event logging completeness

---

### Phase 3: Skills System (Week 3)

#### Goal: Local skills discovery and management

**Deliverables:**

1. **Skills Manager** (`src/skills/manager.ts`)
   - `discoverSkills()` - Scan `.skills/` directory
   - `registerSkill()` - Add skill to database
   - `unregisterSkill()` - Remove skill
   - `loadSkills()` - Load skills into agent memory
   - `unloadSkills()` - Unload skills from memory

2. **Skills Directory Watcher** (`src/skills/watcher.ts`)
   - Watch `.skills/` for changes
   - Auto-reload on file modifications
   - Debounced updates

3. **Skills Memory Integration**
   - Sync skills to `skills` block
   - Track loaded skills in `loaded_skills` block
   - Skills appear in agent's available tools

**File Structure for Skills:**
```
.skills/
├── skill-name/
│   ├── SKILL.md           # Skill definition & instructions
│   ├── implementation.js  # Optional code implementation
│   └── config.json        # Skill metadata
```

**Tests:**
- Discovery from various directory structures
- Skill loading/unloading scenarios
- Watcher debounce behavior

---

### Phase 4: Consolidation Engine (Week 4)

#### Goal: Automated memory management (sleeptime agent equivalent)

**Deliverables:**

1. **Consolidation Scheduler** (`src/consolidator/scheduler.ts`)
   ```typescript
   interface ConsolidationConfig {
     intervalMs: number;        // How often to run
     maxBlockSize: number;      // Target max size
     compressionLevel: number;  // 0-9, higher = more aggressive
     enabled: boolean;
   }
   
   class ConsolidationScheduler {
     private timer: Timer | null = null;
     
     start(config: ConsolidationConfig): void {
       this.timer = setInterval(
         () => this.runConsolidation(),
         config.intervalMs
       );
     }
     
     stop(): void {
       clearInterval(this.timer);
     }
     
     async runConsolidation(): Promise<ConsolidationResult> {
       // Analyze all agents
       // Identify optimization opportunities
       // Apply optimizations
       // Log results
     }
   }
   ```

2. **Memory Analyzer** (`src/consolidator/analyzer.ts`)
   - Analyze block sizes and growth patterns
   - Identify redundant information
   - Suggest block reorganizations
   - Calculate memory efficiency scores

3. **Memory Optimizer** (`src/consolidator/optimizer.ts`)
   - Merge similar blocks
   - Summarize verbose content (local LLM or rule-based)
   - Archive old conversations to cold storage
   - Compress low-activity blocks

4. **Consolidation Events**
   - Log each consolidation session
  /after metrics - Track before
   - Enable rollback of consolidations

**Consolidation Strategies:**

| Strategy | Description | Use Case |
|----------|-------------|----------|
| `summarize` | Reduce verbose blocks | Old project context |
| `merge` | Combine similar content | Duplicate information |
| `archive` | Move to separate table | Historical messages |
| `compress` | Apply compression | Large skill descriptions |

**Tests:**
- Consolidation accuracy
- Performance impact
- Rollback functionality

---

### Phase 5: Context & CLI Integration (Week 5)

#### Goal: CLI experience matching Letta Code

**Deliverables:**

1. **Global Context Module** (`src/agent/context.ts`)
   ```typescript
   const CONTEXT_KEY = Symbol.for("@letta-memory/agentContext");
   
   interface AgentContext {
     agentId: string | null;
     skillsDirectory: string | null;
     hasLoadedSkills: boolean;
     sessionStart: number;
   }
   
   function getContext(): AgentContext {
     const global = globalThis as GlobalWithContext;
     if (!global[CONTEXT_KEY]) {
       global[CONTEXT_KEY] = {
         agentId: null,
         skillsDirectory: null,
         hasLoadedSkills: false,
         sessionStart: Date.now(),
       };
     }
     return global[CONTEXT_KEY];
   }
   ```

2. **Session Management** (`src/context/session.ts`)
   - Track active session
   - Manage message history
   - Handle session context

3. **CLI Commands**
   - `init` - Initialize agent with default blocks
   - `remember` - Add/update memory
   - `forget` - Remove memory
   - `mem` - View current memory blocks
   - `skills` - Manage skills
   - `consolidate` - Manual consolidation trigger

4. **Message History**
   - Store messages in SQLite
   - Retrieve for context window management
   - Persist across sessions

**Tests:**
- Context singleton behavior
- CLI command integration
- Session persistence

---

## 5. API Reference

### 5.1 Agent API

```typescript
interface CreateAgentOptions {
  name?: string;
  model?: string;
  systemPrompt?: string;
  embeddingModel?: string;
  contextWindow?: number;
  enableSleeptime?: boolean;
  skillsDirectory?: string;
  initialBlocks?: MemoryBlockLabel[];
}

interface Agent {
  id: string;
  name: string;
  model: string;
  systemPrompt: string;
  createdAt: Date;
  updatedAt: Date;
}

// Methods
const agentManager = {
  async create(options: CreateAgentOptions): Promise<Agent>,
  async get(id: string): Promise<Agent | null>,
  async list(): Promise<Agent[]>,
  async update(id: string, updates: Partial<Agent>): Promise<Agent>,
  async delete(id: string): Promise<void>,
};
```

### 5.2 Memory Block API

```typescript
interface MemoryBlock {
  id: string;
  agentId: string;
  label: string;
  value: string;
  description?: string;
  limit?: number;
  readOnly: boolean;
  provenance: 'new' | 'global' | 'project';
  createdAt: Date;
  updatedAt: Date;
}

const memoryManager = {
  async createBlock(agentId: string, block: CreateBlock): Promise<MemoryBlock>,
  async getBlock(agentId: string, label: string): Promise<MemoryBlock | null>,
  async updateBlock(agentId: string, label: string, value: string): Promise<MemoryBlock>,
  async deleteBlock(agentId: string, label: string): Promise<void>,
  async listBlocks(agentId: string): Promise<MemoryBlock[]>,
  async searchBlocks(agentId: string, query: string): Promise<SearchResult[]>,
};
```

### 5.3 Consolidation API

```typescript
interface ConsolidationResult {
  sessionId: string;
  agentId: string;
  blocksAnalyzed: number;
  blocksModified: number;
  optimizations: Optimization[];
  startedAt: Date;
  completedAt: Date;
}

interface Optimization {
  blockLabel: string;
  type: 'summarize' | 'merge' | 'archive' | 'compress';
  originalSize: number;
  newSize: number;
  reason: string;
}

const consolidator = {
  async schedule(config: ConsolidationConfig): Promise<void>,
  async runNow(agentId?: string): Promise<ConsolidationResult>,
  async getHistory(agentId: string): Promise<ConsolidationResult[]>,
  async rollback(sessionId: string): Promise<void>,
};
```

---

## 6. Performance Considerations

### 6.1 Index Strategy

```sql
-- Agent lookups
CREATE INDEX IF NOT EXISTS idx_agents_name ON agents(name);
CREATE INDEX IF NOT EXISTS idx_agents_model ON agents(model);

-- Block operations
CREATE INDEX IF NOT EXISTS idx_blocks_agent_updated ON memory_blocks(agent_id, updated_at DESC);

-- Message operations  
CREATE INDEX IF NOT EXISTS idx_messages_role ON messages(agent_id, role);

-- Event filtering
CREATE INDEX IF NOT EXISTS idx_events_type ON memory_events(agent_id, event_type);
```

### 6.2 Connection Pooling

```typescript
// For concurrent access
class ConnectionPool {
  private pool: Database[] = [];
  private maxConnections = 5;
  
  async acquire(): Promise<Database> {
    if (this.pool.length > 0) {
      return this.pool.pop()!;
    }
    if (this.pool.length < this.maxConnections) {
      return new Database(this.dbPath);
    }
    // Wait for release
    return new Promise(resolve => {
      const check = setInterval(() => {
        if (this.pool.length > 0) {
          clearInterval(check);
          resolve(this.pool.pop()!);
        }
      }, 10);
    });
  }
  
  release(db: Database): void {
    this.pool.push(db);
  }
}
```

### 6.3 WAL Mode for Performance

```sql
-- Enable WAL mode for better concurrent access
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = 10000;  -- 10MB cache
PRAGMA temp_store = MEMORY;
PRAGMA mmap_size = 268435456;  -- 256MB memory-mapped I/O
```

---

## 7. Migration Strategy

### 7.1 Versioned Migrations

```typescript
// src/db/migrations.ts
const migrations = [
  {
    version: 1,
    up: () => {
      // Initial schema
    },
    down: () => {
      // Rollback
    }
  },
  {
    version: 2,
    up: () => {
      // Add new features
    },
    down: () => {
      // Rollback
    }
  }
];

async function migrate(): Promise<void> {
  const currentVersion = await getUserVersion();
  const targetVersion = migrations.length;
  
  if (currentVersion < targetVersion) {
    for (let i = currentVersion; i < targetVersion; i++) {
      await migrations[i].up();
    }
    await setUserVersion(targetVersion);
  }
}
```

### 7.2 Import from Letta Cloud

```typescript
async function importFromLettaCloud(
  apiKey: string,
  baseUrl: string
): Promise<void> {
  const client = new LettaClient({ apiKey, baseUrl });
  
  // Fetch all agents
  const agents = await client.agents.list();
  
  for (const agent of agents) {
    // Create local agent
    await agentManager.create({
      name: agent.name,
      model: agent.model,
      systemPrompt: agent.system,
    });
    
    // Import blocks
    const blocks = await client.agents.blocks.list(agent.id);
    for (const block of blocks.items) {
      await memoryManager.createBlock(agent.id, {
        label: block.label,
        value: block.value,
        description: block.description,
        limit: block.limit,
        readOnly: block.readOnly,
      });
    }
    
    // Import messages
    // Import skills
  }
}
```

---

## 8. Testing Strategy

### 8.1 Test Pyramid

```
        ┌───────────────┐
        │   E2E Tests   │  ← CLI integration, full workflows
        └───────┬───────┘
                │
    ┌───────────┴───────────┐
    │   Integration Tests   │  ← Module interactions, DB operations
    └───────────┬───────────┘
                │
    ┌───────────┴───────────┐
    │    Unit Tests         │  ← Individual functions, classes
    └───────────────────────┘
```

### 8.2 Test Coverage Goals

| Module | Coverage Goal |
|--------|--------------|
| Database | 100% |
| Agent Manager | 95% |
| Memory Blocks | 95% |
| Skills Manager | 90% |
| Consolidator | 85% |
| CLI Commands | 90% |

### 8.3 Test Fixtures

```typescript
// Test agents with pre-populated memory
const testAgent = {
  id: 'test-agent-1',
  name: 'Test Agent',
  model: 'anthropic/claude-sonnet-4-20250522',
  blocks: {
    persona: 'You are a test agent.',
    human: 'Test user preferences.',
    project: 'Test project context.',
  }
};
```

---

## 9. Future Enhancements

### 9.1 Semantic Search (Optional)

```typescript
// Using Transformers.js for local embeddings
import { pipeline } from '@xenova/transformers';

class LocalEmbeddingEngine {
  private extractor: Pipeline | null = null;
  
  async initialize(): Promise<void> {
    this.extractor = await pipeline(
      'feature-extraction', 
      'Xenova/all-MiniLM-L6-v2'
    );
  }
  
  async embed(text: string): Promise<number[]> {
    const output = await this.extractor(text, {
      pooling: 'mean',
      normalize: true
    });
    return Array.from(output.data);
  }
}
```

### 9.2 Multi-Project Support

```sql
-- Shared blocks across projects
CREATE TABLE IF NOT EXISTS shared_blocks (
    id TEXT PRIMARY KEY,
    label TEXT NOT NULL,
    value TEXT NOT NULL,
    description TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Link shared blocks to projects
CREATE TABLE IF NOT EXISTS project_shared_blocks (
    project_path TEXT NOT NULL,
    block_id TEXT NOT NULL,
    FOREIGN KEY (block_id) REFERENCES shared_blocks(id) ON DELETE CASCADE
);
```

### 9.3 Encryption

```typescript
// Optional encrypted storage
import { encrypt, decrypt } from './crypto';

async function storeEncryptedBlock(
  db: Database,
  block: MemoryBlock
): Promise<void> {
  const encrypted = await encrypt(block.value);
  await db.prepare(`
    UPDATE memory_blocks 
    SET value = ? 
    WHERE id = ?
  `).run(encrypted, block.id);
}
```

---

## 10. Success Criteria

### 10.1 Functional Requirements

- [ ] Create agent with all default memory blocks
- [ ] Update memory blocks by label
- [ ] Search across memory blocks
- [ ] Skills discovery and loading
- [ ] Periodic memory consolidation
- [ ] Full audit trail of memory changes
- [ ] Import from Letta Cloud

### 10.2 Performance Requirements

- [ ] Agent creation < 500ms
- [ ] Block update < 50ms
- [ ] Block search < 100ms
- [ ] Support 1000+ agents
- [ ] Support 10MB+ blocks

### 10.3 Compatibility Requirements

- [ ] Bun runtime
- [ ] Node.js runtime (via better-sqlite3)
- [ ] Cross-platform (Linux, macOS, Windows)

---

## 11. Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| SQLite limitations for large data | Medium | Archive old data, use compression |
| Local embedding model size | High | Make optional, use keyword fallback |
| Memory consolidation complexity | Medium | Start with simple rules, iterate |
| Data corruption | High | WAL mode, regular backups, integrity checks |

---

## 12. Implementation Timeline

```
Week 1: Foundation
├── Database schema and connection
├── Agent CRUD operations
└── Memory block basics

Week 2: Memory Semantics
├── Block size limits
├── Memory search
└── Audit events

Week 3: Skills System
├── Skills discovery
├── Skills loading/unloading
└── Directory watcher

Week 4: Consolidation
├── Scheduler
├── Analyzer
└── Optimizer

Week 5: CLI Integration
├── Context management
├── Session tracking
└── Full CLI commands
```

---

## 13. File Listing

```
letta-memory/
├── README.md
├── package.json
├── tsconfig.json
├── bunfig.toml
├── CLAUDE.md
├── src/
│   ├── index.ts              # Entry point
│   ├── cli/
│   │   ├── index.ts
│   │   └── commands/
│   │       ├── init.ts
│   │       ├── remember.ts
│   │       ├── forget.ts
│   │       ├── mem.ts
│   │       ├── skills.ts
│   │       └── consolidate.ts
│   ├── db/
│   │   ├── index.ts
│   │   ├── connection.ts
│   │   ├── schema.ts
│   │   ├── migrations.ts
│   │   └── backup.ts
│   ├── agent/
│   │   ├── index.ts
│   │   ├── manager.ts
│   │   ├── context.ts
│   │   └── lifecycle.ts
│   ├── memory/
│   │   ├── index.ts
│   │   ├── blocks.ts
│   │   ├── search.ts
│   │   ├── limits.ts
│   │   ├── events.ts
│   │   └── readonly.ts
│   ├── skills/
│   │   ├── index.ts
│   │   ├── manager.ts
│   │   └── loader.ts
│   ├── consolidator/
│   │   ├── index.ts
│   │   ├── scheduler.ts
│   │   ├── analyzer.ts
│   │   └── optimizer.ts
│   ├── context/
│   │   └── session.ts
│   ├── tools/
│   │   └── memory.ts
│   └── utils/
│       └── crypto.ts
├── test/
│   ├── db/
│   ├── agent/
│   ├── memory/
│   └── skills/
└── .letta/
    └── letta-memory.db       # Default database location
```

---

## 14. Dependencies

```json
{
  "dependencies": {
    "bun": "^1.2.0",
    "better-sqlite3": "^11.0.0",
    "uuid": "^10.0.0"
  },
  "devDependencies": {
    "@types/bun": "^1.2.0",
    "@types/better-sqlite3": "^7.6.11",
    "@types/uuid": "^10.0.0",
    "bun-test": "^1.2.0"
  },
  "optional": {
    "@xenova/transformers": "^2.17.0"  // Local embeddings
  }
}
```

---

## 15. Conclusion

This implementation plan provides a comprehensive roadmap for building a local-first memory management system that replicates Letta Code's sophisticated memory semantics while eliminating dependency on external servers.

Key innovations:
1. **ACID-compliant memory storage** via SQLite
2. **Sleeptime agent replacement** via scheduled consolidation
3. **Full offline capability** with optional cloud sync
4. **Transparent migration** from Letta Cloud

The phased approach ensures a working system at each milestone, with clear success criteria and testing strategies.
