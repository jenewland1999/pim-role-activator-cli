package tui

import (
	"fmt"
	"time"
)

// RunWithSpinner displays a spinner with the given message while fn executes.
// Returns the error from fn, if any.
func RunWithSpinner(msg string, fn func() error) error {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	done := make(chan error, 1)
	go func() { done <- fn() }()

	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	i := 0
	for {
		select {
		case err := <-done:
			// Clear the spinner line and return.
			fmt.Printf("\r\033[K")
			return err
		case <-ticker.C:
			fmt.Printf("\r  %s %s", styleCyan.Render(frames[i%len(frames)]), msg)
			i++
		}
	}
}
