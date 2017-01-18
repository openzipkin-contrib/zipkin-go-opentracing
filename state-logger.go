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
func (se *StateLogger) LogError(err error) {
	se.mutex.Lock()
	defer se.mutex.Unlock()
	if err == se.lastError && time.Since(se.lastErrorTime) < se.logErrorInterval {
		return
	}
	se.logger.Log("err", err.Error())
	se.lastError = err
	se.lastErrorTime = time.Now()
}

// Fixed makes the stateLogger understand that the state is fixed, and when
// the next error will occur, it will log it.
func (se *StateLogger) Fixed(keyVal ...interface{}) {
	se.mutex.Lock()
	defer se.mutex.Unlock()
	if se.logErrorInterval == 0 || se.lastError == nil {
		return
	}
	se.logger.Log(keyVal...)
	se.lastError = nil
}
