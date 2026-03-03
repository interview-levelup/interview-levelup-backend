package services

import (
	"encoding/json"
	"fmt"
	"net/http"
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
// isTerminalStatus returns true when an interview can no longer accept answers.
func isTerminalStatus(s string) bool {
	return s == "finished" || s == "aborted" || s == "user_ended"
}

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
		Detail  *string  `json:"detail"` // fallback key some LLMs use
	}
	if err := json.Unmarshal([]byte(trimmed), &obj); err != nil {
		return score, detail
	}
	// Prefer "details", fall back to "detail"
	cleanDetail := obj.Details
	if cleanDetail == nil {
		cleanDetail = obj.Detail
	}
	if cleanDetail != nil {
		detail = cleanDetail
	}
	// Always take score from JSON when present (overrides separate field if set)
	if obj.Score != nil {
		score = obj.Score
	}
	return score, detail
}

type InterviewService struct {
	repo    *repository.InterviewRepository
	agent   *AgentClient
	whisper *WhisperClient
}

func NewInterviewService(repo *repository.InterviewRepository, agent *AgentClient, whisper *WhisperClient) *InterviewService {
	return &InterviewService{repo: repo, agent: agent, whisper: whisper}
}

func (s *InterviewService) StartInterview(userID, role, level, style string, maxRounds int) (*models.Interview, *models.InterviewRound, error) {
	agentResp, err := s.agent.Chat(AgentChatRequest{
		Role:             role,
		Level:            level,
		Style:            style,
		MaxRounds:        maxRounds,
		InterviewHistory: make([]AgentHistoryEntry, 0),
	})
	if err != nil {
		return nil, nil, err
	}
	now := time.Now().UTC()
	iv := &models.Interview{
		ID:        uuid.NewString(),
		UserID:    userID,
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
		Question:    *agentResp.Question,
		IsFollowup:  false,
		CreatedAt:   now,
	}
	if err := s.repo.CreateRound(rnd); err != nil {
		return nil, nil, err
	}
	return iv, rnd, nil
}

// SubmitStreamCtx holds prepared state shared between PrepareAnswerStream and FinalizeAnswerStream.
type SubmitStreamCtx struct {
	iv       *models.Interview
	latest   models.InterviewRound
	agentReq AgentChatRequest
	answer   string
}

func (s *InterviewService) prepareSubmit(interviewID, answer string) (*SubmitStreamCtx, error) {
	iv, err := s.repo.FindByID(interviewID)
	if err != nil {
		return nil, err
	}
	if isTerminalStatus(iv.Status) {
		return nil, ErrAlreadyFinished
	}

	allRounds, err := s.repo.FindRoundsByInterviewID(interviewID)
	if err != nil {
		return nil, err
	}
	if len(allRounds) == 0 {
		return nil, fmt.Errorf("no rounds found for interview %s", interviewID)
	}

	latest := allRounds[len(allRounds)-1]
	if latest.Answer != nil {
		return nil, fmt.Errorf("latest round %s already has an answer — interview state may be inconsistent", latest.ID)
	}

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
		} else if r.IsSub {
			entry.Type = "sub"
		}
		history = append(history, entry)
	}

	followupCount := 0
	for _, r := range allRounds {
		if r.IsFollowup && r.RoundNum == latest.RoundNum {
			followupCount++
		}
	}

	return &SubmitStreamCtx{
		iv:     iv,
		latest: latest,
		answer: answer,
		agentReq: AgentChatRequest{
			Role:             iv.Role,
			Level:            iv.Level,
			Style:            iv.Style,
			MaxRounds:        iv.MaxRounds,
			CurrentRound:     latest.RoundNum,
			FollowupCount:    followupCount,
			CurrentQuestion:  &latest.Question,
			Answer:           &answer,
			InterviewHistory: history,
		},
	}, nil
}

