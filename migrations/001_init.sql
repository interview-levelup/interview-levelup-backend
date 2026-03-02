-- migrations/001_init.sql
-- Run automatically via docker-compose init script on first start.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS interviews (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role         TEXT NOT NULL,
    level        TEXT NOT NULL DEFAULT 'junior',
    style        TEXT NOT NULL DEFAULT 'standard',
    max_rounds   INTEGER NOT NULL DEFAULT 5,
    status       TEXT NOT NULL DEFAULT 'ongoing',  -- ongoing | finished | aborted | ended
    final_report TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_interviews_user_id ON interviews(user_id);

CREATE TABLE IF NOT EXISTS interview_rounds (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    interview_id      UUID NOT NULL REFERENCES interviews(id) ON DELETE CASCADE,
    round_num         INTEGER NOT NULL DEFAULT 0,
    question          TEXT NOT NULL,
    answer            TEXT,
    score             FLOAT,
    evaluation_detail TEXT,
    is_followup       BOOLEAN NOT NULL DEFAULT FALSE,
    is_sub            BOOLEAN NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    answered_at       TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_interview_rounds_interview_id ON interview_rounds(interview_id);
