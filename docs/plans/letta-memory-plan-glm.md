# Letta Memory: Local SQLite Implementation Plan

## Executive Summary

This plan describes how to reproduce Letta Code's memory-first, agent-based architecture using a fully local SQLite database. The goal is to provide persistent, cross-session agent memory with intelligent retrieval, skills management, and conversation history - all running locally without external dependencies.

## 1. Core Architecture

### 1.1 Design Principles

1. **Local-First**: All data stored locally in SQLite; optional cloud sync
2. **Agent Identity**: Same agent persists across sessions via agent ID
3. **Memory Blocks**: Structured storage with labels, descriptions, values, and limits
4. **Intelligent Retrieval**: Not all memory loaded every time - dynamic context injection
5. **Skills System**: Reusable capabilities from .skills directories
6. **Self-Contained**: Single binary with embedded SQLite; no API dependency for core features

### 1.2 Comparison with Letta Code

| Feature | Letta Code | Local SQLite Implementation |
|---------|-----------|----------------------------|
| Memory Storage | Remote API | Local SQLite |
| Conversation History | Remote API | Local SQLite |
| Agent Identity | Server-side UUID | Content hash + UUID |
| Memory Retrieval | All blocks loaded | Dynamic retrieval |
| Skills | Remote + local files | Local files + stored in DB |
| Offline Capability | No | Yes |
| Privacy | Sends code to API | Fully local |

## 2. Database Schema

### 2.1 Core Tables

```sql
-- Agents: Represents a persistent coding assistant
CREATE TABLE agents (
    id TEXT PRIMARY KEY,  -- UUID
    name TEXT NOT NULL,
    description TEXT,
    model TEXT DEFAULT 'claude-sonnet-4',
    system_prompt TEXT,
    created_at INTEGER NOT NULL,  -- Unix timestamp
    updated_at INTEGER NOT NULL,
    is_pinned INTEGER DEFAULT 0,
    pin_type TEXT,  -- 'local', 'global', NULL
    config_json TEXT  -- JSON for agent-specific settings
);

-- Memory Blocks: Structured memory storage
CREATE TABLE memory_blocks (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    label TEXT NOT NULL,
    description TEXT NOT NULL,
    value TEXT NOT NULL,
    limit INTEGER,  -- Optional character limit
    read_only INTEGER DEFAULT 0,
    scope TEXT DEFAULT 'agent',  -- 'global', 'project', 'agent'
    project_path TEXT,  -- NULL for global/agent-scoped
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    embedding BLOB,  -- Vector embedding for semantic search
    FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE,
    UNIQUE(agent_id, label, project_path)
);

-- Conversations: Individual sessions with an agent
CREATE TABLE conversations (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    project_path TEXT NOT NULL,
    started_at INTEGER NOT NULL,
    ended_at INTEGER,
    summary TEXT,  -- Compressed summary for context
    metadata_json TEXT,  -- Additional session metadata
    FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE
);

-- Messages: Individual messages in a conversation
CREATE TABLE messages (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    role TEXT NOT NULL,  -- 'user', 'assistant', 'system', 'tool'
    content TEXT NOT NULL,
    tokens INTEGER,  -- Approximate token count
    tool_calls TEXT,  -- JSON array of tool calls
    created_at INTEGER NOT NULL,
    embedding BLOB,  -- Vector embedding for semantic search
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);

-- Skills: Discovered and stored skills
CREATE TABLE skills (
    id TEXT PRIMARY KEY,  -- Skill identifier (e.g., "data-analysis")
    name TEXT NOT NULL,
    description TEXT,
    category TEXT,
    tags TEXT,  -- JSON array
    source TEXT NOT NULL,  -- 'bundled', 'global', 'project'
    path TEXT,  -- Path to skill file
    content TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Agent Skills: Skills loaded into an agent's memory
CREATE TABLE agent_skills (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    skill_id TEXT NOT NULL,
    loaded_at INTEGER NOT NULL,
    loaded_from TEXT NOT NULL,  -- 'bundled', 'global', 'project'
    FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE,
    UNIQUE(agent_id, skill_id)
);

-- Memory History: Track changes to memory blocks
CREATE TABLE memory_history (
    id TEXT PRIMARY KEY,
    block_id TEXT NOT NULL,
    old_value TEXT,
    new_value TEXT,
    changed_by TEXT NOT NULL,  -- 'user', 'agent', 'system'
    changed_at INTEGER NOT NULL,
    FOREIGN KEY (block_id) REFERENCES memory_blocks(id) ON DELETE CASCADE
);

-- Conversation Context: Pinned context for current task
CREATE TABLE conversation_context (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    context_type TEXT NOT NULL,  -- 'ticket', 'debug', 'decision', etc.
    content TEXT NOT NULL,
    pinned_at INTEGER NOT NULL,
    is_active INTEGER DEFAULT 1,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);
```

