# GitHub App Workflow Manager

A standalone GitHub App built in Go that provides a web-based UI to manage GitHub Actions workflows. No frameworks — just Go's standard library, htmx for interactivity, and a few essential dependencies.

Inspired by [Terrateam](https://github.com/terrateamio/terrateam)'s architecture: a self-contained HTTP server that receives webhooks directly, verifies signatures, and manages state.

## Features

- **Dashboard** — View all repositories where the app is installed
- **Workflow Management** — List, trigger, and monitor GitHub Actions workflows
- **Run Monitoring** — Real-time status updates via Server-Sent Events (SSE)
- **Run Actions** — Re-run all jobs, re-run failed jobs, or cancel running workflows
- **Log Viewer** — View workflow run logs with job tabs, step grouping, and line numbers
- **Webhook Processing** — Automatic installation sync, run/job state tracking
- **Single Binary** — All templates, CSS, and JS embedded via `go:embed`

## Architecture

```
GitHub ──(webhook)──> Go HTTP Server (:8080)
                           │
                           ├── Verify HMAC-SHA256 signature
                           ├── Parse event, persist to SQLite
                           ├── Broadcast via SSE broker
                           │
                           └── Web UI (htmx + Alpine.js + Tailwind CSS)
                                  │
                                  ├── Dashboard (repo grid)
                                  ├── Repo detail (workflows + runs)
                                  ├── Run detail (jobs, status, actions)
                                  └── Log viewer (monospace, per-job tabs)
```

### Tech Stack

| Component | Technology | Why |
|-----------|-----------|-----|
| Language | Go 1.22+ | Strong stdlib, single binary, great for HTTP servers |
| HTTP Router | `net/http` (stdlib) | Go 1.22 added method + path param routing |
| Templates | `html/template` (stdlib) | Server-rendered HTML, embedded in binary |
| Interactivity | htmx 2.x | AJAX, SSE, partial page updates — no JS framework |
| Client state | Alpine.js 3.x | Dropdowns, modals, tabs — 15KB |
| Styling | Tailwind CSS (CDN) | Utility-first CSS, zero build step |
| Database | SQLite via `modernc.org/sqlite` | Pure Go, no CGo, embedded |
| GitHub API | `google/go-github` v68 | Typed client for Actions API |
| GitHub Auth | `bradleyfalzon/ghinstallation` | JWT + installation token management |
| Real-time | Server-Sent Events | Unidirectional push, htmx SSE extension |
| Logging | `log/slog` (stdlib) | Structured logging |

### Dependencies (3 direct)

```
github.com/google/go-github/v68
github.com/bradleyfalzon/ghinstallation/v2
modernc.org/sqlite
```

Everything else uses the Go standard library.

## Getting Started

### Prerequisites

- Go 1.22 or later
- A GitHub account

### 1. Create a GitHub App

1. Go to **Settings > Developer settings > GitHub Apps > New GitHub App**
2. Set the following:
   - **GitHub App name**: Choose a unique name
   - **Homepage URL**: `http://localhost:8080`
   - **Webhook URL**: Your public URL (see step 3 below)
   - **Webhook secret**: Generate a strong secret (`openssl rand -hex 32`)
3. Set **Permissions**:
   - Repository permissions:
     - **Actions**: Read and write
     - **Contents**: Read-only
     - **Metadata**: Read-only
   - Subscribe to events:
     - `Workflow run`
     - `Workflow job`
     - `Installation`
     - `Installation repositories`
4. After creating, note the **App ID** from the app's settings page
5. Generate a **private key** and save it as `private-key.pem` in the project root

### 2. Configure

Copy the example env file and fill in your values:

```bash
cp .env.example .env
```

Edit `.env`:

```
GITHUB_APP_ID=123456
GITHUB_PRIVATE_KEY_PATH=./private-key.pem
GITHUB_WEBHOOK_SECRET=your-webhook-secret
PORT=8080
DATABASE_PATH=./data/ghapp.db
DEV_MODE=true
```

### 3. Expose Webhooks (Local Development)

GitHub needs to reach your local server. Use [smee.io](https://smee.io/) or [ngrok](https://ngrok.com/):

**Option A: smee.io (free, no signup)**

```bash
# Install smee client
npm install -g smee-client

# Create a channel at https://smee.io and copy the URL
# Set that URL as your GitHub App's webhook URL

# Forward webhooks to your local server
smee -u https://smee.io/YOUR_CHANNEL_ID -t http://localhost:8080/api/webhooks/github
```

**Option B: ngrok**

```bash
ngrok http 8080
# Use the ngrok URL as your GitHub App's webhook URL
```

### 4. Install the App

1. Go to your GitHub App's settings page
2. Click **Install App** in the sidebar
3. Select your account/organization
4. Choose which repositories to install on

### 5. Run

```bash
# Development mode (hot-reload templates)
make dev

# Or directly
DEV_MODE=true go run .
```

Open [http://localhost:8080](http://localhost:8080) — you should see your repositories on the dashboard.

### 6. Build for Production

```bash
# Build a single static binary
make build

# Run it
./bin/ghapp
```

## Project Structure

```
lab-githubapp/
├── main.go                         # Entry point: config, DI, graceful shutdown
├── embed.go                        # go:embed directives for templates and static files
├── internal/
│   ├── config/config.go            # Environment variable loading and validation
│   ├── github/
│   │   ├── auth.go                 # GitHub App JWT + installation token management
│   │   ├── client.go               # Client factory (per-installation)
│   │   ├── workflows.go            # Actions API: list, dispatch, rerun, cancel
│   │   ├── logs.go                 # Download and extract workflow run log zips
│   │   └── installations.go        # Sync installations and repos from GitHub
│   ├── webhook/
│   │   ├── handler.go              # HTTP handler for POST /api/webhooks/github
│   │   ├── verify.go               # HMAC-SHA256 signature verification
│   │   └── events.go               # Event routing and processing
│   ├── store/
│   │   ├── store.go                # Store interface
│   │   ├── sqlite.go               # SQLite implementation with embedded migrations
│   │   └── models.go               # Domain models
│   ├── server/
│   │   ├── server.go               # HTTP server with graceful shutdown
│   │   ├── routes.go               # All route registrations
│   │   └── middleware.go           # Logging, recovery, request ID
│   ├── handler/
│   │   ├── dashboard.go            # GET / (dashboard)
│   │   ├── repos.go                # GET /repos/{owner}/{repo}
│   │   ├── workflows.go            # Workflow list + dispatch
│   │   ├── runs.go                 # Run list, detail, rerun, cancel
│   │   ├── logs.go                 # Log viewer
│   │   └── sse.go                  # SSE stream endpoints
│   ├── sse/
│   │   ├── broker.go               # Fan-out pub/sub SSE broker
│   │   └── event.go                # SSE event formatting
│   └── templates/
│       └── engine.go               # Template engine with helper functions
├── web/
│   ├── templates/                  # Go HTML templates
│   │   ├── layouts/base.html       # Base layout with nav, scripts, footer
│   │   ├── pages/                  # Full page templates
│   │   └── partials/               # htmx swap targets and components
│   └── static/                     # Vendored JS (htmx, Alpine.js), CSS
├── .env.example
├── Makefile
├── Dockerfile
└── go.mod
```

## Routes

### Webhook

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/webhooks/github` | Receives GitHub webhook events |

### Web UI

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/` | Dashboard — installed repositories |
| `GET` | `/repos/{owner}/{repo}` | Repository detail — workflows and recent runs |
| `GET` | `/repos/{owner}/{repo}/workflows` | Workflow list |
| `POST` | `/repos/{owner}/{repo}/workflows/{id}/dispatch` | Trigger workflow dispatch |
| `GET` | `/repos/{owner}/{repo}/runs` | Workflow runs (filterable by status) |
| `GET` | `/repos/{owner}/{repo}/runs/{runID}` | Run detail — jobs, steps, actions |
| `POST` | `/repos/{owner}/{repo}/runs/{runID}/rerun` | Re-run all jobs |
| `POST` | `/repos/{owner}/{repo}/runs/{runID}/rerun-failed` | Re-run failed jobs only |
| `POST` | `/repos/{owner}/{repo}/runs/{runID}/cancel` | Cancel a running workflow |
| `GET` | `/repos/{owner}/{repo}/runs/{runID}/logs` | View workflow run logs |
| `GET` | `/repos/{owner}/{repo}/jobs/{jobID}/logs` | Get raw job logs |

### SSE Streams

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/sse/repos/{owner}/{repo}` | Real-time run status for a repo |
| `GET` | `/api/sse/repos/{owner}/{repo}/runs/{runID}` | Real-time job updates for a run |

## How It Works

This is a control panel for GitHub Actions. Once you install it as a GitHub App on
some repositories, it shows a dashboard of those repos, lists each repo's workflows
and lets you trigger them, tracks workflow runs and their jobs/steps in real time, lets
you re-run (all or failed-only) and cancel runs, and displays run logs.

It's a single Go binary with **no web framework** — just `net/http`, server-rendered
HTML templates, and htmx for interactivity. Everything (templates, CSS, JS) is embedded
into the binary via `go:embed`, and state lives in an embedded SQLite database
(`modernc.org/sqlite`, pure Go, no CGo).

There are two ways data flows through the app: **GitHub pushes webhooks to it**, and
**the user clicks things in the browser** which call the GitHub API outward.

### Startup & Wiring (`main.go`)

`run()` wires everything together with manual dependency injection:

1. Load and validate config from the environment (`internal/config`)
2. Open and migrate the SQLite database
3. Build the GitHub auth provider and API client
4. Start the SSE broker goroutine
5. Kick off a one-time `SyncInstallations` in the background, so the app discovers
   existing installations even without waiting for a fresh webhook
6. Load templates (from disk when `DEV_MODE=true`, otherwise from the embedded FS)
7. Start the HTTP server

It listens for `SIGINT`/`SIGTERM` and shuts down gracefully with a 10-second timeout.

### GitHub App Authentication (`internal/github/auth.go`)

The app uses the standard two-tier GitHub App authentication flow:

1. **App-level**: An RSA private key signs a short-lived JWT (RS256, 10-minute expiry)
   that identifies the app itself — used for app-level calls like listing installations.
2. **Installation-level**: That JWT is exchanged for a 1-hour installation access token
   (auto-refreshed), scoped to the specific repositories where the app is installed.

This is handled by `bradleyfalzon/ghinstallation`, which is plugged in as an
`http.RoundTripper`, so token minting and refresh happen transparently on every API
call. `AuthProvider` caches one transport per installation ID (with double-checked
locking) so tokens get reused across requests.

### Inbound Flow: Webhooks (`internal/webhook/`)

When something happens on GitHub (a run starts, a job finishes, the app is installed
somewhere):

1. GitHub sends a webhook event to `POST /api/webhooks/github`
2. The server verifies the `X-Hub-Signature-256` HMAC-SHA256 signature against your
   webhook secret (`verify.go`)
3. `EventProcessor.Process` routes on the `X-GitHub-Event` header. It handles four
   event types (`events.go`):
   - `installation` — upsert/delete the installation and its repos
   - `installation_repositories` — add/remove individual repos
   - `workflow_run` — upsert the run record
   - `workflow_job` — upsert the job record (steps stored as JSON)
4. State is persisted to SQLite (installations, repos, runs, jobs)
5. For run/job events, an SSE event is published to a topic like `repo:owner/name`
6. htmx picks up the SSE event and swaps the updated HTML partial into the page

### Real-time Updates (`internal/sse/broker.go`)

The SSE broker is a goroutine-based pub/sub fan-out. A single `Run` loop owns a
`map[topic] → set of client channels` and serializes register/unregister/broadcast over
channels, so there are no data races.

- Each browser tab subscribes to a topic (e.g., `repo:owner/name`)
- When a webhook arrives, the event processor publishes to the broker
- The broker fans out to all subscribed clients
- htmx's SSE extension swaps the partial into the matching DOM element

When broadcasting, if a client's channel buffer is full the event is **dropped for that
client** (a non-blocking `select` with a `default` case) rather than blocking everyone —
so a slow browser can't stall the broker.

No polling. No WebSocket complexity. Just standard HTTP streaming.

### Outbound Flow: The Web UI (`internal/server`, `internal/handler`)

Routes use Go 1.22+ method + path pattern routing (e.g.
`GET /repos/{owner}/{repo}/runs/{runID}`). Handlers render server-side HTML templates.
The action endpoints — dispatch, rerun, rerun-failed, cancel — call the GitHub Actions
API outward using the per-installation client. Two SSE endpoints (`/api/sse/...`) let a
repo page or a run page subscribe to live updates.

Pages are full HTML; htmx fetches partials (`web/templates/partials/`) to swap pieces of
the page without a full reload, and Alpine.js handles small client-side bits like
dropdowns and tabs.

## Docker

```bash
# Build image
make docker

# Run
docker run --rm -p 8080:8080 \
  -v ./private-key.pem:/app/private-key.pem:ro \
  -v ./data:/app/data \
  -e GITHUB_APP_ID=123456 \
  -e GITHUB_PRIVATE_KEY_PATH=/app/private-key.pem \
  -e GITHUB_WEBHOOK_SECRET=your-secret \
  ghapp
```

The Docker image is ~25MB (Alpine + single Go binary). All templates and static assets are embedded in the binary.

## Development

```bash
# Run in dev mode (hot-reload templates from disk)
make dev

# Run tests
make test

# Format code
make fmt

# Lint
make vet
```

In dev mode (`DEV_MODE=true`), templates are re-parsed from disk on every request — no restart needed when editing HTML.

## License

MIT
