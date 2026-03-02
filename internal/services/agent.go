package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type AgentClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewAgentClient(baseURL string) *AgentClient {
	return &AgentClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

type AgentStartRequest struct {
	Role      string `json:"role"`
	Level     string `json:"level"`
	Style     string `json:"style"`
	MaxRounds int    `json:"max_rounds"`
}

type AgentStartResponse struct {
	Question string `json:"question"`
}

// AgentHistoryEntry mirrors a previously answered round stored in the backend DB.
type AgentHistoryEntry struct {
	Round    int      `json:"round"`
	Question string   `json:"question"`
	Answer   *string  `json:"answer,omitempty"`
	Score    *float64 `json:"score,omitempty"`
	Type     string   `json:"type,omitempty"` // "followup" if this was a follow-up
}

type AgentAnswerRequest struct {
	Role             string              `json:"role"`
	Level            string              `json:"level"`
	Style            string              `json:"style"`
	MaxRounds        int                 `json:"max_rounds"`
	CurrentRound     int                 `json:"current_round"`
	FollowupCount    int                 `json:"followup_count"`
	CurrentQuestion  string              `json:"current_question"`
	Answer           string              `json:"answer"`
	InterviewHistory []AgentHistoryEntry `json:"interview_history"`
}

type AgentAnswerResponse struct {
	EvaluationScore  *float64 `json:"evaluation_score"`
	EvaluationDetail *string  `json:"evaluation_detail"`
	Finished         bool     `json:"finished"`
	NextQuestion     *string  `json:"next_question"`
	IsFollowup       bool     `json:"is_followup"`
	CurrentRound     int      `json:"current_round"`
	FollowupCount    int      `json:"followup_count"`
	Report           *string  `json:"report"`
}

func (c *AgentClient) Start(req AgentStartRequest) (*AgentStartResponse, error) {
	var resp AgentStartResponse
	if err := c.post("/start", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *AgentClient) Answer(req AgentAnswerRequest) (*AgentAnswerResponse, error) {
	var resp AgentAnswerResponse
	if err := c.post("/answer", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *AgentClient) post(path string, body, out interface{}) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Post(c.baseURL+path, "application/json", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("agent request to %s failed: %w", path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("agent returned %d: %s", resp.StatusCode, raw)
	}
	return json.Unmarshal(raw, out)
}
