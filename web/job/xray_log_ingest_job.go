package job

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/disp911/space-ui/v3/database"
	"github.com/disp911/space-ui/v3/database/model"
	"github.com/disp911/space-ui/v3/logger"
	"github.com/disp911/space-ui/v3/xray"

	"gorm.io/gorm/clause"
)

// xrayLogTimeFormat is the timestamp Xray writes at the start of every access
// log line.
const xrayLogTimeFormat = "2006/01/02 15:04:05.999999"

// xrayLogInsertBatchSize bounds how many rows go into a single INSERT.
const xrayLogInsertBatchSize = 500

// XrayLogIngestJob parses Xray's access log into the log database.
//
// It tails two files: Xray's own access log, which carries the newest lines,
// and today's persistent copy, which CheckClientIpJob folds the access log
// into roughly once an hour before truncating it. Reading both is what makes
// ingestion lossless - the lines written between a fold and the truncation
// that follows it only ever exist in the persistent copy.
//
// Rows are inserted against a unique index with ON CONFLICT DO NOTHING, so
// reading the same line from both files costs nothing. That also makes the
// byte offsets below a pure fast path rather than state that has to survive:
// a restart or a truncation simply re-reads a file and re-inserts rows that
// are already there.
type XrayLogIngestJob struct {
	rawOffset        int64
	persistentOffset int64
	persistentDay    string
	backfilled       bool
}

// NewXrayLogIngestJob creates a new Xray access log ingestion job.
func NewXrayLogIngestJob() *XrayLogIngestJob {
	return new(XrayLogIngestJob)
}

// Run is an interface method of the Job interface
func (j *XrayLogIngestJob) Run() {
	if database.GetLogDB() == nil {
		return
	}

	if !j.backfilled {
		j.backfilled = true
		j.backfill()
	}

	if accessLogPath, err := xray.GetAccessLogPath(); err == nil && accessLogPath != "" && accessLogPath != "none" {
		j.ingest(accessLogPath, &j.rawOffset)
	}

	today := time.Now().Format(xray.AccessPersistentLogDateFormat)
	if j.persistentDay != today {
		j.persistentDay = today
		j.persistentOffset = 0
	}
	j.ingest(xray.GetAccessPersistentLogPath(), &j.persistentOffset)
}

// backfill imports every persistent log file still on disk. It runs once per
// panel start so that the days already archived show up in the viewer right
// away instead of the history taking a week to rebuild.
func (j *XrayLogIngestJob) backfill() {
	matches, err := filepath.Glob(xray.AccessPersistentLogGlob())
	if err != nil {
		logger.Warning("Failed to list persistent access logs:", err)
		return
	}

	for _, path := range matches {
		var offset int64
		j.ingest(path, &offset)
	}
}

// ingest stores whatever has been appended to path since the last run. A file
// shorter than the offset has been truncated, so reading starts over.
func (j *XrayLogIngestJob) ingest(path string, offset *int64) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	if info.Size() < *offset {
		*offset = 0
	}
	if info.Size() == *offset {
		return
	}

	file, err := os.Open(path)
	if err != nil {
		logger.Warning("Failed to open Xray log for ingestion:", path, "-", err)
		return
	}
	defer file.Close()

	if _, err := file.Seek(*offset, io.SeekStart); err != nil {
		logger.Warning("Failed to seek Xray log:", path, "-", err)
		return
	}

	chunk, err := io.ReadAll(file)
	if err != nil {
		logger.Warning("Failed to read Xray log:", path, "-", err)
		return
	}

	// A trailing partial line is left for the next run, so a line is never
	// parsed before Xray has finished writing it.
	end := bytes.LastIndexByte(chunk, '\n')
	if end < 0 {
		return
	}
	consumed := chunk[:end+1]

	entries := make([]*model.XrayLogEntry, 0, 256)
	for _, line := range strings.Split(string(consumed), "\n") {
		if entry, ok := parseXrayLogLine(line); ok {
			entries = append(entries, entry)
		}
	}

	if len(entries) > 0 {
		err = database.GetLogDB().
			Clauses(clause.OnConflict{DoNothing: true}).
			CreateInBatches(entries, xrayLogInsertBatchSize).Error
		if err != nil {
			logger.Warning("Failed to store Xray log entries:", err)
			return
		}
	}

	*offset += int64(len(consumed))
}

// parseXrayLogLine turns one access log line into a row. ok is false for
// blank lines, the panel's own api traffic, and anything without a parsable
// timestamp.
func parseXrayLogLine(line string) (*model.XrayLogEntry, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.Contains(line, "api -> api") {
		return nil, false
	}

	parts := strings.Fields(line)
	if len(parts) < 2 {
		return nil, false
	}

	dateTime, err := time.ParseInLocation(xrayLogTimeFormat, parts[0]+" "+parts[1], time.Local)
	if err != nil {
		return nil, false
	}

	entry := &model.XrayLogEntry{
		Day:       dateTime.Format(xray.AccessPersistentLogDateFormat),
		Timestamp: dateTime.UnixMicro(),
	}

	for i, part := range parts {
		switch {
		case part == "from" && i+1 < len(parts):
			entry.FromAddress = strings.TrimLeft(parts[i+1], "/")
		case part == "accepted" && i+1 < len(parts):
			entry.ToAddress = strings.TrimLeft(parts[i+1], "/")
		case part == "email:" && i+1 < len(parts):
			entry.Email = parts[i+1]
		case strings.HasPrefix(part, "["):
			entry.Inbound = part[1:]
		case strings.HasSuffix(part, "]"):
			entry.Outbound = part[:len(part)-1]
		}
	}

	return entry, true
}
