# NeuralClaw Architecture

## Overview
This Go application (`neuralclaw`) serves as the central, unified orchestrator in a single binary. It replaces the distinct Python, Rust, and Node.js components of the past with pure Go native implementations:

1. **Agent Runtime**: Native Tool-Calling ReAct loop (replacing `zeroclaw`).
2. **Hybrid Memory**: Pure Go JSON-backed Vector/BM25 Store with **Living Memory** (replacing `memory-lancedb-pro`).
3. **OCR Ingestion**: Native Go HTTP client for Zhipu MaaS (replacing `GLM-OCR` Python wrapper).
4. **Token Analytics**: Built-in JSONL usage tracker + Web Dashboard for LLM cost visibility.

## Design Principles
- **Monolithic Pure Go**: Zero CGO, zero external binaries, zero Python requirements.
- **Provider Agnostic**: Embedders and Agent LLMs interact via generic Go interfaces (`internal/agent/llm`).
- **Multi-Scope Memory**: Enforces memory isolation across contexts (global, project, user).
- **Scheduled DMN**: A reflection/consolidation module that runs continuously in the background, summarizing and tagging memories autonomously.
- **Living Memory**: Memories evolve through usage — access frequency, recency, and causal lineage influence retrieval scoring.

## Type Taxonomy
- **MemoryItem**: The core storage record. Contains metadata, plaintext, normalized BM25 text, Vector embeddings, and **Living Memory fields** (`AccessCount`, `LastAccessedAt`, `CausalLinks`).
- **ItemType**: Hard typing mapping between `raw`, `daily_summary`, `weekly_summary`, `monthly_summary`, `concept_edges`, and `anomalies`. This prevents standard agent searches from hallucinating over generalized summaries when looking for raw quotes.
- **Scope**: Determines the constrained working space of the memory database.
- **TokenUsageLog**: Tracks per-call LLM token consumption with source, model, and timestamp for analytics.

## Module Breakdown

### `internal/agent`
Contains definitions for the agent runtime.
- `/llm`: Manages `ChatMessage`, `ToolCall`, `TokenUsage`, and the OpenAI-compatible REST Client.
- `tools.go`: Provides the native `Tool` interface and built-in capabilities (`ShellRunnerTool`, `FileReaderTool`).
- `dispatch.go`: Executes the native ReAct tool-calling loop in isolated goroutines. **Instrumented** with `TokenTracker` to record token consumption after each LLM call.

### `internal/ingest`
Pipeline for creating memory items from external inputs.
- `/ocr_glm`: A direct Go HTTP adapter for the Zhipu `layout_parsing` AI service.
- `pipeline.go`: Chunks the extracted text (paragraph-aware), invokes the `Embedder`, and writes `ItemTypeRaw` records to the active `MemoryStore`.

### `internal/memory`
Houses the active memory and search implementation.
- `/store`: Provides `JSONStore` (thread-safe transactional database), `Retriever` (Cosine Similarity + BM25 keyword scoring + RRF fusion + **LTP access boost**), and `Embedder` interfaces. The `touchAccess()` method auto-updates `AccessCount` and `LastAccessedAt` on every retrieval.
- `/reaper`: Scans the store on a schedule and deletes items based on their `ItemType` expiration policy.

### `internal/dmn`
(Default Mode Network) module logic. Handles time-aware consolidation. Periodically fetches records from the `MemoryStore`, invokes the native LLM to generate summaries, and writes them back with explicit DMN item types (`ItemTypeDailySummary`, `ItemTypeConceptEdges`). **Populates `CausalLinks`** to trace each summary back to its source raw memories—forming a memory DAG.

### `internal/observability`
- `logger.go`: Zap-based structured logger with global singleton.
- `token_tracker.go`: Append-only JSONL logger for LLM token consumption. Provides in-memory queries for daily summaries (grouped by model) and source summaries (grouped by TaskRun/Cron/Ingest).

### `internal/taskstore`
JSON-file-backed persistence for `Task` and `Run` entities. Provides CRUD operations for the Web GUI task queue.

### `internal/web`
Provides the native HTMX + Tailwind dashboard with a dark glassmorphic UI.
- **Dashboard** (`/web`): Workspace overview, task/run counts, system operations.
- **Task Queue** (`/web/tasks`): Create, view, and dispatch agent tasks.
- **Execution Runs** (`/web/runs`): History of agent runs with live SSE log streaming.
- **Token Usage** (`/web/tokens`): Daily LLM usage analytics (by model), per-source breakdown.
- **Context Browser** (`/web/context`): Browse project files with estimated token footprint (~4 bytes/token).
- Uses `html/template` with embedded filesystem (`//go:embed templates/*`).
