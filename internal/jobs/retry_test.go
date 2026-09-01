package jobs

import (
	"errors"
	"testing"
	"time"
)

func TestCalculateBackoff(t *testing.T) {
	base := 1 * time.Second
	max := 30 * time.Second

	// attempt 1: 1s * 2^0 = 1s
	if d := CalculateBackoff(1, base, max); d != 1*time.Second {
		t.Errorf("expected 1s, got %v", d)
	}
	// attempt 2: 1s * 2^1 = 2s
	if d := CalculateBackoff(2, base, max); d != 2*time.Second {
		t.Errorf("expected 2s, got %v", d)
	}
	// attempt 3: 1s * 2^2 = 4s
	if d := CalculateBackoff(3, base, max); d != 4*time.Second {
		t.Errorf("expected 4s, got %v", d)
	}
	// attempt 10: should cap at max (30s)
	if d := CalculateBackoff(10, base, max); d != 30*time.Second {
		t.Errorf("expected 30s max cap, got %v", d)
	}
}

func TestPermanentError(t *testing.T) {
	baseErr := errors.New("bad request")
	permErr := NewPermanentError(baseErr)

	if !IsPermanent(permErr) {
		t.Error("expected error to be recognized as permanent")
	}

	stdErr := errors.New("connection timeout")
	if IsPermanent(stdErr) {
		t.Error("expected standard error not to be permanent")
	}
}
