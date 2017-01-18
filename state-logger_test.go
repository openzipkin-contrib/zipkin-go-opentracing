package zipkintracer

import (
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
	"fmt"
)


type mockLogger struct {
	mock.Mock
}

// Log is a mock for the log function
func (l *mockLogger) Log(keyvals ...interface{}) error {
	args := l.Called(keyvals...)
	return args.Error(0)
}

func TestStateLogger(t *testing.T) {
	safeWait := 100 * time.Millisecond
	err1 := fmt.Errorf("error 1")
	err2 := fmt.Errorf("error 2")
	fixed := "fixed"

	m := new(mockLogger)

	l := NewStateLogger(m, safeWait)

	m.On("Log", "err", err1.Error()).Return(nil).Once()
	l.LogError(err1)
	m.AssertNumberOfCalls(t, "Log", 1)

	m.On("Log", "err", err2.Error()).Return(nil).Once()
	l.LogError(err2)
	m.AssertNumberOfCalls(t, "Log", 2)

	m.On("Log", "err", err1.Error()).Return(nil).Once()
	l.LogError(err1)
	l.LogError(err1)
	m.AssertNumberOfCalls(t, "Log", 3)

	time.Sleep(safeWait)

	m.On("Log", "err", err1.Error()).Return(nil).Once()
	l.LogError(err1)
	l.LogError(err1)
	m.AssertNumberOfCalls(t, "Log", 4)

	m.On("Log", "err", err2.Error()).Return(nil).Once()
	l.LogError(err2)
	m.AssertNumberOfCalls(t, "Log", 5)

	m.On("Log", fixed).Return(nil).Once()
	l.Fixed(fixed)
	m.AssertNumberOfCalls(t, "Log", 6)

	l.Fixed(fixed)
	m.AssertNumberOfCalls(t, "Log", 6)

	m.On("Log", "err", err2.Error()).Return(nil).Once()
	l.LogError(err2)
	m.AssertNumberOfCalls(t, "Log", 7)
}

func TestStateLoggerAlwaysLog(t *testing.T) {
	err1 := fmt.Errorf("error 1")
	err2 := fmt.Errorf("error 2")
	fixed := "fixed"

	m := new(mockLogger)

	l := NewStateLogger(m, 0)

	m.On("Log", "err", err1.Error()).Return(nil).Once()
	l.LogError(err1)
	m.AssertNumberOfCalls(t, "Log", 1)

	m.On("Log", "err", err1.Error()).Return(nil).Once()
	l.LogError(err1)
	m.AssertNumberOfCalls(t, "Log", 2)

	m.On("Log", "err", err2.Error()).Return(nil).Once()
	l.LogError(err2)
	m.AssertNumberOfCalls(t, "Log", 3)

	m.On("Log", "err", err2.Error()).Return(nil).Once()
	l.LogError(err2)
	m.AssertNumberOfCalls(t, "Log", 4)

	m.On("Log", "err", err1.Error()).Return(nil).Once()
	l.LogError(err1)
	m.AssertNumberOfCalls(t, "Log", 5)

	l.Fixed(fixed)
	m.AssertNumberOfCalls(t, "Log", 5)

	l.Fixed(fixed)
	m.AssertNumberOfCalls(t, "Log", 5)

	m.On("Log", "err", err2.Error()).Return(nil).Once()
	l.LogError(err2)
	m.AssertNumberOfCalls(t, "Log", 6)
}
