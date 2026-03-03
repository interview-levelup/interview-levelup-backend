package models

import "time"

type Interview struct {
	ID             string    `db:"id"             json:"id"`
	UserID         string    `db:"user_id"        json:"user_id"`
	Role           string    `db:"role"           json:"role"`
	Level          string    `db:"level"          json:"level"`
	Style          string    `db:"style"          json:"style"`
	MaxRounds      int       `db:"max_rounds"     json:"max_rounds"`
	Status         string    `db:"status"         json:"status"`
	FinalReport    *string   `db:"final_report"   json:"final_report,omitempty"`
	CreatedAt      time.Time `db:"created_at"     json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"     json:"updated_at"`
	// Computed by list query: answered main rounds (excludes followups and sub interactions)
	AnsweredRounds int       `db:"answered_rounds" json:"answered_rounds"`
}

type InterviewRound struct {
	ID               string     `db:"id"                json:"id"`
	InterviewID      string     `db:"interview_id"      json:"interview_id"`
	RoundNum         int        `db:"round_num"         json:"round_num"`
	Question         string     `db:"question"          json:"question"`
	Answer           *string    `db:"answer"            json:"answer,omitempty"`
	Score            *float64   `db:"score"             json:"score,omitempty"`
	EvaluationDetail *string    `db:"evaluation_detail" json:"evaluation_detail,omitempty"`
	IsFollowup       bool       `db:"is_followup"       json:"is_followup"`
	IsSub            bool       `db:"is_sub"             json:"is_sub"`
	CreatedAt        time.Time  `db:"created_at"        json:"created_at"`
	AnsweredAt       *time.Time `db:"answered_at"       json:"answered_at,omitempty"`
}
