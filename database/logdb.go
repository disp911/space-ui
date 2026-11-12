package database

import (
	"os"
	"path/filepath"

	"github.com/disp911/space-ui/v3/config"
	"github.com/disp911/space-ui/v3/database/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var logDB *gorm.DB

// GetLogDBPath returns the path of the Xray access log database. It sits next
// to the log files instead of in space-ui.db so that panel backups stay small and
// the constant stream of log writes never contends with configuration writes.
func GetLogDBPath() string {
	return filepath.Join(config.GetLogFolder(), "xraylogs.db")
}

// InitLogDB opens the Xray access log database and migrates its schema.
func InitLogDB() error {
	if err := os.MkdirAll(config.GetLogFolder(), 0o750); err != nil {
		return err
	}

	var gormLogger logger.Interface
	if config.IsDebug() {
		gormLogger = logger.Default
	} else {
		gormLogger = logger.Discard
	}

	db, err := gorm.Open(sqlite.Open(GetLogDBPath()), &gorm.Config{Logger: gormLogger})
	if err != nil {
		return err
	}

	// WAL lets the panel read pages of the log while the ingest job is still
	// appending to it; NORMAL trades an fsync per transaction for throughput,
	// which is the right side of that trade for logs.
	if err := db.Exec("PRAGMA journal_mode=WAL;").Error; err != nil {
		return err
	}
	if err := db.Exec("PRAGMA synchronous=NORMAL;").Error; err != nil {
		return err
	}

	if err := db.AutoMigrate(&model.XrayLogEntry{}); err != nil {
		return err
	}

	logDB = db
	return nil
}

// GetLogDB returns the Xray access log database, or nil if it failed to open.
// Callers must handle nil: the panel deliberately keeps running without it.
func GetLogDB() *gorm.DB {
	return logDB
}
