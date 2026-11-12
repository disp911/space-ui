package job

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/disp911/space-ui/v3/database"
	"github.com/disp911/space-ui/v3/database/model"
	"github.com/disp911/space-ui/v3/logger"
	"github.com/disp911/space-ui/v3/xray"
)

// ClearLogsJob clears old log files to prevent disk space issues.
type ClearLogsJob struct{}

// NewClearLogsJob creates a new log cleanup job instance.
func NewClearLogsJob() *ClearLogsJob {
	return new(ClearLogsJob)
}

// ensureFileExists creates the necessary directories and file if they don't exist
func ensureFileExists(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	file.Close()
	return nil
}

// Here Run is an interface method of the Job interface
func (j *ClearLogsJob) Run() {
	logFiles := []string{xray.GetIPLimitLogPath(), xray.GetIPLimitBannedLogPath()}
	logFilesPrev := []string{xray.GetIPLimitBannedPrevLogPath()}

	// Ensure all log files and their paths exist
	for _, path := range append(logFiles, logFilesPrev...) {
		if err := ensureFileExists(path); err != nil {
			logger.Warning("Failed to ensure log file exists:", path, "-", err)
		}
	}

	// Clear log files and copy to previous logs
	for i := range len(logFiles) {
		if i > 0 {
			// Copy to previous logs
			logFilePrev, err := os.OpenFile(logFilesPrev[i-1], os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
			if err != nil {
				logger.Warning("Failed to open previous log file for writing:", logFilesPrev[i-1], "-", err)
				continue
			}

			logFile, err := os.OpenFile(logFiles[i], os.O_RDONLY, 0644)
			if err != nil {
				logger.Warning("Failed to open current log file for reading:", logFiles[i], "-", err)
				logFilePrev.Close()
				continue
			}

			_, err = io.Copy(logFilePrev, logFile)
			if err != nil {
				logger.Warning("Failed to copy log file:", logFiles[i], "to", logFilesPrev[i-1], "-", err)
			}

			logFile.Close()
			logFilePrev.Close()
		}

		err := os.Truncate(logFiles[i], 0)
		if err != nil {
			logger.Warning("Failed to truncate log file:", logFiles[i], "-", err)
		}
	}

	pruneOldAccessPersistentLogs()
	pruneOldXrayLogRows()
}

// pruneOldAccessPersistentLogs deletes per-day persistent access log files
// older than xray.AccessPersistentLogRetentionDays, so they don't
// accumulate on disk forever.
func pruneOldAccessPersistentLogs() {
	matches, err := filepath.Glob(xray.AccessPersistentLogGlob())
	if err != nil {
		logger.Warning("Failed to list persistent access logs:", err)
		return
	}

	cutoff := time.Now().AddDate(0, 0, -xray.AccessPersistentLogRetentionDays)
	for _, path := range matches {
		date, ok := xray.ParseAccessPersistentLogDate(path)
		if !ok || !date.Before(cutoff) {
			continue
		}
		if err := os.Remove(path); err != nil {
			logger.Warning("Failed to remove old persistent access log:", path, "-", err)
		}
	}
}

// pruneOldXrayLogRows drops access log rows past the retention window. The
// window matches the one used for the files so the viewer's date list and the
// files on disk stay in step.
func pruneOldXrayLogRows() {
	db := database.GetLogDB()
	if db == nil {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -xray.AccessPersistentLogRetentionDays).UnixMicro()
	if err := db.Where("timestamp < ?", cutoff).Delete(&model.XrayLogEntry{}).Error; err != nil {
		logger.Warning("Failed to prune old Xray log rows:", err)
	}
}
