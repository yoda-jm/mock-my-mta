package log

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		level    string
		expected LogLevel
	}{
		{"DEBUG", DEBUG},
		{"INFO", INFO},
		{"WARNING", WARNING},
		{"ERROR", ERROR},
		{"FATAL", FATAL},
	}

	for _, test := range tests {
		t.Run(test.level, func(t *testing.T) {
			actual := ParseLogLevel(test.level)
			if actual != test.expected {
				t.Errorf("expected %v, got %v", test.expected, actual)
			}
		})
	}
}

func TestSetMinimumLogLevel(t *testing.T) {
	for _, level := range []LogLevel{DEBUG, INFO, WARNING, ERROR, FATAL} {
		SetMinimumLogLevel(level)
		if minimumLogLevel != level {
			t.Errorf("expected %v, got %v", level, minimumLogLevel)
		}
	}
}

func TestParseLogLevelInvalid(t *testing.T) {
	// check that panic is called (only because we reimplemented os.Exit in the test)
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic")
		}
	}()
	// override os.Exit
	var exitCode int
	privateExitFunc = func(code int) {
		exitCode = code
	}

	_ = ParseLogLevel("invalid")
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %v", exitCode)

	}
}

func TestLogf(t *testing.T) {
	// test that the log message is printed
	var logMessage string
	privatePrintfFunc = func(format string, args ...interface{}) {
		logMessage = fmt.Sprintf(format, args...)
	}
	var exitCode int
	privateExitFunc = func(code int) {
		exitCode = code
	}
	type testCase struct {
		minumumLogLevel         string
		logLevelsWithMessage    []string
		logLevelsWithoutMessage []string
	}
	testCases := []testCase{
		{
			minumumLogLevel:         "DEBUG",
			logLevelsWithMessage:    []string{"DEBUG", "INFO", "WARNING", "ERROR", "FATAL"},
			logLevelsWithoutMessage: []string{},
		},
		{
			minumumLogLevel:         "INFO",
			logLevelsWithMessage:    []string{"INFO", "WARNING", "ERROR", "FATAL"},
			logLevelsWithoutMessage: []string{"DEBUG"},
		},
		{
			minumumLogLevel:         "WARNING",
			logLevelsWithMessage:    []string{"WARNING", "ERROR", "FATAL"},
			logLevelsWithoutMessage: []string{"DEBUG", "INFO"},
		},
		{
			minumumLogLevel:         "ERROR",
			logLevelsWithMessage:    []string{"ERROR", "FATAL"},
			logLevelsWithoutMessage: []string{"DEBUG", "INFO", "WARNING"},
		},
		{
			minumumLogLevel:         "FATAL",
			logLevelsWithMessage:    []string{"FATAL"},
			logLevelsWithoutMessage: []string{"DEBUG", "INFO", "WARNING", "ERROR"},
		},
	}
	for _, testCase := range testCases {
		SetMinimumLogLevel(ParseLogLevel(testCase.minumumLogLevel))
		for _, logLevelString := range testCase.logLevelsWithMessage {
			exitCode = 0
			logMessage = ""
			message := "test log message"
			logLevel := ParseLogLevel(logLevelString)
			Logf(logLevel, message)
			_, filename, line, _ := runtime.Caller(0)
			location := fmt.Sprintf("%s:%d", filepath.Base(filename), line-1)
			// check that message starts with the location of the call
			prefix := fmt.Sprintf("%s: %s:", location, strings.ToLower(logLevelString))
			if !strings.HasPrefix(logMessage, prefix) {
				t.Errorf("expected log message to start with %q, got %q", prefix, logMessage)
			}
			// check that message ends with the message
			if !strings.HasSuffix(logMessage, message) {
				t.Errorf("expected log message to end with %q, got %q", message, logMessage)
			}
			if logLevelString == "FATAL" {
				if exitCode != 1 {
					t.Errorf("expected exit code 1, got %v", exitCode)
				}
			} else {
				if exitCode != 0 {
					t.Errorf("expected exit code 0, got %v", exitCode)
				}
			}
		}
		for _, logLevelString := range testCase.logLevelsWithoutMessage {
			logMessage = ""
			message := "test log message"
			logLevel := ParseLogLevel(logLevelString)
			Logf(logLevel, message)
			if logMessage != "" {
				t.Errorf("expected empty log message for minimum log level %v and log level %v, got %q", testCase.minumumLogLevel, logLevelString, logMessage)
			}
		}
	}
}
