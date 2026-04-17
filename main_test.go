package main

import (
	"sync"
	"testing"
	"time"
)

func runScenario() {
	// Disable logs in tests to keep output clean and reduce overhead.
	a := NewAirlockWithLogging(false)
	var wg sync.WaitGroup

	wg.Add(2)
	go insideAstronaut(a, &wg)
	go outsideAstronaut(a, &wg)
	wg.Wait()

	a.mu.Lock()
	defer a.mu.Unlock()
	a.assertSafe()
}

func TestAirlock_NoDeadlockAndSafe(t *testing.T) {
	// Run the scenario multiple times to stress different goroutine interleavings.
	for i := 0; i < 100; i++ {
		done := make(chan struct{})
		go func() {
			runScenario()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			// Timeout usually means both actors are waiting on each other (deadlock).
			t.Fatalf("scenario %d timed out (possible deadlock)", i)
		}
	}
}
