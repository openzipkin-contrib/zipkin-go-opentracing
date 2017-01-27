package zipkintracer

import (
	"fmt"
	"sync"
	"time"
)

var errNoError = fmt.Errorf("not an error")

// StateLogger is a Logger that logs error only if logErrorInterval have passed
// from the last error, or it is a different error than the last seen.
type StateLogger struct {
	logger           Logger
	logErrorInterval time.Duration
	lastError        error
	lastErrorTime    time.Time
	mutex            *sync.Mutex
}

// NewStateLogger creates a new stateLogger
func NewStateLogger(logger Logger, logErrorInterval time.Duration) *StateLogger {
	return &StateLogger{
		logger:           logger,
		logErrorInterval: logErrorInterval,
		lastError:        errNoError,
		mutex:            &sync.Mutex{},
	}
}

// LogError logs an error if it is different from the last seen error,
// or that logErrorInterval have passed since the last reported error.
func (sl *StateLogger) LogError(err error) {
	if sl.logErrorInterval != 0 {
		sl.mutex.Lock()
		defer sl.mutex.Unlock()
		if err == sl.lastError && time.Since(sl.lastErrorTime) < sl.logErrorInterval {
			return
		}
		sl.lastError = err
		sl.lastErrorTime = time.Now()
	}

	sl.logger.Log("err", err.Error())
}

// Fixed makes the stateLogger understand that the state is fixed, and when
// the next error will occur, it will log it.
func (sl *StateLogger) Fixed(keyVal ...interface{}) {
	// In case the interval is 0, do not log a success message
	if sl.logErrorInterval == 0 {
		return
	}

	sl.mutex.Lock()
	defer sl.mutex.Unlock()
	if sl.lastError == nil {
		return
	}
	sl.lastError = nil

	sl.logger.Log(keyVal...)
}
