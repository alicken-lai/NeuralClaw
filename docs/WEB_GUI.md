# ZeroClaw Web GUI

The ZeroClaw architecture includes a lightweight, built-in Web GUI designed for task dispatching and simple management without the need for a complex frontend build step.

## Running the Web GUI

To start the server, use the `web` subcommand. You must provide a target scope that this specific dashboard will manage. Multi-scope isolation prevents a user on one dashboard from seeing or executing tasks in another scope.

```sh
# Start the web UI explicitly on port 8080 targeting the "project:research" scope.
./bin/zclaw web --addr 127.0.0.1:8080 --scope project:research
```

If `--addr` is omitted, it will fall back to the `web.addr` configured in `config.yaml` or default to `127.0.0.1:8080`.

## Features
- **Task Queue**: View and create actionable tasks for the agent.
- **Task Dispatch**: Manually dispatch a queued task to the ZeroClaw runtime.
- **Execution Runs**: Track the history of task runs.
- **Live Logs (SSE)**: View real-time streaming logs for a running task via Server-Sent Events.

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
