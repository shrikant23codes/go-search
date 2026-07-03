package circuit

import (
	"errors"
	"testing"

	"github.com/sony/gobreaker/v2"
)

func TestBreakerOpensAfterConsecutiveFailures(t *testing.T) {
	breaker := NewBreaker[int]("test-shard", nil)

	backendErr := errors.New("backend unavailable")

	for i := 0; i < 4; i++ {
		_, err := breaker.Execute(func() (int, error) {
			return 0, backendErr
		})

		if !errors.Is(err, backendErr) {
			t.Fatalf("request: %d returned %v", i+1, err)
		}
	}

	if got := breaker.State(); got != gobreaker.StateClosed {
		t.Fatalf(
			"state after 4 failures = %s, want closed",
			got,
		)
	}

	_, err := breaker.Execute(func() (int, error) {
		return 0, backendErr
	})

	if !errors.Is(err, backendErr) {
		t.Fatalf("5th request returned %v", err)
	}

	if got := breaker.State(); got != gobreaker.StateOpen {
		t.Fatalf("state after 5 failures = %s, want open", got)
	}

	called := false

	_, err = breaker.Execute(func() (int, error) {
		called = true
		return 1, nil
	})

	if !errors.Is(err, gobreaker.ErrOpenState) {
		t.Fatalf("open breaker returned %v but want ErrOpenState", err)
	}

	if called {
		t.Fatalf("Function called while circuit was open")
	}

}
