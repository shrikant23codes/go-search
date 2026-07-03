package circuit

import (
	"log"
	"time"

	"github.com/sony/gobreaker/v2"
)

func NewBreaker[T any](name string, isExcluded func(error) bool) *gobreaker.CircuitBreaker[T] {

	return gobreaker.NewCircuitBreaker[T](gobreaker.Settings{
		Name:        name,
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     30 * time.Second,

		ReadyToTrip: func(counts gobreaker.Counts) bool {
			if counts.ConsecutiveFailures >= 1 {
				return true
			}

			if counts.Requests < 10 {
				return false
			}

			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)

			return failureRatio >= 0.5
		},
		IsExcluded: isExcluded,

		OnStateChange: func(name string, from, to gobreaker.State) {
			log.Printf(
				"circuit breaker: %s: %s -> %s",
				name, from, to,
			)
		},
	})

}
