# interview-levelup-backend

> Go REST API & SSE streaming backend for an AI-powered mock interview platform — handles auth, interview lifecycle, answer evaluation, and real-time question streaming.

Go/Gin REST API that orchestrates users, interview sessions, and real-time streaming between the frontend and the [interview-levelup-agent](https://github.com/interview-levelup/interview-levelup-agent) LangGraph AI service.

## Stack

| Layer | Tech |
|---|---|
| Language | Go 1.25 |
| Framework | Gin |
| Database | PostgreSQL 16 (sqlx) |
| Auth | JWT HS256 |
| Speech-to-text | OpenAI Whisper (direct HTTP) |
| AI Agent | [interview-levelup-agent](https://github.com/interview-levelup/interview-levelup-agent) (FastAPI + LangGraph) |
| Infra | Docker Compose |

## Project Layout

```
interview-levelup-backend/
├── cmd/server/main.go          # entrypoint & wiring
├── internal/
│   ├── config/                 # env config
│   ├── database/               # sqlx connection pool
│   ├── models/                 # DB structs
│   ├── repository/             # SQL queries
│   ├── services/               # business logic, agent client, whisper client
│   ├── middleware/             # JWT auth middleware
│   ├── handlers/               # Gin HTTP handlers
│   └── router/                 # route registration
├── migrations/001_init.sql     # schema, auto-applied on first run
├── Dockerfile
├── docker-compose.yml
└── .env.example
```

## Quickstart

```bash
cp .env.example .env
# Fill in: JWT_SECRET, DB_*, AGENT_BASE_URL, WHISPER_API_KEY

docker compose up --build -d
# Backend → http://localhost:8080
```

> The AI agent runs independently. Set `AGENT_BASE_URL` in `.env` to point to wherever it is running (e.g. `http://host.docker.internal:8000` when running locally on the host).

## Environment Variables

| Variable | Description |
|---|---|
| `JWT_SECRET` | Secret key for signing JWT tokens |
| `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` | PostgreSQL connection |
| `DB_SSLMODE` | PostgreSQL SSL mode (e.g. `disable`) |
| `AGENT_BASE_URL` | Base URL of the AI agent service |
| `WHISPER_API_KEY` | API key for OpenAI Whisper |
| `WHISPER_BASE_URL` | Whisper base URL (leave empty for OpenAI default) |
| `BACKEND_PORT` | Host port to expose the backend on (default: `8080`) |

## API Reference

### Auth

| Method | Path | Body |
|---|---|---|
| `POST` | `/api/v1/auth/register` | `{"email", "password"}` |
| `POST` | `/api/v1/auth/login` | `{"email", "password"}` → `{"token"}` |
| `POST` | `/api/v1/auth/change-password` | `{"current_password", "new_password"}` |

### Interviews *(Bearer token required)*

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/interviews` | Create interview (blocking) |
| `POST` | `/api/v1/interviews/stream` | Create + stream first question via SSE |
| `GET` | `/api/v1/interviews` | List user's interviews |
| `GET` | `/api/v1/interviews/:id` | Fetch interview + all rounds |
| `POST` | `/api/v1/interviews/:id/answer` | Submit answer (blocking) |
| `POST` | `/api/v1/interviews/:id/answer/stream` | Submit answer, stream next question via SSE |
| `POST` | `/api/v1/interviews/:id/transcribe` | Transcribe audio via Whisper |

#### Create interview request body

```json
{
  "role": "product manager",
  "level": "junior",
  "style": "standard",
  "max_rounds": 5
}
```

#### SSE event types (`/interviews/stream`, `/answer/stream`)

```
data: {"type":"created", "interview":{...}}   // interview row ready, client can navigate
data: {"type":"token",   "content":"..."}     // LLM token
data: {"type":"done",    "round":{...}}       // round persisted, stream ends
data: {"type":"error",   "message":"..."}     // on failure
```

## Local Development (without Docker)

```bash
# Start Postgres locally, then:
export $(grep -v '^#' .env | xargs)
go run ./cmd/server
```

