// Package config provides configuration management utilities for the Space UI panel,
// including version information, logging levels, database paths, and environment variable handling.
package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

//go:embed version
var version string

//go:embed name
var name string

// LogLevel represents the logging level for the application.
type LogLevel string

// Logging level constants
const (
	Debug   LogLevel = "debug"
	Info    LogLevel = "info"
	Notice  LogLevel = "notice"
	Warning LogLevel = "warning"
	Error   LogLevel = "error"
)

// GetVersion returns the version string of the Space UI application.
func GetVersion() string {
	return strings.TrimSpace(version)
}

// GetName returns the name of the Space UI application.
func GetName() string {
	return strings.TrimSpace(name)
}

// GetLogLevel returns the current logging level based on environment variables or defaults to Info.
func GetLogLevel() LogLevel {
	if IsDebug() {
		return Debug
	}
	logLevel := os.Getenv("SPACE_UI_LOG_LEVEL")
	if logLevel == "" {
		return Info
	}
	return LogLevel(logLevel)
}

// IsDebug returns true if debug mode is enabled via the SPACE_UI_DEBUG environment variable.
func IsDebug() bool {
	return os.Getenv("SPACE_UI_DEBUG") == "true"
}

// GetBinFolderPath returns the path to the binary folder, defaulting to "bin" if not set via SPACE_UI_BIN_FOLDER.
func GetBinFolderPath() string {
	binFolderPath := os.Getenv("SPACE_UI_BIN_FOLDER")
	if binFolderPath == "" {
		binFolderPath = "bin"
	}
	return binFolderPath
}

func getBaseDir() string {
	exePath, err := os.Executable()
	if err != nil {
		return "."
	}
	exeDir := filepath.Dir(exePath)
	exeDirLower := strings.ToLower(filepath.ToSlash(exeDir))
	if strings.Contains(exeDirLower, "/appdata/local/temp/") || strings.Contains(exeDirLower, "/go-build") {
		wd, err := os.Getwd()
		if err != nil {
			return "."
		}
		return wd
	}
	return exeDir
}

// GetDBFolderPath returns the path to the database folder based on environment variables or platform defaults.
func GetDBFolderPath() string {
	dbFolderPath := os.Getenv("SPACE_UI_DB_FOLDER")
	if dbFolderPath != "" {
		return dbFolderPath
	}
	if runtime.GOOS == "windows" {
		return getBaseDir()
	}
	return "/etc/space-ui"
}

// GetDBPath returns the full path to the database file.
func GetDBPath() string {
	return fmt.Sprintf("%s/%s.db", GetDBFolderPath(), GetName())
}

// GetSiteFolder returns the folder of the administrator's own website, which
// the panel shows on port 443 instead of the one it makes. It sits beside the
// database, so an update, which replaces the panel's own folder, leaves it.
func GetSiteFolder() string {
	return filepath.Join(GetDBFolderPath(), "site")
}

// GetCertFolder returns the folder where the panel keeps the certificates it
// obtains itself and its accounts with the certificate authorities.
func GetCertFolder() string {
	return filepath.Join(GetDBFolderPath(), "certs")
}

// GetSiteSocket returns the socket the website listens on when Xray holds
// port 443: a REALITY inbound passes everyone who is not a client to it.
func GetSiteSocket() string {
	if s := os.Getenv("SPACE_UI_SITE_SOCKET"); s != "" {
		return s
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(getBaseDir(), "site.sock")
	}
	return "/run/space-ui/site.sock"
}

// GetLogFolder returns the path to the log folder based on environment variables or platform defaults.
func GetLogFolder() string {
	logFolderPath := os.Getenv("SPACE_UI_LOG_FOLDER")
	if logFolderPath != "" {
		return logFolderPath
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(".", "log")
	}
	return "/var/log/space-ui"
}
