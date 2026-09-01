package jobs

import (
	"errors"
	"time"
)

type PermanentError struct {
	Err error
}

func (e *PermanentError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return "permanent error"
}

func (e *PermanentError) Unwrap() error {
	return e.Err
}

func NewPermanentError(err error) error {
	return &PermanentError{Err: err}
}

func IsPermanent(err error) bool {
	var permErr *PermanentError
	return errors.As(err, &permErr)
}

func CalculateBackoff(attempt int, baseDelay, maxDelay time.Duration) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	// delay = baseDelay * 2^(attempt - 1)
	multiplier := time.Duration(1 << uint(attempt-1))
	delay := baseDelay * multiplier

	if delay > maxDelay {
		return maxDelay
	}
	if delay < 0 {
		return maxDelay // overflow protection
	}
	return delay
}