### 2.2 Full-Text Search Tables

```sql
-- FTS5 for message content
CREATE VIRTUAL TABLE messages_fts USING fts5(
    content,
    tokenize='trigram'
);

-- FTS5 for memory block content
CREATE VIRTUAL TABLE memory_blocks_fts USING fts5(
    value,
    label,
    description,
    tokenize='trigram'
);

-- Triggers to keep FTS in sync
CREATE TRIGGER messages_fts_insert AFTER INSERT ON messages BEGIN
    INSERT INTO messages_fts(rowid, content) VALUES (new.id, new.content);
END;

CREATE TRIGGER messages_fts_update AFTER UPDATE ON messages BEGIN
    UPDATE messages_fts SET content = new.content WHERE rowid = new.id;
END;

CREATE TRIGGER messages_fts_delete AFTER DELETE ON messages BEGIN
    DELETE FROM messages_fts WHERE rowid = old.id;
END;
```

### 2.3 Indexes

```sql
-- Performance indexes
CREATE INDEX idx_conversations_agent ON conversations(agent_id);
CREATE INDEX idx_conversations_project ON conversations(project_path, started_at DESC);
CREATE INDEX idx_messages_conversation ON messages(conversation_id, created_at);
CREATE INDEX idx_messages_agent ON messages(conversation_id)  -- Via join
CREATE INDEX idx_memory_blocks_agent ON memory_blocks(agent_id);
CREATE INDEX idx_memory_blocks_scope ON memory_blocks(scope, project_path);
CREATE INDEX idx_agent_skills_agent ON agent_skills(agent_id);
```

## 3. Memory Block System

### 3.1 Default Memory Blocks

| Label | Scope | Description | Read-Only |
|-------|-------|-------------|-----------|
| `persona` | Agent | Behavioral adaptations and preferences | No |
| `human` | Global | User preferences across projects | No |
| `project` | Project | Project-specific conventions, commands | No |
| `skills` | Agent | Available skills catalog | Yes |
| `loaded_skills` | Agent | Currently loaded skills | Yes |
| `style` | Global | Coding style preferences | No |
| `ticket` | Session | Current work item context | No |

### 3.2 Memory Block Management

#### Creating Blocks
```typescript
interface CreateMemoryBlock {
    agentId: string;
    label: string;
    description: string;
    value: string;
    limit?: number;
    readOnly?: boolean;
    scope?: 'global' | 'project' | 'agent';
    projectPath?: string;
}
```

#### Updating Blocks
- Track changes in `memory_history` table
- Maintain embedding vectors for semantic search
- Emit events for context invalidation

#### Block Scoping
- **Global blocks**: Shared across all projects, stored once
- **Project blocks**: Specific to a project directory
- **Agent blocks**: Private to a specific agent

### 3.3 Memory Retrieval Strategy

#### Tiered Loading

**Tier 1: Always Loaded**
- `persona` block
- `loaded_skills` block
- System prompt

**Tier 2: Project Context**
- `project` block (if in a project)
- `style` block (global)
- `human` block (global)

**Tier 3: Dynamic Retrieval**
- Search past conversations for relevant context
- Retrieve specific memory blocks based on:
  - Current file being edited
  - Git history of current file
  - User's current task
  - Recent conversation topics

