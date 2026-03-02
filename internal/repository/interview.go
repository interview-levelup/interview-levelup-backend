package repository

import (
	"github.com/fan/interview-levelup-backend/internal/models"
	"github.com/jmoiron/sqlx"
)

type InterviewRepository struct {
	db *sqlx.DB
}

func NewInterviewRepository(db *sqlx.DB) *InterviewRepository {
	return &InterviewRepository{db: db}
}

func (r *InterviewRepository) Create(iv *models.Interview) error {
	const q = `
		INSERT INTO interviews
			(id, user_id, role, level, style, max_rounds, status, created_at, updated_at)
		VALUES
			(:id, :user_id, :role, :level, :style, :max_rounds, :status, :created_at, :updated_at)`
	_, err := r.db.NamedExec(q, iv)
	return err
}

func (r *InterviewRepository) Update(iv *models.Interview) error {
	const q = `
		UPDATE interviews SET
			status       = :status,
			final_report = :final_report,
			updated_at   = :updated_at
		WHERE id = :id`
	_, err := r.db.NamedExec(q, iv)
	return err
}

func (r *InterviewRepository) FindByID(id string) (*models.Interview, error) {
	var iv models.Interview
	err := r.db.Get(&iv, `SELECT * FROM interviews WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	return &iv, nil
}

func (r *InterviewRepository) FindByUserID(userID string) ([]models.Interview, error) {
	var ivs []models.Interview
	err := r.db.Select(&ivs,
		`SELECT * FROM interviews WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	return ivs, err
}

func (r *InterviewRepository) CreateRound(rnd *models.InterviewRound) error {
	const q = `
		INSERT INTO interview_rounds
			(id, interview_id, round_num, question, answer, score, evaluation_detail, is_followup, created_at)
		VALUES
			(:id, :interview_id, :round_num, :question, :answer, :score, :evaluation_detail, :is_followup, :created_at)`
	_, err := r.db.NamedExec(q, rnd)
	return err
}

func (r *InterviewRepository) UpdateRound(rnd *models.InterviewRound) error {
	const q = `
		UPDATE interview_rounds SET
			answer            = :answer,
			score             = :score,
			evaluation_detail = :evaluation_detail
		WHERE id = :id`
	_, err := r.db.NamedExec(q, rnd)
	return err
}

func (r *InterviewRepository) FindRoundsByInterviewID(interviewID string) ([]models.InterviewRound, error) {
	var rounds []models.InterviewRound
	err := r.db.Select(&rounds,
		`SELECT * FROM interview_rounds WHERE interview_id = $1 ORDER BY round_num, created_at`,
		interviewID,
	)
	return rounds, err
}

func (r *InterviewRepository) FindLatestRound(interviewID string) (*models.InterviewRound, error) {
	var rnd models.InterviewRound
	err := r.db.Get(&rnd,
		`SELECT * FROM interview_rounds WHERE interview_id = $1 ORDER BY created_at DESC LIMIT 1`,
		interviewID,
	)
	if err != nil {
		return nil, err
	}
	return &rnd, nil
}
