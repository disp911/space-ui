package xray

import (
	"regexp"
	"runtime"
	"strings"

	"github.com/disp911/space-ui/v3/logger"
)

// NewLogWriter returns a new LogWriter for processing Xray log output.
func NewLogWriter() *LogWriter {
	return &LogWriter{}
}

// LogWriter processes and filters log output from the Xray process, handling crash detection and message filtering.
type LogWriter struct {
	lastLine string
}

// Write processes and filters log output from the Xray process, handling crash detection and message filtering.
func (lw *LogWriter) Write(m []byte) (n int, err error) {
	crashRegex := regexp.MustCompile(`(?i)(panic|exception|stack trace|fatal error)`)

	// Convert the data to a string
	message := strings.TrimSpace(string(m))
	msgLowerAll := strings.ToLower(message)

	// Suppress noisy Windows process-kill signal that surfaces as exit status 1
	if runtime.GOOS == "windows" && strings.Contains(msgLowerAll, "exit status 1") {
		return len(m), nil
	}

	// Check if the message contains a crash
	if crashRegex.MatchString(message) {
		logger.Debug("Core crash detected:\n", message)
		lw.lastLine = message
		err1 := writeCrashReport(m)
		if err1 != nil {
			logger.Error("Unable to write crash report:", err1)
		}
		return len(m), nil
	}

	regex := regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}\.\d{6}) \[([^\]]+)\] (.+)$`)
	messages := strings.SplitSeq(message, "\n")

	for msg := range messages {
		matches := regex.FindStringSubmatch(msg)

		// A line Xray gave a level keeps it: a single connection that failed,
		// e.g. a visitor REALITY passed on to the website, is Info to Xray, not
		// an error of the panel's.
		if len(matches) > 3 {
			level := matches[2]
			msgBody := matches[3]
			switch level {
			case "Debug":
				logger.Debug("XRAY: " + msgBody)
			case "Info":
				logger.Info("XRAY: " + msgBody)
			case "Warning":
				logger.Warning("XRAY: " + msgBody)
			case "Error":
				logger.Error("XRAY: " + msgBody)
			default:
				logger.Debug("XRAY: " + msg)
			}
			lw.lastLine = ""
		} else if msg != "" {
			// A line without a level, e.g. from Xray's start, is an error when
			// it says something failed.
			if strings.Contains(strings.ToLower(msg), "failed") {
				logger.Error("XRAY: " + msg)
			} else {
				logger.Debug("XRAY: " + msg)
			}
			lw.lastLine = msg
		}
	}

	return len(m), nil
}
