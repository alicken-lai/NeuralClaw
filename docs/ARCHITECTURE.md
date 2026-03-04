# ZeroClaw + DMN Architecture

## Overview
This Go application (`zclaw`) serves as the central orchestrator for:
1. **ZeroClaw**: Agent runtime orchestrator. (Wrapped via CLI/RPC)
2. **memory-lancedb-pro**: Long-term memory subsystem. (Wrapped via HTTP API)
3. **GLM-OCR**: Ingestion pipeline for images/PDFs. (Wrapped via CLI Subprocess)

## Design Principles
- **Monorepo structure**: Code is categorized by functionality (`agent`, `memory`, `ingest`, `dmn`).
- **Adapter pattern**: External non-Go tools are wrapped via their respective adapters in `/internal`.
- **Multi-Scope Memory**: Enforces memory isolation across contexts (global, project, user, session).
- **Scheduled DMN**: A reflection/consolidation module that runs daily.

## Type Taxonomy
- **MemoryItem**: The core storage record. Contains metadata, text, and vector embeddings.
- **ItemType**: Hard typing mapping between `raw`, `daily_summary`, `concept_edges`, and `anomalies`. This prevents standard searches from returning generalized summaries when looking for raw quotes.
- **ScopeLevel**: Determines `global`, `project:(id)`, or `user:(id)` constraints on memory.

## Module Breakdown

### `internal/agent`
Contains definitions for the agent runtime and dependency abstractions (e.g. `Embedder`).

### `internal/ingest`
Pipeline for creating memory items from external inputs. The `ocrglm` adapter extracts text natively, while the generic pipeline chunks the text, invokes the `Embedder`, and writes `ItemTypeRaw` records to the `MemoryStore`.

### `internal/memory`
Houses the active memory implementation. `lancedbpro` provides the concrete adapter, while `hybrid` routes queries dynamically using vector searches, BM25 filters, and enforcing strict `scope` isolations.

### `internal/dmn`
(Daily Memory Network) module logic. Handles time-aware consolidation. Periodically fetches records from the `MemoryStore`, clusters them, generates summaries, and writes them back utilizing explicit DMN item types (`ItemTypeDailySummary` and `ItemTypeConceptEdges`).
