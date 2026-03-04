# NeuralClaw Architecture

## Overview
This Go application (`neuralclaw`) serves as the central, unified orchestrator in a single binary. It replaces the distinct Python, Rust, and Node.js components of the past with pure Go native implementations:

1. **Agent Runtime**: Native Tool-Calling ReAct loop (replacing `zeroclaw`).
2. **Hybrid Memory**: Pure Go JSON-backed Vector/BM25 Store (replacing `memory-lancedb-pro`).
3. **OCR Ingestion**: Native Go HTTP client for Zhipu MaaS (replacing `GLM-OCR` Python wrapper).

## Design Principles
- **Monolithic Pure Go**: Zero CGO, zero external binaries, zero Python requirements.
- **Provider Agnostic**: Embedders and Agent LLMs interact via generic Go interfaces (`internal/agent/llm`).
- **Multi-Scope Memory**: Enforces memory isolation across contexts (global, project, user).
- **Scheduled DMN**: A reflection/consolidation module that runs continuously in the background, summarizing and tagging memories autonomously.

## Type Taxonomy
- **MemoryItem**: The core storage record. Contains metadata, plaintext, normalized BM25 text, and Vector embeddings.
- **ItemType**: Hard typing mapping between `raw`, `daily_summary`, `weekly_summary`, `monthly_summary`, `concept_edges`, and `anomalies`. This prevents standard agent searches from hallucinating over generalized summaries when looking for raw quotes.
- **Scope**: Determines the constrained working space of the memory database.

## Module Breakdown

### `internal/agent`
Contains definitions for the agent runtime.
- `/llm`: Manages `ChatMessage`, `ToolCall`, and the OpenAI-compatible REST Client.
- `tools.go`: Provides the native `Tool` interface and built-in capabilities (e.g., `ShellRunnerTool`).
- `dispatch.go`: Executes the 15-iteration native ReAct tool-calling loop in isolated goroutines.

### `internal/ingest`
Pipeline for creating memory items from external inputs. 
- `/ocr_glm`: A direct Go HTTP adapter for the Zhipu `layout_parsing` AI service. 
- `pipeline.go`: Chunks the extracted text (paragraph-aware), invokes the `Embedder`, and writes `ItemTypeRaw` records to the active `MemoryStore`.

### `internal/memory`
Houses the active memory and search implementation.
- `/store`: Provides `JSONStore` (the thread-safe transactional database for vectors/metadata), `Retriever` (providing native Cosine Similarity and Keyword scoring + RRF), and `Embedder` interfaces.
- `/reaper`: Scans the store on a schedule and deletes items based on their `ItemType` expiration policy to keep token usage efficient.

### `internal/dmn`
(Default Mode Network) module logic. Handles time-aware consolidation. Periodically fetches records from the `MemoryStore`, invokes the native LLM to generate summaries, and writes them back utilizing explicit DMN item types (`ItemTypeDailySummary` and `ItemTypeConceptEdges`).

### `internal/web`
Provides the native HTMX + Tailwind dashboard. Uses `html/template` to server-render real-time task queues and utilizes Server-Sent Events (SSE) to stream output from the `agent` dispatcher to the frontend safely.
