package handlers

import (
	"errors"
	"log"
	"net/http"

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"interview": iv, "current_question": round})
}

func (h *InterviewHandler) End(c *gin.Context) {
	interviewID := c.Param("id")
	iv, err := h.ivSvc.EndInterview(interviewID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		log.Printf("[SubmitAnswer] interview=%s error=%v", interviewID, err)
		if errors.Is(err, services.ErrAlreadyFinished) {
			c.JSON(http.StatusConflict, gin.H{"error": "interview already finished"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"interviews": ivs})
}

func (h *InterviewHandler) Get(c *gin.Context) {
	userID := c.GetString(middleware.UserIDKey)
	interviewID := c.Param("id")
	iv, rounds, err := h.ivSvc.GetInterview(interviewID, userID)
	if err != nil {
		if errors.Is(err, services.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"interview": iv, "rounds": rounds})
}
