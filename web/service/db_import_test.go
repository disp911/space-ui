package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/disp911/space-ui/v3/config"
	"github.com/disp911/space-ui/v3/database"
	"github.com/disp911/space-ui/v3/database/model"
	"github.com/disp911/space-ui/v3/logger"

	"github.com/op/go-logging"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// importTestDB opens a panel database in a folder of its own, with one inbound.
func importTestDB(t *testing.T) {
	t.Helper()
	logger.InitLogger(logging.CRITICAL)
	t.Setenv("SPACE_UI_DB_FOLDER", t.TempDir())
	if err := database.InitDB(config.GetDBPath()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.CloseDB() })
	if err := database.GetDB().Create(&model.Inbound{Remark: "kept", Port: 1001, Protocol: "vless", Tag: "in-1001", Settings: "{}"}).Error; err != nil {
		t.Fatal(err)
	}
}

func inboundRemarks(t *testing.T) []string {
	t.Helper()
	var remarks []string
	if err := database.GetDB().Model(&model.Inbound{}).Order("id").Pluck("remark", &remarks).Error; err != nil {
		t.Fatalf("the panel's database: %v", err)
	}
	return remarks
}

// sqliteFile makes an SQLite database at a new path with the statements.
func sqliteFile(t *testing.T, statements ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "upload.db")
	gdb, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range statements {
		if err := gdb.Exec(s).Error; err != nil {
			t.Fatal(s, err)
		}
	}
	sqlDB, _ := gdb.DB()
	sqlDB.Close()
	return path
}

func prepare(t *testing.T, path string) (string, error) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	return new(ServerService).PrepareDBImport(f)
}

// leftBeside is the files beside the panel's database other than it and its WAL files.
func leftBeside(t *testing.T) []string {
	t.Helper()
	entries, _ := os.ReadDir(config.GetDBFolderPath())
	var left []string
	base := filepath.Base(config.GetDBPath())
	for _, e := range entries {
		if n := e.Name(); n != base && n != base+"-wal" && n != base+"-shm" {
			left = append(left, n)
		}
	}
	return left
}

// TestDBImportRestoresABackup checks a backup made by the panel coming back
// in place of the database, nothing left beside it.
func TestDBImportRestoresABackup(t *testing.T) {
	importTestDB(t)
	data, err := new(ServerService).GetDb()
	if err != nil {
		t.Fatal(err)
	}
	backup := filepath.Join(t.TempDir(), "space-ui.db")
	if err := os.WriteFile(backup, data, 0o600); err != nil {
		t.Fatal(err)
	}
	database.GetDB().Create(&model.Inbound{Remark: "after", Port: 1002, Protocol: "vless", Tag: "in-1002", Settings: "{}"})

	path, err := prepare(t, backup)
	if err != nil {
		t.Fatal(err)
	}
	QueueDBImport(path)
	ApplyQueuedDBImport()
	if e := LastDBImportError(); e != "" {
		t.Fatal(e)
	}
	if r := inboundRemarks(t); len(r) != 1 || r[0] != "kept" {
		t.Fatalf("inbounds after the import: %v", r)
	}
	if left := leftBeside(t); len(left) != 0 {
		t.Fatalf("left beside the database: %v", left)
	}
	// Nothing queued, nothing happens.
	ApplyQueuedDBImport()
	if r := inboundRemarks(t); len(r) != 1 {
		t.Fatalf("inbounds after an empty run: %v", r)
	}
}

// TestDBImportRefusesOtherDatabases checks that only a panel's database is
// taken: another program's would come up as an empty panel with the default user.
func TestDBImportRefusesOtherDatabases(t *testing.T) {
	importTestDB(t)
	for name, statements := range map[string][]string{
		"another program's": {"CREATE TABLE notes (id INTEGER)"},
		"without settings":  {"CREATE TABLE users (id INTEGER, username TEXT)", "INSERT INTO users VALUES (1, 'a')", "CREATE TABLE inbounds (id INTEGER)"},
		"without a user":    {"CREATE TABLE users (id INTEGER)", "CREATE TABLE inbounds (id INTEGER)", "CREATE TABLE settings (id INTEGER)"},
	} {
		if _, err := prepare(t, sqliteFile(t, statements...)); err == nil || !strings.Contains(err.Error(), "not a panel database") {
			t.Errorf("%s: %v", name, err)
		}
	}
	if left := leftBeside(t); len(left) != 0 {
		t.Fatalf("left beside the database: %v", left)
	}
	if r := inboundRemarks(t); len(r) != 1 {
		t.Fatalf("inbounds: %v", r)
	}
}

// TestDBImportKeepsTheDatabaseOnFailure checks a database that passes the
// checks but does not open: the current one comes back, open, and the error
// is kept for the window.
func TestDBImportKeepsTheDatabaseOnFailure(t *testing.T) {
	importTestDB(t)
	// A view where the panel makes a table: the database opens, its tables do not.
	path, err := prepare(t, sqliteFile(t,
		"CREATE TABLE users (id INTEGER PRIMARY KEY, username TEXT, password TEXT)",
		"INSERT INTO users (username, password) VALUES ('a', 'b')",
		"CREATE TABLE inbounds (id INTEGER PRIMARY KEY)",
		"CREATE TABLE settings (id INTEGER PRIMARY KEY)",
		"CREATE VIEW client_traffics AS SELECT 1 AS id",
	))
	if err != nil {
		t.Fatal(err)
	}
	QueueDBImport(path)
	ApplyQueuedDBImport()
	if e := LastDBImportError(); e == "" {
		t.Fatal("the failed import left no error")
	}
	if r := inboundRemarks(t); len(r) != 1 || r[0] != "kept" {
		t.Fatalf("inbounds after the failed import: %v", r)
	}
	if left := leftBeside(t); len(left) != 0 {
		t.Fatalf("left beside the database: %v", left)
	}
}
