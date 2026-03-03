package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/fan/interview-levelup-backend/internal/apierror"
	"github.com/fan/interview-levelup-backend/internal/middleware"
	"github.com/fan/interview-levelup-backend/internal/services"
	"github.com/gin-gonic/gin"
)

type InterviewHandler struct {
	ivSvc *services.InterviewService
}

func NewInterviewHandler(svc *services.InterviewService) *InterviewHandler {
	return &InterviewHandler{ivSvc: svc}
}

type startInterviewRequest struct {
	Role      string `json:"role"       binding:"required"`
	Level     string `json:"level"`
	Style     string `json:"style"`
	MaxRounds int    `json:"max_rounds"`
}

type submitAnswerRequest struct {
	Answer string `json:"answer" binding:"required"`
}

func (h *InterviewHandler) Start(c *gin.Context) {
	userID := c.GetString(middleware.UserIDKey)
	var req startInterviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Level == "" {
		req.Level = "junior"
	}
	if req.Style == "" {
		req.Style = "standard"
	}
	if req.MaxRounds <= 0 {
		req.MaxRounds = 5
	}
	iv, round, err := h.ivSvc.StartInterview(userID, req.Role, req.Level, req.Style, req.MaxRounds)
	if err != nil {
		apierror.Respond(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"interview": iv, "current_question": round})
}

func (h *InterviewHandler) End(c *gin.Context) {
	interviewID := c.Param("id")
	iv, err := h.ivSvc.EndInterview(interviewID)
	if err != nil {
		apierror.Respond(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"interview": iv})
}

func (h *InterviewHandler) SubmitAnswer(c *gin.Context) {
	interviewID := c.Param("id")
	var req submitAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	iv, answeredRound, nextRound, err := h.ivSvc.SubmitAnswer(interviewID, req.Answer)
	if err != nil {
		apierror.Respond(c, err)
		return
	}
	resp := gin.H{"interview": iv, "answered_round": answeredRound}
	if iv.Status == "finished" {
		resp["finished"] = true
		resp["final_report"] = iv.FinalReport
	} else {
		resp["finished"] = false
		resp["next_question"] = nextRound
	}
	c.JSON(http.StatusOK, resp)
}

func (h *InterviewHandler) List(c *gin.Context) {
	userID := c.GetString(middleware.UserIDKey)
	ivs, err := h.ivSvc.ListInterviews(userID)
	if err != nil {
		apierror.Respond(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"interviews": ivs})
}

func (h *InterviewHandler) Get(c *gin.Context) {
	userID := c.GetString(middleware.UserIDKey)
	interviewID := c.Param("id")
	iv, rounds, err := h.ivSvc.GetInterview(interviewID, userID)
	if err != nil {
		apierror.Respond(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"interview": iv, "rounds": rounds})
}

// SubmitAnswerStream handles POST /:id/answer/stream.
// Proxies the SSE token stream from the Python agent, then saves the result to DB
// and emits a final "saved" event so the client gets all round data in one connection.
func (h *InterviewHandler) SubmitAnswerStream(c *gin.Context) {
	interviewID := c.Param("id")
	var req submitAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, agentHTTPResp, err := h.ivSvc.PrepareAnswerStream(interviewID, req.Answer)
	if err != nil {
		apierror.Respond(c, err)
		return
	}
	defer agentHTTPResp.Body.Close()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	w := c.Writer
	scanner := bufio.NewScanner(agentHTTPResp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		rawData := line[len("data: "):]

		var evt struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(rawData), &evt); err != nil {
			continue
		}

		switch evt.Type {
		case "token":
			fmt.Fprintf(w, "data: %s\n\n", rawData)
			w.Flush()
		case "done":
			var agentResp services.AgentChatResponse
			if err := json.Unmarshal([]byte(rawData), &agentResp); err != nil {
				sendSSEError(w, "failed to parse done event: "+err.Error())
				return
			}
			iv, answeredRound, nextRound, err := h.ivSvc.FinalizeAnswerStream(ctx, &agentResp)
			if err != nil {
				apierror.RespondSSE(c, err)
				return
			}
			isFinished := iv.Status != "ongoing"
			saved := gin.H{
				"type":           "saved",
				"interview":      iv,
				"finished":       isFinished,
				"answered_round": answeredRound,
				"next_question":  nextRound,
			}
			if isFinished {
				saved["final_report"] = iv.FinalReport
			}
			savedJSON, _ := json.Marshal(saved)
			fmt.Fprintf(w, "data: %s\n\n", savedJSON)
			w.Flush()
			return
		case "error":
			fmt.Fprintf(w, "data: %s\n\n", rawData)
			w.Flush()
			return
		}
	}
	if err := scanner.Err(); err != nil {
		apierror.RespondSSE(c, err)
	}
}

