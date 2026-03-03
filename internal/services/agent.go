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

// AgentHistoryEntry mirrors a previously answered round stored in the backend DB.
type AgentHistoryEntry struct {
	Round    int      `json:"round"`
	Question string   `json:"question"`
	Answer   *string  `json:"answer,omitempty"`
	Score    *float64 `json:"score,omitempty"`
	Type     string   `json:"type,omitempty"` // "followup" if this was a follow-up
}

type AgentChatRequest struct {
	Role             string              `json:"role"`
	Level            string              `json:"level"`
	Style            string              `json:"style"`
	MaxRounds        int                 `json:"max_rounds"`
	CurrentRound     int                 `json:"current_round"`
	FollowupCount    int                 `json:"followup_count"`
	CurrentQuestion  *string             `json:"current_question"` // nil for start
	Answer           *string             `json:"answer"`           // nil for start
	InterviewHistory []AgentHistoryEntry `json:"interview_history"`
}

type AgentChatResponse struct {
	Question         *string  `json:"question"` // next question (start or next round)
	EvaluationScore  *float64 `json:"evaluation_score"`
	EvaluationDetail *string  `json:"evaluation_detail"`
	Finished         bool     `json:"finished"`
	Aborted          bool     `json:"aborted"`
	UserEnded        bool     `json:"user_ended"`
	IsSub            bool     `json:"is_sub"`
	IsFollowup       bool     `json:"is_followup"`
	CurrentRound     int      `json:"current_round"`
	FollowupCount    int      `json:"followup_count"`
	Report           *string  `json:"report"`
}

func (c *AgentClient) Chat(req AgentChatRequest) (*AgentChatResponse, error) {
	var resp AgentChatResponse
	if err := c.post("/chat", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ChatStream opens a streaming POST to the agent's /chat/stream endpoint and
// returns the raw HTTP response. The caller is responsible for closing Body.
// The response body carries SSE events (text/event-stream).
func (c *AgentClient) ChatStream(req AgentChatRequest) (*http.Response, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	// Use a client without timeout for long streaming responses.
	streamClient := &http.Client{}
	httpReq, err := http.NewRequest(http.MethodPost, c.baseURL+"/chat/stream", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	resp, err := streamClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("agent stream request failed: %w", err)
	}
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("agent stream returned %d: %s", resp.StatusCode, raw)
	}
	return resp, nil
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
