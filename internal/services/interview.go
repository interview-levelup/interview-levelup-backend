package services

import (
	"time"

	"github.com/fan/interview-levelup-backend/internal/models"
	"github.com/fan/interview-levelup-backend/internal/repository"
	"github.com/google/uuid"
)

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
		ThreadID:  agentResp.ThreadID,
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
	var rnd *models.InterviewRound
	if agentResp.Question != nil {
		rnd = &models.InterviewRound{
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
	}
	return iv, rnd, nil
}

func (s *InterviewService) SubmitAnswer(interviewID, answer string) (*models.Interview, *models.InterviewRound, error) {
	iv, err := s.repo.FindByID(interviewID)
	if err != nil {
		return nil, nil, err
	}
	agentResp, err := s.agent.Answer(AgentAnswerRequest{
		ThreadID: iv.ThreadID,
		Answer:   answer,
	})
	if err != nil {
		return nil, nil, err
	}
	now := time.Now().UTC()
	latestRound, err := s.repo.FindLatestRound(interviewID)
	if err != nil {
		return nil, nil, err
	}
	latestRound.Answer = &answer
	latestRound.Score = agentResp.State.EvaluationScore
	latestRound.EvaluationDetail = agentResp.State.EvaluationDetail
	if err := s.repo.UpdateRound(latestRound); err != nil {
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
	} else if agentResp.Question != nil {
		isFollowup := agentResp.State.InterviewStage == "followup"
		nextRound = &models.InterviewRound{
			ID:          uuid.NewString(),
			InterviewID: iv.ID,
			RoundNum:    agentResp.State.CurrentRound,
			Question:    *agentResp.Question,
			IsFollowup:  isFollowup,
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
