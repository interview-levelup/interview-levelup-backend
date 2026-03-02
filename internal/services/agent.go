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
	ThreadID string            `json:"thread_id"`
	Question *string           `json:"question"`
	State    AgentStatePayload `json:"state"`
}

type AgentAnswerRequest struct {
	ThreadID string `json:"thread_id"`
	Answer   string `json:"answer"`
}

type AgentAnswerResponse struct {
	ThreadID string            `json:"thread_id"`
	Question *string           `json:"question"`
	Report   *string           `json:"report"`
	Finished bool              `json:"finished"`
	State    AgentStatePayload `json:"state"`
}

type AgentStatePayload struct {
	Role             string      `json:"role"`
	Level            string      `json:"level"`
	Style            string      `json:"style"`
	CurrentRound     int         `json:"current_round"`
	MaxRounds        int         `json:"max_rounds"`
	CurrentQuestion  *string     `json:"current_question"`
	CandidateAnswer  *string     `json:"candidate_answer"`
	EvaluationScore  *float64    `json:"evaluation_score"`
	EvaluationDetail *string     `json:"evaluation_detail"`
	FollowupCount    int         `json:"followup_count"`
	InterviewHistory interface{} `json:"interview_history"`
	InterviewStage   string      `json:"interview_stage"`
	FinalReport      *string     `json:"final_report"`
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
