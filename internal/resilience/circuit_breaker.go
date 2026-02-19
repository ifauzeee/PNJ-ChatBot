package resilience

import (
	"fmt"
	"sync"
	"time"

	"github.com/pnj-anonymous-bot/internal/logger"
	"go.uber.org/zap"
)

type CircuitState int

const (
	StateClosed CircuitState = iota

	StateOpen

	StateHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

type CircuitBreakerConfig struct {
	Name string

	FailureThreshold int

	ResetTimeout time.Duration

	HalfOpenMaxAttempts int
}

func DefaultCircuitBreakerConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:                name,
		FailureThreshold:    5,
		ResetTimeout:        30 * time.Second,
		HalfOpenMaxAttempts: 2,
	}
}

type CircuitBreaker struct {
	mu               sync.RWMutex
	config           CircuitBreakerConfig
	state            CircuitState
	failureCount     int
	successCount     int
	lastFailureTime  time.Time
	halfOpenAttempts int
}

func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config: cfg,
		state:  StateClosed,
	}
}

func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.allowRequest() {
		return fmt.Errorf("circuit breaker [%s] is open — service temporarily unavailable", cb.config.Name)
	}

	err := fn()
	cb.recordResult(err)
	return err
}

func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:

		if time.Since(cb.lastFailureTime) >= cb.config.ResetTimeout {
			cb.state = StateHalfOpen
			cb.halfOpenAttempts = 0
			cb.successCount = 0
			logger.Info("Circuit breaker transitioning to half-open",
				zap.String("breaker", cb.config.Name),
			)
			return true
		}
		return false
	case StateHalfOpen:
		return cb.halfOpenAttempts < cb.config.HalfOpenMaxAttempts
	default:
		return false
	}
}

func (cb *CircuitBreaker) recordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.onFailure()
	} else {
		cb.onSuccess()
	}
}

func (cb *CircuitBreaker) onFailure() {
	cb.failureCount++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		if cb.failureCount >= cb.config.FailureThreshold {
			cb.state = StateOpen
			logger.Warn("Circuit breaker opened",
				zap.String("breaker", cb.config.Name),
				zap.Int("failures", cb.failureCount),
			)
		}
	case StateHalfOpen:
		cb.halfOpenAttempts++
		cb.state = StateOpen
		logger.Warn("Circuit breaker re-opened from half-open",
			zap.String("breaker", cb.config.Name),
		)
	}
}

func (cb *CircuitBreaker) onSuccess() {
	switch cb.state {
	case StateClosed:
		cb.failureCount = 0
	case StateHalfOpen:
		cb.halfOpenAttempts++
		cb.successCount++
		if cb.successCount >= cb.config.HalfOpenMaxAttempts {
			cb.state = StateClosed
			cb.failureCount = 0
			cb.successCount = 0
			logger.Info("Circuit breaker closed (service recovered)",
				zap.String("breaker", cb.config.Name),
			)
		}
	}
}
