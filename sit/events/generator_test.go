package events_test

import (
	"testing"
	"time"

	"github.com/streadway/handy/atomic"
	"github.com/stretchr/testify/assert"

	"github.com/Fellah/go-sit/sit/events"
)

func TestEventGenerator(t *testing.T) {
	execs := 11
	execFreq := 101 * time.Millisecond
	generatorPeriod := time.Duration(execs) * execFreq

	var execsCounter atomic.Int
	stopCh := events.StartLinearGenerator(execFreq, func() {
		execsCounter.Add(1)
	})

	time.Sleep(generatorPeriod)
	stopCh <- true

	assert.InDelta(t, execs, execsCounter.Get(), 1)
}
