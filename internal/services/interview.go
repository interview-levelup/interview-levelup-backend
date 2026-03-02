package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fan/interview-levelup-backend/internal/models"
	"github.com/fan/interview-levelup-backend/internal/repository"
	"github.com/google/uuid"
)

// sanitizeEval normalises the evaluation fields coming from the agent.
// If the agent's JSON parser failed, EvaluationDetail may itself be a raw
// JSON string like {"score":45,"details":"..."}. We extract the fields here
// so the DB and API always surface clean typed data.
func sanitizeEval(score *float64, detail *string) (*float64, *string) {
	if detail == nil {
		return score, detail
	}
	trimmed := strings.TrimSpace(*detail)
	if !strings.HasPrefix(trimmed, "{") {
		return score, detail
	}
	var obj struct {
		Score   *float64 `json:"score"`
		Details *string  `json:"details"`
	}
	if err := json.Unmarshal([]byte(trimmed), &obj); err != nil {
		return score, detail
	}
	if obj.Details != nil {
		detail = obj.Details
	}
	if score == nil && obj.Score != nil {
		score = obj.Score
	}
	return score, detail
}

type InterviewService struct {
	repo  *repository.InterviewRepository
	agent *AgentClient
}

func NewInterviewService(repo *repository.InterviewRepository, agent *AgentClient) *InterviewService {
	return &InterviewService{repo: repo, agent: agent}
}

func (s *InterviewService) StartInterview(userID, role, level, style string, maxRounds int) (*models.Interview, *models.InterviewRound, error) {
	agentResp, err := s.agent.Start(AgentStartRequest{
		Role:      role,
		Level:     level,
		Style:     style,
		MaxRounds: maxRounds,
	})
	if err != nil {
		return nil, nil, err
	}
	now := time.Now().UTC()
	iv := &models.Interview{
		ID:        uuid.NewString(),
		UserID:    userID,
		ThreadID:  "",
		Role:      role,
		Level:     level,
		Style:     style,
		MaxRounds: maxRounds,
		Status:    "ongoing",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Create(iv); err != nil {
		return nil, nil, err
	}
	rnd := &models.InterviewRound{
		ID:          uuid.NewString(),
		InterviewID: iv.ID,
		RoundNum:    0,
		Question:    agentResp.Question,
		IsFollowup:  false,
		CreatedAt:   now,
	}
	if err := s.repo.CreateRound(rnd); err != nil {
		return nil, nil, err
	}
	return iv, rnd, nil
}

func (s *InterviewService) SubmitAnswer(interviewID, answer string) (*models.Interview, *models.InterviewRound, error) {
	iv, err := s.repo.FindByID(interviewID)
	if err != nil {
		return nil, nil, err
	}
	if iv.Status == "finished" {
		return nil, nil, ErrAlreadyFinished
	}

	// Fetch all rounds so we can reconstruct state for the stateless agent.
	allRounds, err := s.repo.FindRoundsByInterviewID(interviewID)
	if err != nil {
		return nil, nil, err
	}
	if len(allRounds) == 0 {
		return nil, nil, fmt.Errorf("no rounds found for interview %s", interviewID)
	}

	// Latest round = the current unanswered question.
	latest := allRounds[len(allRounds)-1]
	if latest.Answer != nil {
		return nil, nil, fmt.Errorf("latest round %s already has an answer — interview state may be inconsistent", latest.ID)
	}

	// Build history from all *previously answered* rounds (everything except latest).
	// Use an empty slice (not nil) so it serializes as [] not null for the agent.
	history := make([]AgentHistoryEntry, 0, len(allRounds)-1)
	for _, r := range allRounds[:len(allRounds)-1] {
		entry := AgentHistoryEntry{
			Round:    r.RoundNum,
			Question: r.Question,
			Answer:   r.Answer,
			Score:    r.Score,
		}
		if r.IsFollowup {
			entry.Type = "followup"
		}
		history = append(history, entry)
	}

	// followup_count = how many followup rounds share the same round_num as the
	// current question (including the current one, since that count was already
	// incremented when the agent generated this question).
	followupCount := 0
	for _, r := range allRounds {
		if r.IsFollowup && r.RoundNum == latest.RoundNum {
			followupCount++
		}
	}

	agentResp, err := s.agent.Answer(AgentAnswerRequest{
		Role:             iv.Role,
		Level:            iv.Level,
		Style:            iv.Style,
		MaxRounds:        iv.MaxRounds,
		CurrentRound:     latest.RoundNum,
		FollowupCount:    followupCount,
		CurrentQuestion:  latest.Question,
		Answer:           answer,
		InterviewHistory: history,
	})
	if err != nil {
		return nil, nil, err
	}

	now := time.Now().UTC()

	// Normalise eval fields — backend is the single source of truth, not the frontend.
	cleanScore, cleanDetail := sanitizeEval(agentResp.EvaluationScore, agentResp.EvaluationDetail)

	// Persist evaluation result onto the round that was just answered.
	latest.Answer = &answer
	latest.Score = cleanScore
	latest.EvaluationDetail = cleanDetail
	if err := s.repo.UpdateRound(&latest); err != nil {
		return nil, nil, err
	}

	var nextRound *models.InterviewRound
	if agentResp.Finished {
		iv.Status = "finished"
		iv.FinalReport = agentResp.Report
		iv.UpdatedAt = now
		if err := s.repo.Update(iv); err != nil {
			return nil, nil, err
		}
	} else if agentResp.NextQuestion != nil {
		nextRound = &models.InterviewRound{
			ID:          uuid.NewString(),
			InterviewID: iv.ID,
			RoundNum:    agentResp.CurrentRound,
			Question:    *agentResp.NextQuestion,
			IsFollowup:  agentResp.IsFollowup,
			CreatedAt:   now,
		}
		if err := s.repo.CreateRound(nextRound); err != nil {
			return nil, nil, err
		}
		iv.UpdatedAt = now
		if err := s.repo.Update(iv); err != nil {
			return nil, nil, err
		}
	}
	return iv, nextRound, nil
}

func (s *InterviewService) GetInterview(interviewID, userID string) (*models.Interview, []models.InterviewRound, error) {
	iv, err := s.repo.FindByID(interviewID)
	if err != nil {
		return nil, nil, err
	}
	if iv.UserID != userID {
		return nil, nil, ErrForbidden
	}
	rounds, err := s.repo.FindRoundsByInterviewID(interviewID)
	if err != nil {
		return nil, nil, err
	}
	return iv, rounds, nil
}

func (s *InterviewService) ListInterviews(userID string) ([]models.Interview, error) {
	return s.repo.FindByUserID(userID)
}
