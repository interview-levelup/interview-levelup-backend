package services

import "errors"

var ErrForbidden = errors.New("forbidden")
var ErrAlreadyFinished = errors.New("interview already finished")
