# NeuralClaw Usage

The `neuralclaw` CLI coordinates the agent runtime, memory injection, retrieval, and background DMN consolidation workflows.

## Prerequisites

- Go 1.22+
- (Optional) running instances of `memory-Native JSONStore-pro` and `GLM-OCR`.

## Building

```sh
sh scripts/dev.sh
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
Run a hybrid retrieval query scoped to a specific context.

```sh
./neuralclaw memory query --scope "project:zero" --q "How does DMN work?"
```

### 3. Agent Execution
Trigger a NeuralClaw agent run.

```sh
./neuralclaw run --task "Analyze the latest docs."
```

### 4. DMN (Daily Reflection)
Force the DMN module to consolidate memories for a specific date and scope.

```sh
./neuralclaw dmn run --scope "project:zero" --date "2024-05-10"
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
Simulate memory eviction in a specific scope safely:
```sh
./neuralclaw memory reap --scope "project:zero" --dry-run
```

**Run Reaper (Execute):**
Performs the actual eviction. *It is recommended to run this daily via cron (e.g. at 03:00) against your active scopes.*
```sh
./neuralclaw memory reap --scope "project:zero"
```
You can simulate a run for a future date using the `--now` flag, e.g. `--now "2026-05-10T15:00:00Z"`.

### 7. Web GUI (Task Dispatch)
NeuralClaw provides a built-in, lightweight web UI for dispatching tasks and streaming real-time logs.

```sh
# Start the dashboard for a specific scope
./neuralclaw web --addr 127.0.0.1:8080 --scope project:research
```
Open `http://127.0.0.1:8080/web` in your browser. 

For more details on authentication and features, see [WEB_GUI.md](./WEB_GUI.md).
