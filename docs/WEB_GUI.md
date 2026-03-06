# NeuralClaw Web GUI

The NeuralClaw architecture includes a built-in Web GUI with a premium dark glassmorphic design for task dispatching, analytics, and file browsing.

## Running the Web GUI

To start the server, use the `web` subcommand. You must provide a target scope that this specific dashboard will manage. Multi-scope isolation prevents a user on one dashboard from seeing or executing tasks in another scope.

```sh
# Start the web UI on port 8080 targeting the "project:research" scope.
./neuralclaw web --addr 127.0.0.1:8080 --scope project:research
```

If `--addr` is omitted, it will fall back to the `web.addr` configured in `config.yaml` or default to `127.0.0.1:8080`.

## Pages

### Dashboard (`/web`)
The main workspace overview showing:
- **Total Queue**: Number of queued tasks
- **Agent Runs**: Total execution runs
- **DMN Status**: Background consolidation status
- **System Operations**: Force DMN Reflection and Execute Reaper buttons
- **Memory Evidence Explorer**: Recent memory list with `Explain` and `Evidence` buttons
- **Security Summary**: Pending approvals, quarantined items, and recent blocked prompts

### Task Queue (`/web/tasks`)
- View all queued tasks with status indicators
- Create new tasks via a modal form
- Dispatch tasks to the NeuralClaw agent
- Shows `blocked` and `pending_approval` task states with security reasons

### Execution Runs (`/web/runs`)
- Full history of agent execution runs
- Live status tracking with animation
- Click into a run for detailed SSE log streaming

### Token Usage (`/web/tokens`)
- **Summary Cards**: Total Input / Output / Total tokens consumed (last 5 days)
- **Daily Usage Table**: Token consumption grouped by date and LLM model
- **Source Breakdown**: Aggregated usage by source (TaskRun, Cron:DMN, Ingest)
- Data is persisted to `logs/token_usage.jsonl` and loaded on startup

### Context Browser (`/web/context`)
- Browse the project directory structure
- View file sizes in human-readable format
- Estimated LLM context footprint per file (~4 bytes per token heuristic)
- Useful for auditing which files consume the most context

### Security Overview (`/web/security`)
- Summary cards for pending approvals, quarantined items, and recent blocked prompts
- Entry point into approval, quarantine, and audit views

### Security Approvals (`/web/security/approvals`)
- Lists pending / approved / rejected approval requests
- Shows request time, kind, status, and reason
- Current patch uses CLI to approve or reject requests

### Security Quarantine (`/web/security/quarantine`)
- Lists quarantined memory items withheld from the main memory store
- Shows risk level, source, reasons, and a content preview

### Security Events (`/web/security/events`)
- Recent audit log entries table
- Includes time, event type, risk level, actor, and summary

## Memory Explain / Evidence API

The dashboard uses HTMX actions to call two memory detail APIs:

- `GET /api/memory/{id}/explain`
  - Returns `[]types.ExplainedHit` JSON
  - Includes score components (`vector_score`, `bm25_score`, `time_boost`, `access_boost`, `final_score`)
- `GET /api/memory/{id}/evidence`
  - Returns a recursive evidence-chain JSON document
  - Expands both `derived_from` and `evidence_of` links for the target memory

Both endpoints are scope-isolated by the Web server middleware; items outside the active scope return `404`.

## Security Visibility And Behavior

When the Security Guard layer is enabled:

- Task creation is inspected by the prompt firewall before execution
- High-risk tasks may be marked as `pending_approval`
- Critical tasks may be marked as `blocked`
- Security events are visible in `/web/security/events`
- Quarantined OCR / DMN memory writes are visible in `/web/security/quarantine`

If a task requires approval, the Web UI shows the pending state and leaves manual approval to the CLI:

```sh
./neuralclaw security approvals list --scope project:research
./neuralclaw security approvals approve --id <approval-id>
./neuralclaw security approvals reject --id <approval-id>
```

## Security & Authentication

The Web GUI is designed primarily for local development or secure internal networks. However, you can enable a simple token-based Auth Guard.

In your `config.yaml`:
```yaml
web:
  addr: "127.0.0.1:8080"
  auth_token: "super-secret-dev-token"
```

When `auth_token` is set, all requests must include the token either:
1. In the HTTP headers: `X-Auth-Token: super-secret-dev-token`
2. As a URL query parameter: `http://127.0.0.1:8080/web?token=super-secret-dev-token`

## Design

The UI uses a dark glassmorphic aesthetic with:
- **Tailwind CSS** and **HTMX** loaded via CDN (zero build step)
- **Go `html/template`** with embedded filesystem (`//go:embed templates/*`)
- Custom gradient animations, ambient glow effects, and Lucide-style SVG icons
- Responsive sidebar navigation with "Analytics" section for Token Usage and Context Browser