**Tier 4: On-Demand**
- User-pinned context (tickets, debugging notes)
- Historical decisions
- Project-specific gotchas

#### Retrieval Algorithm

```typescript
async function retrieveContext(agentId: string, currentTask: string, currentFile?: string): Promise<ContextBundle> {
    const context = new ContextBundle();

    // Tier 1: Always load
    context.add(await loadMemoryBlock(agentId, 'persona'));
    context.add(await loadMemoryBlock(agentId, 'loaded_skills'));

    // Tier 2: Project context
    const projectPath = getProjectPath();
    if (projectPath) {
        context.add(await loadProjectBlock(projectPath, 'project'));
    }
    context.add(await loadGlobalBlock('style'));
    context.add(await loadGlobalBlock('human'));

    // Tier 3: Semantic search
    const relevantConversations = await searchConversations(currentTask, limit=3);
    for (const conv of relevantConversations) {
        context.add(conv.summary);
    }

    // Tier 4: File-specific context
    if (currentFile) {
        const fileHistory = await getFileHistory(currentFile);
        context.add(fileHistory);
    }

    // Respect token limits
    context.truncateToTokenLimit(getContextWindowLimit());

    return context;
}
```

## 4. Conversation History System

### 4.1 Conversation Lifecycle

```
User starts conversation → Create conversation record
↓
Messages exchanged → Append to messages table
↓
Context grows → Check token limit
↓
Limit exceeded → Compress old messages → Store summary
↓
User ends session → Update conversation.ended_at
```

### 4.2 Message Compression

When context window approaches limit:

1. **Rolling Summary**: Compress oldest messages into summary
2. **Keep Recent**: Always keep last N messages (e.g., 10)
3. **Keep Key**: Preserve important interactions (user pin, agent decisions)

```typescript
async function compressConversation(conversationId: string): Promise<void> {
    const messages = await getOldMessages(conversationId, olderThan: 20);
    const summary = await generateSummary(messages);

    // Store summary in memory or conversation metadata
    await updateConversationSummary(conversationId, summary);

    // Delete old messages or mark as compressed
    await markMessagesCompressed(messages.map(m => m.id));
}
```

### 4.3 Semantic Search

Use vector embeddings (with a local embedding model like `all-MiniLM-L6-v2`):

```sql
-- Find similar conversations
SELECT c.id, c.summary, distance(embedding, ?) as similarity
FROM conversations c
JOIN (SELECT id FROM messages WHERE embedding IS NOT NULL) m ON c.id = m.conversation_id
ORDER BY similarity
LIMIT 5;
```

## 5. Skills System

### 5.1 Skill Discovery

Three sources (priority order):
1. **Project skills**: `.skills/` in current directory
2. **Global skills**: `~/.letta-memory/skills/`
3. **Bundled skills**: Embedded in binary

```typescript
async function discoverSkills(projectPath: string): Promise<Skill[]> {
    const skills: Skill[] = [];

    // Load bundled skills
    skills.push(...loadBundledSkills());

    // Load global skills
    const globalPath = path.join(os.homedir(), '.letta-memory', 'skills');
    skills.push(...loadSkillsFromDir(globalPath, 'global'));

    // Load project skills
    const projectSkillsPath = path.join(projectPath, '.skills');
    if (fs.existsSync(projectSkillsPath)) {
        skills.push(...loadSkillsFromDir(projectSkillsPath, 'project'));
    }

    // Project skills override global, which override bundled
    return deduplicateBySkillId(skills);
}
```

### 5.2 Skill Loading

Skills are loaded into memory as blocks:

