package events

import "time"

// ExecFunc is function that will be executed with specific frequency by event generator.
type ExecFunc func()

// StartLinearGenerator launches new goroutine to execute provided function with certain frequency.
func StartLinearGenerator(freq time.Duration, execFn ExecFunc) chan<- bool {
	ticker := time.NewTicker(freq)
	stopCh := make(chan bool)

	go func() {
		for {
			select {
			case <-ticker.C:
				go execFn()
			case <-stopCh:
				close(stopCh)
				return
			}
		}
	}()

	return stopCh
}