func (s *InterviewService) finalizeSubmit(ctx *SubmitStreamCtx, agentResp *AgentChatResponse) (*models.Interview, *models.InterviewRound, *models.InterviewRound, error) {
	now := time.Now().UTC()
	cleanScore, cleanDetail := sanitizeEval(agentResp.EvaluationScore, agentResp.EvaluationDetail)

	ctx.latest.Answer = &ctx.answer
	ctx.latest.Score = cleanScore
	ctx.latest.EvaluationDetail = cleanDetail
	ctx.latest.AnsweredAt = &now
	if err := s.repo.UpdateRound(&ctx.latest); err != nil {
		return nil, nil, nil, err
	}

	var nextRound *models.InterviewRound
	if agentResp.Finished {
		if agentResp.UserEnded {
			ctx.iv.Status = "user_ended"
		} else if agentResp.Aborted {
			ctx.iv.Status = "aborted"
		} else {
			ctx.iv.Status = "finished"
		}
		ctx.iv.FinalReport = agentResp.Report
		ctx.iv.UpdatedAt = now
		if err := s.repo.Update(ctx.iv); err != nil {
			return nil, nil, nil, err
		}
	} else if agentResp.Question != nil {
		nextRound = &models.InterviewRound{
			ID:          uuid.NewString(),
			InterviewID: ctx.iv.ID,
			RoundNum:    agentResp.CurrentRound,
			Question:    *agentResp.Question,
			IsFollowup:  agentResp.IsFollowup,
			IsSub:       agentResp.IsSub,
			CreatedAt:   now,
		}
		if err := s.repo.CreateRound(nextRound); err != nil {
			return nil, nil, nil, err
		}
		ctx.iv.UpdatedAt = now
		if err := s.repo.Update(ctx.iv); err != nil {
			return nil, nil, nil, err
		}
	}
	return ctx.iv, &ctx.latest, nextRound, nil
}

func (s *InterviewService) SubmitAnswer(interviewID, answer string) (*models.Interview, *models.InterviewRound, *models.InterviewRound, error) {
	ctx, err := s.prepareSubmit(interviewID, answer)
	if err != nil {
		return nil, nil, nil, err
	}

	agentResp, err := s.agent.Chat(ctx.agentReq)
	if err != nil {
		return nil, nil, nil, err
	}

	return s.finalizeSubmit(ctx, agentResp)
}

// PrepareAnswerStream prepares the agent request and opens a streaming SSE connection
// to the agent. The caller must close the returned http.Response.Body when done.
func (s *InterviewService) PrepareAnswerStream(interviewID, answer string) (*SubmitStreamCtx, *http.Response, error) {
	ctx, err := s.prepareSubmit(interviewID, answer)
	if err != nil {
		return nil, nil, err
	}
	resp, err := s.agent.ChatStream(ctx.agentReq)
	if err != nil {
		return nil, nil, err
	}
	return ctx, resp, nil
}

// FinalizeAnswerStream persists the agent response to the DB (same finalisation as SubmitAnswer).
func (s *InterviewService) FinalizeAnswerStream(ctx *SubmitStreamCtx, agentResp *AgentChatResponse) (*models.Interview, *models.InterviewRound, *models.InterviewRound, error) {
	return s.finalizeSubmit(ctx, agentResp)
}

// EndInterview is a safety-net for edge cases (e.g. no pending round to submit through).
// The normal path for user-initiated ending is submitting "我想结束面试" via SubmitAnswer.
func (s *InterviewService) EndInterview(interviewID string) (*models.Interview, error) {
	iv, err := s.repo.FindByID(interviewID)
	if err != nil {
		return nil, err
	}
	if isTerminalStatus(iv.Status) {
		return iv, nil // idempotent
	}
	iv.Status = "finished"
	iv.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(iv); err != nil {
		return nil, err
	}
	return iv, nil
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

// StartStreamCtx holds prepared state for the streaming start flow.
type StartStreamCtx struct {
	iv *models.Interview
}

// Interview returns the interview created for this stream.
func (c *StartStreamCtx) Interview() *models.Interview { return c.iv }

// PrepareStartStream creates the interview DB row immediately and opens an agent
// stream for the first question. The caller must close the returned http.Response.Body.
func (s *InterviewService) PrepareStartStream(userID, role, level, style string, maxRounds int) (*StartStreamCtx, *http.Response, error) {
	now := time.Now().UTC()
	iv := &models.Interview{
		ID:        uuid.NewString(),
		UserID:    userID,
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
	agentReq := AgentChatRequest{
		Role:             role,
		Level:            level,
		Style:            style,
		MaxRounds:        maxRounds,
		InterviewHistory: make([]AgentHistoryEntry, 0),
	}
	resp, err := s.agent.ChatStream(agentReq)
	if err != nil {
		return nil, nil, err
	}
	return &StartStreamCtx{iv: iv}, resp, nil
}

// FinalizeStartStream saves the first question round to DB.
func (s *InterviewService) FinalizeStartStream(ctx *StartStreamCtx, question string) (*models.InterviewRound, error) {
	rnd := &models.InterviewRound{
		ID:          uuid.NewString(),
		InterviewID: ctx.iv.ID,
		RoundNum:    0,
		Question:    question,
		IsFollowup:  false,
		CreatedAt:   time.Now().UTC(),
	}
	if err := s.repo.CreateRound(rnd); err != nil {
		return nil, err
	}
	return rnd, nil
}

// Transcribe sends audio to the Whisper API and returns the transcript.
func (s *InterviewService) Transcribe(audioData []byte, filename, contentType string) (string, error) {
	return s.whisper.Transcribe(audioData, filename, contentType)
}
