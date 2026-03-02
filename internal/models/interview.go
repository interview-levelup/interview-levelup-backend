package models

import "time"

type Interview struct {
	ID          string    `db:"id"           json:"id"`
	UserID      string    `db:"user_id"      json:"user_id"`
	ThreadID    string    `db:"thread_id"    json:"thread_id"`
	Role        string    `db:"role"         json:"role"`
	Level       string    `db:"level"        json:"level"`
	Style       string    `db:"style"        json:"style"`
	MaxRounds   int       `db:"max_rounds"   json:"max_rounds"`
	Status      string    `db:"status"       json:"status"`
	FinalReport *string   `db:"final_report" json:"final_report,omitempty"`
	CreatedAt   time.Time `db:"created_at"   json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"   json:"updated_at"`
}

type InterviewRound struct {
	ID               string    `db:"id"                json:"id"`
	InterviewID      string    `db:"interview_id"      json:"interview_id"`
	RoundNum         int       `db:"round_num"         json:"round_num"`
	Question         string    `db:"question"          json:"question"`
	Answer           *string   `db:"answer"            json:"answer,omitempty"`
	Score            *float64  `db:"score"             json:"score,omitempty"`
	EvaluationDetail *string   `db:"evaluation_detail" json:"evaluation_detail,omitempty"`
	IsFollowup       bool      `db:"is_followup"       json:"is_followup"`
	CreatedAt        time.Time `db:"created_at"        json:"created_at"`
}