```typescript
async function loadSkillIntoMemory(agentId: string, skill: Skill): Promise<void> {
    // Store skill in skills table
    await db.run(`
        INSERT OR REPLACE INTO skills
        (id, name, description, category, tags, source, path, content, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `, [skill.id, skill.name, skill.description, skill.category,
        JSON.stringify(skill.tags), skill.source, skill.path,
        skill.content, Date.now(), Date.now()]);

    // Add to agent's loaded skills
    await db.run(`
        INSERT INTO agent_skills (id, agent_id, skill_id, loaded_at, loaded_from)
        VALUES (?, ?, ?, ?, ?)
    `, [uuidv4(), agentId, skill.id, Date.now(), skill.source]);

    // Update loaded_skills memory block
    await updateLoadedSkillsBlock(agentId);
}
```

### 5.3 Skill Learning

Agent can learn new skills from its trajectory:

```typescript
async function learnSkillFromTrajectory(conversationId: string, skillName: string): Promise<void> {
    // Extract the skill from the conversation
    const skill = await extractSkillFromMessages(conversationId);

    // Save to global or project skills directory
    const skillPath = await saveSkillToDisk(skill, 'project');

    // Reload skills
    await refreshSkills();
}
```

## 6. Agent Identity System

### 6.1 Agent Creation

```typescript
interface CreateAgentOptions {
    name?: string;
    model?: string;
    systemPrompt?: string;
    initBlocks?: string[];  // Which memory blocks to initialize
}

async function createAgent(options: CreateAgentOptions): Promise<Agent> {
    const agentId = uuidv4();

    // Create agent record
    await db.run(`
        INSERT INTO agents (id, name, model, system_prompt, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?)
    `, [agentId, options.name || 'Default Agent', options.model || 'claude-sonnet-4',
        options.systemPrompt || '', Date.now(), Date.now()]);

    // Initialize default memory blocks
    const defaultBlocks = ['persona', 'human', 'style'];
    for (const blockLabel of defaultBlocks) {
        await createMemoryBlock(agentId, blockLabel, getDefaultBlockContent(blockLabel));
    }

    return await getAgent(agentId);
}
```

### 6.2 Agent Portability

Agents can be exported/imported:

```typescript
interface AgentExport {
    agent: Agent;
    memoryBlocks: MemoryBlock[];
    skills: Skill[];
    conversations: ConversationSummary[];  // Optional: include recent conversations
}

async function exportAgent(agentId: string, includeConversations = false): Promise<AgentExport> {
    const agent = await getAgent(agentId);
    const memoryBlocks = await getAgentMemoryBlocks(agentId);
    const skills = await getAgentSkills(agentId);
    const conversations = includeConversations
        ? await getAgentConversations(agentId, limit: 10)
        : [];

    return { agent, memoryBlocks, skills, conversations };
}
```

## 7. Context Injection System

### 7.1 Dynamic Context Loading

```typescript
interface ContextBundle {
    systemPrompt: string;
    memoryBlocks: Map<string, string>;
    conversationHistory: Message[];
    relevantContext: string[];
    totalTokens: number;
}

async function buildContextForRequest(
    agentId: string,
    userMessage: string,
    currentFile?: string
): Promise<ContextBundle> {
    const context = new ContextBundle();

    // Load system prompt
    context.systemPrompt = await getSystemPrompt(agentId);

    // Load essential memory blocks
    const persona = await loadMemoryBlock(agentId, 'persona');
    const loadedSkills = await loadMemoryBlock(agentId, 'loaded_skills');
    context.addMemoryBlock('persona', persona.value);
    context.addMemoryBlock('loaded_skills', loadedSkills.value);

    // Load project-specific blocks
    if (currentFile) {
        const projectPath = path.dirname(currentFile);
        const projectBlock = await loadProjectBlock(projectPath, 'project');
        if (projectBlock) {
            context.addMemoryBlock('project', projectBlock.value);
        }
    }

    // Search for relevant past conversations
    const relevantConvos = await searchConversationsByEmbedding(
        agentId,
        userMessage,
        limit: 2
    );
    for (const convo of relevantConvos) {
        context.addRelevantContext(convo.summary);
    }

    // Load recent messages from current conversation
    const recentMessages = await getRecentMessages(agentId, limit: 10);
    context.conversationHistory = recentMessages;

    // Trim to context window limit
    context.trimToTokenLimit(getContextWindowLimit());

    return context;
}
```

### 7.2 Token Budget Management

```typescript
class ContextBudget {
    private totalBudget: number;
    private allocations: Map<string, number>;

    constructor(totalBudget: number) {
        this.totalBudget = totalBudget;
        this.allocations = new Map([
            ['system_prompt', 0.05],  // 5%
            ['memory_blocks', 0.20],  // 20%
            ['conversation_history', 0.50],  // 50%
            ['relevant_context', 0.20],  // 20%
            ['margin', 0.05]  // 5% buffer
        ]);
    }

    getBudget(component: string): number {
        return this.totalBudget * (this.allocations.get(component) || 0);
    }

    checkBudget(components: Map<string, number>): boolean {
        let total = 0;
        for (const [key, value] of components) {
            total += value;
        }
        return total <= this.totalBudget;
    }
}
```

## 8. Implementation Phases

### Phase 1: Core Database (Week 1)
- [ ] SQLite schema implementation
- [ ] Basic CRUD operations for agents and memory blocks
- [ ] Migration system for schema updates
- [ ] CLI for database management

### Phase 2: Memory System (Week 2)
- [ ] Memory block creation/update/delete
- [ ] Block scoping (global/project/agent)
- [ ] Memory history tracking
- [ ] CLI commands for memory management

### Phase 3: Conversation System (Week 3)
- [ ] Conversation creation and lifecycle
- [ ] Message storage and retrieval
- [ ] Context window management
- [ ] Message compression and summarization

### Phase 4: Skills System (Week 4)
- [ ] Skill discovery (bundled/global/project)
- [ ] Skill loading into memory
- [ ] Agent-skills association
- [ ] CLI for skill management

### Phase 5: Context Retrieval (Week 5)
- [ ] Semantic search implementation (embeddings)
- [ ] FTS5 full-text search
- [ ] Dynamic context injection
- [ ] Token budget management

### Phase 6: CLI Integration (Week 6)
- [ ] Agent creation and management
- [ ] Interactive mode with context loading
- [ ] Conversation history search
- [ ] Agent export/import

### Phase 7: Advanced Features (Week 7-8)
- [ ] Skill learning from trajectories
- [ ] Agent cloning and branching
- [ ] Multi-device sync (optional)
- [ ] Performance optimization

## 9. Technology Stack

### 9.1 Core
- **Language**: TypeScript/Node.js or Rust (for single binary)
- **Database**: SQLite3 with FTS5 extension
- **CLI**: Commander.js or Clap
- **Embeddings**: ONNX Runtime with quantized models (fully local)

### 9.2 Embedding Model Options
1. **all-MiniLM-L6-v2**: ~80MB, good balance of speed/accuracy
2. **bge-small-en-v1.5**: ~130MB, better accuracy
3. **Quantized versions**: ~20-30MB, faster inference

### 9.3 Optional Integrations
- **Ollama**: For local LLM inference (offline mode)
- **LLM APIs**: Claude, GPT, Gemini (optional for inference)
- **Cloud Sync**: Optional remote backup/sync

## 10. API Design

### 10.1 Core API

```typescript
// Agent Management
createAgent(options: CreateAgentOptions): Promise<Agent>
getAgent(agentId: string): Promise<Agent>
listAgents(): Promise<Agent[]>
deleteAgent(agentId: string): Promise<void>

// Memory Management
createMemoryBlock(params: CreateMemoryBlock): Promise<MemoryBlock>
updateMemoryBlock(blockId: string, value: string): Promise<void>
getMemoryBlock(agentId: string, label: string): Promise<MemoryBlock>
listMemoryBlocks(agentId: string): Promise<MemoryBlock[]>

// Context Retrieval
retrieveContext(agentId: string, query: string, options?: RetrieveOptions): Promise<ContextBundle>
searchConversations(agentId: string, query: string): Promise<Conversation[]>

// Skills
discoverSkills(projectPath: string): Promise<Skill[]>
loadSkill(agentId: string, skillId: string): Promise<void>
listAgentSkills(agentId: string): Promise<Skill[]>

// Conversations
createConversation(agentId: string, projectPath: string): Promise<Conversation>
appendMessage(conversationId: string, message: Message): Promise<void>
getConversationHistory(conversationId: string, options?: HistoryOptions): Promise<Message[]>
```

### 10.2 CLI Commands

```bash
# Agent management
letta-mem agent create --name "My Agent" --model claude-sonnet-4
letta-mem agent list
letta-mem agent info <agent-id>
letta-mem agent export <agent-id> --output agent.json
letta-mem agent import agent.json

# Memory management
letta-mem memory create --agent <id> --label project --description "Project info"
letta-mem memory update <block-id> --content "New content"
letta-mem memory list --agent <id>
letta-mem memory history <block-id>

# Interactive mode
letta-mem chat --agent <id> --project /path/to/project
letta-mem chat --agent <id> --context-file ./file.ts

# Skills
letta-mem skills discover --project /path/to/project
letta-mem skills load --agent <id> --skill data-analysis
letta-mem skills list --agent <id>

# Search
letta-mem search conversations --agent <id> --query "how to fix X"
letta-mem search memory --agent <id> --query "testing conventions"
```

## 11. Key Challenges & Solutions

### 11.1 Context Window Limits

**Challenge**: Memory grows over time, exceeding context window.

**Solution**:
- Rolling summarization of old messages
- Tiered loading (essential vs. optional context)
- Semantic retrieval instead of loading everything
- Memory block pruning (remove outdated content)

### 11.2 Semantic Search Performance

**Challenge**: Embedding search can be slow with large datasets.

**Solution**:
- Use quantized embedding models (ONNX)
- Cache embeddings in database
- Incremental indexing (only embed new content)
- Hybrid search (FTS + vector similarity)

### 11.3 Memory Staleness

**Challenge**: Project changes, old memory becomes obsolete.

**Solution**:
- Track last_updated timestamps
- Periodic memory validation prompts
- User-initiated memory refresh
- Automatic pruning of very old content

### 11.4 Multi-Agent Coordination

**Challenge**: Multiple agents in same project, sharing context.

**Solution**:
- Global memory blocks (shared across agents)
- Project-scoped blocks (same project, different agents)
- Agent cloning (inherit memory from parent agent)
- Agent merging (combine memory from multiple agents)

## 12. Future Enhancements

1. **Subagents**: Specialized agents for specific tasks (testing, debugging)
2. **Memory Graph**: Explicit relationships between memory blocks
3. **Learning from Feedback**: Reinforcement learning for context retrieval
4. **Multi-Modal Memory**: Store images, diagrams, audio transcriptions
5. **Temporal Memory**: Track project evolution over time
6. **Collaborative Memory**: Share agents between team members
7. **Memory Templates**: Pre-configured memory for different project types
8. **Smart Pruning**: ML-based memory importance scoring

## 13. Success Metrics

1. **Context Relevance**: 80%+ of retrieved context is relevant to current task
2. **Memory Retention**: Agents remember >90% of important information after 30 days
3. **Performance**: Context retrieval <100ms, memory operations <50ms
4. **Storage Efficiency**: Compressed memory <50MB per active agent
5. **User Satisfaction**: Subjective rating of memory helpfulness

## 14. Security & Privacy Considerations

1. **Encryption**: Encrypt database at rest (optional, via SQLite encryption extension)
2. **Access Control**: OS-level file permissions on database
3. **Data Isolation**: Each agent's data in separate rows with proper foreign keys
4. **No External Transmissions**: By default, nothing leaves the local machine
5. **Audit Log**: Track all memory changes in history table
6. **Backup**: Easy export/import for backup and migration

## 15. Conclusion

This implementation plan provides a comprehensive local SQLite-based memory system that reproduces Letta Code's agent-based architecture while addressing its key limitations:

- **Full offline capability**: No API dependency for core features
- **Privacy**: All data stays local
- **Performance**: SQLite is fast and lightweight
- **Portability**: Single binary with embedded database
- **Flexibility**: Dynamic context retrieval instead of brute-force loading

The phased approach ensures incremental progress with continuous validation at each stage.
