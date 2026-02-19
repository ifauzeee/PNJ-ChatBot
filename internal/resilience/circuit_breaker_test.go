package resilience

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/pnj-anonymous-bot/internal/logger"
)

func TestMain(m *testing.M) {
	_ = os.Setenv("APP_ENV", "test")
	_ = os.Setenv("LOG_LEVEL", "error")
	logger.Init()
	code := m.Run()
	_ = logger.Log.Sync()
	os.Exit(code)
}

func TestCircuitBreakerClosedAllowsRequests(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig("test"))
	err := cb.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if cb.State() != StateClosed {
		t.Fatalf("expected StateClosed, got %s", cb.State())
	}
}

func TestCircuitBreakerOpensAfterThreshold(t *testing.T) {
	cfg := CircuitBreakerConfig{
		Name:                "test",
		FailureThreshold:    3,
		ResetTimeout:        1 * time.Second,
		HalfOpenMaxAttempts: 1,
	}
	cb := NewCircuitBreaker(cfg)

	testErr := errors.New("service down")
	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error { return testErr })
	}

	if cb.State() != StateOpen {
		t.Fatalf("expected StateOpen after %d failures, got %s", cfg.FailureThreshold, cb.State())
	}

	err := cb.Execute(func() error { return nil })
	if err == nil {
		t.Fatal("expected error when circuit is open")
	}
}

func TestCircuitBreakerTransitionsToHalfOpen(t *testing.T) {
	cfg := CircuitBreakerConfig{
		Name:                "test",
		FailureThreshold:    2,
		ResetTimeout:        50 * time.Millisecond,
		HalfOpenMaxAttempts: 1,
	}
	cb := NewCircuitBreaker(cfg)

	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error { return errors.New("fail") })
	}
	if cb.State() != StateOpen {
		t.Fatalf("expected StateOpen, got %s", cb.State())
	}

	time.Sleep(60 * time.Millisecond)

	err := cb.Execute(func() error { return nil })
	if err != nil {
		t.Fatalf("expected success in half-open, got %v", err)
	}

	if cb.State() != StateClosed {
		t.Fatalf("expected StateClosed after half-open success, got %s", cb.State())
	}
}

func TestCircuitBreakerReopensOnHalfOpenFailure(t *testing.T) {
	cfg := CircuitBreakerConfig{
		Name:                "test",
		FailureThreshold:    2,
		ResetTimeout:        50 * time.Millisecond,
		HalfOpenMaxAttempts: 2,
	}
	cb := NewCircuitBreaker(cfg)

	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error { return errors.New("fail") })
	}

	time.Sleep(60 * time.Millisecond)

	_ = cb.Execute(func() error { return errors.New("still failing") })

	if cb.State() != StateOpen {
		t.Fatalf("expected StateOpen after half-open failure, got %s", cb.State())
	}
}

func TestCircuitBreakerResetOnSuccess(t *testing.T) {
	cfg := CircuitBreakerConfig{
		Name:                "test",
		FailureThreshold:    3,
		ResetTimeout:        1 * time.Second,
		HalfOpenMaxAttempts: 1,
	}
	cb := NewCircuitBreaker(cfg)

	_ = cb.Execute(func() error { return errors.New("fail1") })
	_ = cb.Execute(func() error { return errors.New("fail2") })

	if cb.State() != StateClosed {
		t.Fatalf("should still be closed with %d failures", 2)
	}

	_ = cb.Execute(func() error { return nil })

	if cb.State() != StateClosed {
		t.Fatalf("expected StateClosed after success, got %s", cb.State())
	}

	_ = cb.Execute(func() error { return errors.New("f1") })
	_ = cb.Execute(func() error { return errors.New("f2") })

	if cb.State() != StateClosed {
		t.Fatalf("should still be closed, got %s", cb.State())
	}

	_ = cb.Execute(func() error { return errors.New("f3") })
	if cb.State() != StateOpen {
		t.Fatalf("expected StateOpen after 3 consecutive failures, got %s", cb.State())
	}
}

func TestCircuitStateString(t *testing.T) {
	tests := []struct {
		state    CircuitState
		expected string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{CircuitState(99), "unknown"},
	}

	for _, tt := range tests {
		if tt.state.String() != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.state.String())
		}
	}
}
