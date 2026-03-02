# interview-levelup-backend

Go/Gin REST API that manages users and interview sessions powered by the [interview-levelup-agent](../interview-levelup-agent) Python service.

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
