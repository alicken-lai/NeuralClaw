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

### Task Queue (`/web/tasks`)
- View all queued tasks with status indicators
- Create new tasks via a modal form
- Dispatch tasks to the NeuralClaw agent

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
