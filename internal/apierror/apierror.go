package apierror

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AppError is a user-facing error with an associated HTTP status code.
type AppError struct {
	Status  int
	Message string
}

func (e *AppError) Error() string { return e.Message }

// Constructors for common cases.

func BadRequest(msg string) *AppError   { return &AppError{http.StatusBadRequest, msg} }
func Unauthorized(msg string) *AppError { return &AppError{http.StatusUnauthorized, msg} }
func Forbidden() *AppError              { return &AppError{http.StatusForbidden, "forbidden"} }
func NotFound(msg string) *AppError     { return &AppError{http.StatusNotFound, msg} }
func Conflict(msg string) *AppError     { return &AppError{http.StatusConflict, msg} }

// Sentinel errors shared across packages.
var (
	ErrForbidden       = Forbidden()
	ErrNotFound        = NotFound("not found")
	ErrAlreadyFinished = Conflict("interview already finished")
)

// Respond writes the appropriate JSON response for err.
// If err is an *AppError it is treated as a known, user-facing error.
// Any other error is logged and mapped to 500 with a generic message.
func Respond(c *gin.Context, err error) {
	var ae *AppError
	if errors.As(err, &ae) {
		c.JSON(ae.Status, gin.H{"error": ae.Message})
		return
	}
	log.Printf("[%s %s] internal error: %v", c.Request.Method, c.FullPath(), err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}

// RespondSSE writes an SSE error event (already in streaming mode).
func RespondSSE(c *gin.Context, err error) {
	var ae *AppError
	if errors.As(err, &ae) {
		writeSSEError(c.Writer, ae.Message)
		return
	}
	log.Printf("[%s %s] internal error: %v", c.Request.Method, c.FullPath(), err)
	writeSSEError(c.Writer, "internal server error")
}

func writeSSEError(w gin.ResponseWriter, msg string) {
	errJSON := `data: {"type":"error","message":"` + msg + `"}` + "\n\n"
	w.WriteString(errJSON)
	w.Flush()
}
