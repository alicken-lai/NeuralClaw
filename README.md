# NeuralClaw 🧠🐾

**The Pure Go Autonomous Agent with a Default-Mode Brain**
*Zero CGO · Zero Python · Zero Rust · Single Binary. Deploy Anywhere.*

[![Go 1.25+](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
![Zero CGO](https://img.shields.io/badge/CGO-Disabled-brightgreen?style=flat)
![Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue?style=flat)

---

**NeuralClaw** is a unified, production-grade AI agent framework written entirely in Go. It draws inspiration from the human brain's *Default Mode Network (DMN)*—the part that consolidates long-term memory while you sleep—to give your agent a *reflective* second mind that works continuously in the background, keeping memories organized, summarized, and ready for instant recall.

**What makes it different:**

| | Traditional RAG Agents | NeuralClaw |
|---|---|---|
| **Dependencies** | Python, Node.js, Rust, Docker | **Go 1.25+ only** |
| **Vector Store** | External DB (LanceDB, Pinecone) | **Pure Go in-process store** |
| **OCR** | Python subprocess | **Native Zhipu GLM-OCR API client** |
| **Agent Loop** | Rust/Python runtimes | **Native pure-Go ReAct loop** |
| **Memory** | Static retrieval | **Living Memory with LTP boost & causal DAG** |
| **Memory Aging** | Manual cleanup | **Tiered retention with time-decay scoring** |
| **Explainability** | Black-box scores | **Per-result score breakdown & evidence chains** |
| **Eval** | External tooling | **Built-in Recall@K / MRR / NDCG evaluation** |

---

## ✨ Features

### 🧠 Dual-Brain Architecture
- **Foreground Agent (ReAct Loop)**: When you dispatch a task from the Web GUI or CLI, the native Go agent calls your LLM provider, receives tool calls, executes them concurrently, and loops until done—all within a single goroutine.
- **Background DMN Reflection**: While the agent is idle, the DMN process continuously distills raw memories into `daily_summary`, `weekly_summary`, `monthly_summary`, and `concept_edges`, keeping your context window lean.

### 🗃️ Pure Go Hybrid Memory Store
- **Dual-path retrieval**: Cosine-Similarity vector search + BM25 keyword scoring fused with linear combination.
- **Explainable scoring**: Every result can include a full `ScoreBreakdown` (vector, BM25, time decay, access boost, rerank, final).
- **Evidence chains**: Memories track `DerivedFrom` / `EvidenceOf` backlinks—the Web UI renders them as recursive trees.
- **Time-aware scoring**: Recency half-life, importance weighting, and hard floor cutoffs.
- **HTTP Reranking**: Client for Jina, SiliconFlow, and OpenAI-compatible rerank APIs.
- **Thread-safe JSON persistence**: Zero external dependencies, single-file database, readable and portable.
- **Living Memory**: Access frequency tracking (`AccessCount`, `LastAccessedAt`) with LTP-style score boost for frequently recalled memories. Causal linking (`CausalLinks`) builds a memory DAG for lineage tracing.

### 👁️ Native GLM-OCR Integration
- Reads PDFs, PNGs, and JPGs, encodes them as Base64 Data URIs, and calls Zhipu's `layout_parsing` API—all from native `net/http`.
- No Python runtime. No subprocess. Just HTTP.

### 🤖 OpenAI-Compatible LLM Provider
- Works with OpenAI, OpenRouter, Groq, DeepSeek, Ollama, or any OpenAI-compatible API.
- Native tool-calling JSON serialization with `stop_reason` normalization.

### 🎛️ Web Dashboard & Analytics
- HTMX + TailwindCSS via CDN, rendered by Go's `html/template` with dark glassmorphic UI.
- **Dashboard**: Create and dispatch tasks, view run history, stream live logs via SSE.
- **Memory Evidence Explorer**: Inline **Explain** and **Evidence** buttons on each memory item—click to see score breakdown tables and recursive evidence-chain trees, powered by HTMX.
- **Token Usage Analytics**: Daily LLM token consumption (grouped by model), per-source breakdown (TaskRun/Cron/Ingest).
- **Context File Browser**: Browse project files and estimate their LLM context footprint (~tokens per file).
- Token-based auth middleware with scope isolation.

### 📊 Built-in Retrieval Evaluation
- CLI command `zclaw eval retrieval` reads golden-query YAML files and computes **Recall@K**, **MRR@K**, and **NDCG@K**.
- Outputs a human-readable table to the terminal and an optional JSON report.
- No external eval frameworks needed.

### ⏳ Tiered Retention Policy
- Raw memories: 90 days
- Daily summaries: 2 years
- Weekly summaries: 5 years
- Concept edges: 5 years
- Monthly summaries: 10 years
- Background Reaper prunes expired items on a schedule.

---

## 🚀 Quick Start

### Requirements
- **Go 1.25+** (that's it)

### Build & Run

```bash
git clone https://github.com/alicken-lai/NeuralClaw.git
cd NeuralClaw

# Create your config
cp configs/config.example.yaml configs/config.yaml
# Edit configs/config.yaml and fill in your API keys

# Build (zero CGO)
CGO_ENABLED=0 go build -o neuralclaw ./cmd/zclaw

# Launch the web dashboard
./neuralclaw web
# → Open http://localhost:8080/web
```

---

## 🛠️ CLI Reference

```bash
# Run a single agent task directly from the terminal
./neuralclaw run --task "Summarize all Go files in ./internal" --scope "codebase"

# Ingest a PDF or image into long-term memory
./neuralclaw ingest ocr --input ./docs/architecture.pdf --scope "project:docs"

# Query the hybrid memory store
./neuralclaw memory query --scope "project:docs" -q "What is the embedding architecture?"

# Query with explainable score breakdown
./neuralclaw memory query --scope "project:docs" -q "DMN consolidation" --explain

# List all known memory scopes
./neuralclaw scope list

# Trigger DMN background reflection manually
./neuralclaw dmn run --scope "project:docs" --date 2026-03-05

# Run DMN continuously on a schedule (interval in minutes)
./neuralclaw dmn schedule --interval 60 --scope "project:docs"

# Prune expired memories
./neuralclaw memory reap --scope "project:docs"

# Evaluate retrieval quality against golden queries
./neuralclaw eval retrieval --golden ./eval/golden.yaml --k 10

# Write evaluation results to a JSON report
./neuralclaw eval retrieval --golden ./eval/golden.yaml --k 10 --output ./eval/report.json
```

---

## 🏗️ Architecture

```text
neuralclaw/
├── cmd/zclaw/              ← CLI entry point (Cobra)
├── configs/                ← YAML configuration
├── docs/                   ← WEB_GUI.md, USAGE.md, RETRIEVAL_EVAL.md
├── internal/
│   ├── agent/
│   │   ├── llm/            ← OpenAI-compatible provider + token types
│   │   ├── dispatch.go     ← ReAct agent loop (instrumented with token tracking)
│   │   └── tools.go        ← Native Tool interface (ShellRunner, FileReader)
│   ├── cmd/                ← CLI subcommand handlers (memory, eval, web, dmn, ...)
│   ├── config/             ← Viper configuration structs + retention policies
│   ├── dmn/                ← Background reflection pipeline + evidence backlinks
│   ├── eval/               ← Retrieval evaluation (Recall@K, MRR, NDCG)
│   ├── ingest/
│   │   └── ocr_glm/        ← Native GLM-OCR HTTP client
│   ├── memory/
│   │   ├── store/          ← Pure Go hybrid store (JSONStore, Retriever, Embedder)
│   │   └── reaper/         ← Tiered retention & memory expiration
│   ├── observability/      ← Zap logger + TokenTracker (JSONL usage analytics)
│   ├── taskstore/          ← JSON-backed task/run persistence
│   └── web/                ← HTMX dashboard, Memory Explorer, Token Dashboard
│       └── templates/      ← Go html/template (layout, dashboard, tokens, context...)
└── pkg/types/              ← Shared domain types (MemoryItem, ExplainedHit, ScoreBreakdown)
```

---

## 🌐 Web API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/web` | Dashboard with stats, system ops, and Memory Evidence Explorer |
| GET | `/web/tasks` | Task queue management |
| GET | `/web/runs` | Execution run history |
| GET | `/web/runs/{id}` | Run detail with live SSE log streaming |
| GET | `/web/tokens` | Token usage analytics |
| GET | `/web/context` | Context file browser |
| POST | `/api/tasks` | Create a new task |
| POST | `/api/tasks/{id}/dispatch` | Dispatch a task to the agent |
| GET | `/api/runs/{id}` | Run JSON detail |
| GET | `/api/runs/{id}/events` | SSE event stream for a run |
| GET | `/api/memory/{id}/explain` | Score breakdown for a memory item |
| GET | `/api/memory/{id}/evidence` | Recursive evidence chain for a memory item |
| GET | `/api/context/files` | Project file listing with token estimates |

---

## ⚙️ Configuration (`configs/config.yaml`)

```yaml
agent:
  provider: "openrouter"
  base_url: "https://openrouter.ai/api/v1"
  api_key: "sk-or-v1-..."
  model: "openai/gpt-4o-mini"

ingest:
  ocr:
    endpoint: "https://open.bigmodel.cn/api/paas/v4/layout_parsing"
    api_key: "..."
    model: "glm-ocr"

memory:
  db_path: "./data/memory"
  embedding:
    provider: "jina"
    api_key: "jina_..."
    base_url: "https://api.jina.ai/v1"
    model: "jina-embeddings-v3"
    dimensions: 1024
  retrieval:
    vector_weight: 0.7
    bm25_weight: 0.3
    time_decay_half_life_days: 30
    candidate_pool_size: 50
    hard_min_score: 0.01
    rerank_provider: "jina"
    rerank_api_key: "jina_..."
    rerank_endpoint: "https://api.jina.ai/v1/rerank"
    rerank_model: "jina-reranker-v2-base-multilingual"

retention:
  raw_days: 90
  daily_summary_days: 730
  weekly_summary_days: 1825
  monthly_summary_days: 3650
  concept_edges_days: 1825
  anomalies_days: 730

web:
  addr: "127.0.0.1:8080"
  auth_token: ""
```

---

## 💡 Design Philosophy

1. **Zero Compromise on Simplicity**: One Go binary. No Docker. No virtual environments. Copy it to your Raspberry Pi and run.
2. **Local-first, Cloud-optional**: Your memories stay on disk as human-readable JSON. The only external calls are to the LLM and embedding APIs you chose.
3. **DMN as a First-Class Citizen**: Agents forget. NeuralClaw doesn't. Background consolidation is not a plugin—it's the core loop.
4. **Explainability by Default**: Every retrieval result can expose its full scoring pipeline. Evidence chains let you trace any summary back to its raw source memories.

---

## 📚 Documentation

- [USAGE.md](docs/USAGE.md) — Full CLI command reference
- [WEB_GUI.md](docs/WEB_GUI.md) — Web dashboard pages, Explain/Evidence API, and authentication
- [RETRIEVAL_EVAL.md](docs/RETRIEVAL_EVAL.md) — Evaluation methodology, golden query format, and metrics

---

## 📄 License

[Apache License 2.0](LICENSE)
