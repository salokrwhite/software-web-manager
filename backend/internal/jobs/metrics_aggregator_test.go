package jobs

import (
	"context"
	"testing"
	"time"
)

type noopLogger struct{}

func (noopLogger) Printf(string, ...interface{}) {}

func TestStartNoPanicWithNilDependencies(t *testing.T) {
	Start(context.Background(), nil, time.Second, noopLogger{})
	Start(context.Background(), nil, time.Second, nil)
}

func TestRunOnceNoPanicWithNilDependencies(t *testing.T) {
	RunOnce(nil, time.Now(), noopLogger{})
	RunOnce(nil, time.Now(), nil)
}

