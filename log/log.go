package log

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARNING
	ERROR
	FATAL
)

const (
	// Set the minimum log level here
	//MinimumLogLevel = DEBUG
	MinimumLogLevel = INFO
)

func getCallerInfo() string {
	_, file, line, ok := runtime.Caller(2) // Get the caller of the customLog function (2 levels up)
	if ok {
		return fmt.Sprintf("%s:%d", filepath.Base(file), line)
	}
	return "unknown:0"
}

// customLog is a custom logging function that takes a log level and log message.
func Logf(level LogLevel, format string, args ...interface{}) {
	if level < MinimumLogLevel {
		return
	}

	var logPrefix string
	switch level {
	case DEBUG:
		logPrefix = "debug:"
	case INFO:
		logPrefix = "info:"
	case WARNING:
		logPrefix = "warning:"
	case ERROR:
		logPrefix = "error:"
	case FATAL:
		logPrefix = "fatal:"
	}

	callerInfo := getCallerInfo()
	fullMsg := fmt.Sprintf("%s: %s %s", callerInfo, logPrefix, format)
	log.Printf(fullMsg, args...)

	if level == FATAL {
		os.Exit(1)
	}
}
