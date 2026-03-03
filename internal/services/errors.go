package services

import "github.com/fan/interview-levelup-backend/internal/apierror"

// Sentinel errors exposed by the service layer.
var (
	ErrForbidden       = apierror.ErrForbidden
	ErrAlreadyFinished = apierror.ErrAlreadyFinished
)
