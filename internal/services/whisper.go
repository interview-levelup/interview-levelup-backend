package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

const defaultWhisperURL = "https://api.openai.com/v1/audio/transcriptions"

// WhisperClient calls the OpenAI (or compatible) Whisper STT API directly.
type WhisperClient struct {
	apiKey  string
	baseURL string // leave empty to use defaultWhisperURL
	http    *http.Client
}

func NewWhisperClient(apiKey, baseURL string) *WhisperClient {
	if baseURL == "" {
		baseURL = defaultWhisperURL
	}
	return &WhisperClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

// Transcribe sends audioData to the Whisper API and returns the transcript text.
func (w *WhisperClient) Transcribe(audioData []byte, filename, contentType string) (string, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	// model field
	if err := mw.WriteField("model", "whisper-1"); err != nil {
		return "", err
	}

	// audio file field
	part, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}
	if _, err := part.Write(audioData); err != nil {
		return "", err
	}
	mw.Close()

	req, err := http.NewRequest(http.MethodPost, w.baseURL, &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+w.apiKey)

	resp, err := w.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("whisper request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("whisper returned %d: %s", resp.StatusCode, raw)
	}

	var result struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("whisper parse error: %w", err)
	}
	return result.Text, nil
}
