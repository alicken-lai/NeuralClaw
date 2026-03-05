# NeuralClaw Usage

The `neuralclaw` CLI coordinates the agent runtime, memory injection, retrieval, background DMN consolidation, and web dashboard.

## Prerequisites

- Go 1.22+
- A valid `configs/config.yaml` with your API keys (OpenRouter, Jina AI, Zhipu)

## Building

```sh
CGO_ENABLED=0 go build -o neuralclaw ./cmd/zclaw
```

This will produce the binary `neuralclaw`.

## Commands

### 1. Ingestion (OCR)
Ingest an image/PDF file using GLM-OCR. Outputs will be embedded and upserted into memory.

```sh
./neuralclaw ingest ocr --input ./sample.pdf
```
*(By default, this writes to the `global` scope.)*

### 2. Querying Memory (Hybrid Search)
Run a hybrid retrieval query scoped to a specific context. Results are ranked by cosine similarity + BM25 keyword match, with Living Memory LTP boost for frequently recalled items.

```sh
./neuralclaw memory query --scope "project:research" -q "How does DMN work?"
```

### 3. Agent Execution
Trigger a NeuralClaw agent run. Token usage is automatically tracked and stored to `logs/token_usage.jsonl`.

```sh
./neuralclaw run --task "Analyze the latest docs."
```

### 4. DMN (Daily Reflection)
Force the DMN module to consolidate memories for a specific date and scope. Generated summaries include `CausalLinks` tracing back to source raw memories.

```sh
./neuralclaw dmn run --scope "project:research" --date "2026-03-05"
```

To run the DMN continuously on a cron schedule:
```sh
./neuralclaw dmn schedule --interval 60
```

### 5. Managing Scopes
List all accessible context scopes or set a default CLI scope constraint:

```sh
./neuralclaw scope list
./neuralclaw scope set my_default_project
```

### 6. Memory Retention and Aging (Reaper)
The system enforces a tiered aging policy based on the `ItemType` to keep token usage efficient over time.

**Default Policies:**
- Raw memory: 90 days
- Daily summary: 730 days (2 years)
- Concept edges: 1825 days (5 years)
- Weekly summary: 1825 days (5 years)
- Monthly summary: 3650 days (10 years)
- Anomalies: 730 days (2 years)

**View Current Policy:**
```sh
./neuralclaw memory policy show
```

**Run Reaper (Dry Run):**
```sh
./neuralclaw memory reap --scope "project:research" --dry-run
```

**Run Reaper (Execute):**
```sh
./neuralclaw memory reap --scope "project:research"
```

### 7. Web GUI (Dashboard + Analytics)
NeuralClaw provides a built-in web UI for task management and token analytics.

```sh
# Start the dashboard for a specific scope
./neuralclaw web --addr 127.0.0.1:8080 --scope project:research
```

Open `http://127.0.0.1:8080/web` in your browser.

**Available Pages:**
| Path | Description |
|------|-------------|
| `/web` | Workspace overview, stats, system operations |
| `/web/tasks` | Task queue — create and dispatch agent tasks |
| `/web/runs` | Execution run history with live SSE log streaming |
| `/web/tokens` | Token Usage Dashboard — daily usage by model, per-source breakdown |
| `/web/context` | Context File Browser — project files with estimated token footprint |

For more details on authentication and features, see [WEB_GUI.md](./WEB_GUI.md).

### 8. Retrieval Evaluation
Evaluate memory retrieval quality against golden query YAML files.

```sh
./neuralclaw eval retrieval --golden ./eval/golden.yaml --k 10
```

Write a machine-readable JSON report:

```sh
./neuralclaw eval retrieval --golden ./eval/golden.yaml --k 10 --output ./eval/report.json
```

Output includes per-query and averaged metrics:
- `Recall@K`
- `MRR@K`
- `NDCG@K`

Golden format and metric details are documented in [RETRIEVAL_EVAL.md](./RETRIEVAL_EVAL.md).
