# interview-levelup-backend

> **GitHub description:** Go REST API & SSE streaming backend for an AI-powered mock interview platform — handles auth, interview lifecycle, answer evaluation, and real-time question streaming.

Go/Gin REST API that orchestrates users, interview sessions, and real-time streaming between the frontend and the [interview-levelup-agent](../interview-levelup-agent) LangGraph AI service.

## Stack

| Layer | Tech |
|---|---|
| Language | Go 1.22 |
| Framework | Gin |
| Database | PostgreSQL 16 (sqlx) |
| Auth | JWT HS256 |
| Speech-to-text | OpenAI Whisper (direct HTTP) |
| AI Agent | [interview-levelup-agent](../interview-levelup-agent) (FastAPI + LangGraph) |
| Infra | Docker Compose |

## Features

- **JWT auth** — register / login, all interview routes protected
- **Interview lifecycle** — create, list, fetch with all rounds
- **SSE streaming** — `POST /interviews/stream` emits a `created` event the moment the DB row exists, then proxies LLM tokens in real-time, so the frontend can navigate immediately and show text as it arrives
- **Answer submission streaming** — `POST /interviews/:id/answer/stream` evaluates the answer and streams the next question token-by-token
- **Whisper transcription** — `POST /interviews/:id/transcribe` accepts audio and calls OpenAI Whisper directly from Go (no Python hop)
- **Final report** — agent returns a structured debrief stored on the interview row

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
# Fill in: JWT_SECRET, DB_*, LLM_API_KEY, AGENT_BASE_URL

docker compose up --build
# Backend → http://localhost:8080
# Agent   → http://localhost:8000  (started by compose)
```

## API Reference

### Auth

| Method | Path | Body |
|---|---|---|
| `POST` | `/api/v1/auth/register` | `{"email", "password"}` |
| `POST` | `/api/v1/auth/login` | `{"email", "password"}` → `{"token"}` |

### Interviews *(Bearer token required)*

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/interviews` | Create interview (blocking) |
| `POST` | `/api/v1/interviews/stream` | Create + stream first question via SSE |
| `GET` | `/api/v1/interviews` | List user's interviews |
| `GET` | `/api/v1/interviews/:id` | Fetch interview + all rounds |
| `POST` | `/api/v1/interviews/:id/answer` | Submit answer (blocking) |
| `POST` | `/api/v1/interviews/:id/answer/stream` | Submit answer, stream next question |
| `POST` | `/api/v1/interviews/:id/transcribe` | Transcribe audio via Whisper |

#### Create interview body

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
data: {"type":"created", "interview":{...}}   // interview row ready
data: {"type":"token",   "content":"..."}     // LLM token
data: {"type":"done",    "round":{...}}       // round persisted, stream ends
data: {"type":"error",   "message":"..."}     // on failure
```

## Local Development (without Docker)

```bash
# Start Postgres, then:
export $(grep -v '^#' .env | xargs)
go run ./cmd/server
```

Run the agent separately:

```bash
cd ../interview-levelup-agent
pip install -r requirements.txt
uvicorn main:app --reload --port 8000
```


## Stack

| Layer | Tech |
|---|---|
| API | Go 1.22 + Gin |
| Database | PostgreSQL 16 |
| Auth | JWT (HS256) |
| AI Agent | Python FastAPI + LangGraph |
| Infra | Docker Compose |

## Project Layout

```
interview-levelup-backend/
├── cmd/server/main.go          # entrypoint
├── internal/
│   ├── config/                 # env config
│   ├── database/               # sqlx connection
│   ├── models/                 # DB models
│   ├── repository/             # SQL queries
│   ├── services/               # business logic + agent client
│   ├── middleware/             # JWT middleware
│   ├── handlers/               # Gin handlers
│   └── router/                 # route registration
├── migrations/001_init.sql     # schema auto-applied on first run
├── Dockerfile
├── docker-compose.yml
└── .env.example
```

## Quickstart

```bash
# 1. copy and fill in env vars
cp .env.example .env
# edit .env: set JWT_SECRET, DB_* and LLM_* values

# 2. start everything
docker compose up --build
```

The backend is available at `http://localhost:8080`.

## API

### Auth

| Method | Path | Body |
|---|---|---|
| `POST` | `/api/v1/auth/register` | `{"email","password"}` |
| `POST` | `/api/v1/auth/login` | `{"email","password"}` → returns `token` |

### Interviews (Bearer token required)

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/interviews` | Start a new interview |
| `GET` | `/api/v1/interviews` | List your interviews |
| `GET` | `/api/v1/interviews/:id` | Get interview + rounds |
| `POST` | `/api/v1/interviews/:id/answer` | Submit answer, get next question or final report |

#### Start interview body

```json
{
  "role": "product manager",
  "level": "junior",
  "style": "standard",
  "max_rounds": 5
}
```

#### Submit answer body

```json
{ "answer": "I would prioritize by impact and effort..." }
```

## Development (without Docker)

```bash
# start postgres locally, then:
export $(cat .env | xargs)
export AGENT_BASE_URL=http://localhost:8000  # agent running locally
go run ./cmd/server
```

Run the agent separately:

```bash
cd ../interview-levelup-agent
pip install -r requirements.txt
uvicorn main:app --reload
```
