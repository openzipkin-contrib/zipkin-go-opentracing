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
	sl.mutex.Lock()
	defer sl.mutex.Unlock()
	if err == sl.lastError && time.Since(sl.lastErrorTime) < sl.logErrorInterval {
		return
	}
	sl.logger.Log("err", err.Error())
	sl.lastError = err
	sl.lastErrorTime = time.Now()
}

// Fixed makes the stateLogger understand that the state is fixed, and when
// the next error will occur, it will log it.
func (sl *StateLogger) Fixed(keyVal ...interface{}) {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()
	if sl.logErrorInterval == 0 || sl.lastError == nil {
		return
	}
	sl.logger.Log(keyVal...)
	sl.lastError = nil
}