func sendSSEError(w gin.ResponseWriter, msg string) {
	errJSON, _ := json.Marshal(gin.H{"type": "error", "message": msg})
	fmt.Fprintf(w, "data: %s\n\n", errJSON)
	w.Flush()
}

// StartStream handles POST /interviews/stream.
// It creates the interview row immediately and then proxies the first-question SSE
// from the agent. Events emitted:
//
//	{"type": "created", "interview": {...}}   — DB row saved, client can navigate now
//	{"type": "token",   "content": "..."}     — LLM token
//	{"type": "done",    "round": {...}}        — first question persisted
func (h *InterviewHandler) StartStream(c *gin.Context) {
	userID := c.GetString(middleware.UserIDKey)
	var req startInterviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Level == "" {
		req.Level = "junior"
	}
	if req.Style == "" {
		req.Style = "standard"
	}
	if req.MaxRounds <= 0 {
		req.MaxRounds = 5
	}

	ctx, agentHTTPResp, err := h.ivSvc.PrepareStartStream(userID, req.Role, req.Level, req.Style, req.MaxRounds)
	if err != nil {
		apierror.Respond(c, err)
		return
	}
	defer agentHTTPResp.Body.Close()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	w := c.Writer
	// Send interview immediately so the client can navigate before streaming finishes.
	createdJSON, _ := json.Marshal(gin.H{"type": "created", "interview": ctx.Interview()})
	fmt.Fprintf(w, "data: %s\n\n", createdJSON)
	w.Flush()

	// Proxy tokens and accumulate the full question text.
	var questionBuf strings.Builder
	scanner := bufio.NewScanner(agentHTTPResp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		rawData := line[len("data: "):]

		var evt struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(rawData), &evt); err != nil {
			continue
		}

		switch evt.Type {
		case "token":
			var tokenEvt struct {
				Content string `json:"content"`
			}
			if err := json.Unmarshal([]byte(rawData), &tokenEvt); err == nil {
				questionBuf.WriteString(tokenEvt.Content)
			}
			fmt.Fprintf(w, "data: %s\n\n", rawData)
			w.Flush()
		case "done":
			var agentResp services.AgentChatResponse
			if err := json.Unmarshal([]byte(rawData), &agentResp); err != nil {
				sendSSEError(w, "failed to parse done event: "+err.Error())
				return
			}
			fullQuestion := questionBuf.String()
			if fullQuestion == "" && agentResp.Question != nil {
				fullQuestion = *agentResp.Question
			}
			rnd, err := h.ivSvc.FinalizeStartStream(ctx, fullQuestion)
			if err != nil {
				apierror.RespondSSE(c, err)
				return
			}
			doneJSON, _ := json.Marshal(gin.H{"type": "done", "round": rnd})
			fmt.Fprintf(w, "data: %s\n\n", doneJSON)
			w.Flush()
			return
		case "error":
			fmt.Fprintf(w, "data: %s\n\n", rawData)
			w.Flush()
			return
		}
	}
	if err := scanner.Err(); err != nil {
		apierror.RespondSSE(c, err)
	}
}

// Transcribe handles POST /transcribe — receives an audio blob and returns Whisper text.
func (h *InterviewHandler) Transcribe(c *gin.Context) {
	file, header, err := c.Request.FormFile("audio")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "audio file required"})
		return
	}
	defer file.Close()
	audioData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read audio"})
		return
	}
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "audio/webm"
	}
	text, err := h.ivSvc.Transcribe(audioData, header.Filename, contentType)
	if err != nil {
		apierror.Respond(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"text": text})
}
