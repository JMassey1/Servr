package consts

import (
	"os"
	"path/filepath"
	"time"

	CharmLog "github.com/charmbracelet/log"
)

const (
	SFTP_PORT int = 2022
)

var (
	SFTPRoot string

	LOGGER_CONFIG = CharmLog.Options{
		ReportTimestamp: true,
		TimeFormat:      time.Kitchen,
		Prefix:          "SFTP Service 📁",
	}
)

func init() {
	logger := CharmLog.NewWithOptions(os.Stderr, LOGGER_CONFIG)
	logger.SetPrefix("SFTP Service 📁 (consts.init)")
	logger.Info("Initializing SFTP root directory")

	root := os.Getenv("SFTP_ROOT")
	if root == "" {
		logger.Fatal("SFTP_ROOT environment variable must be set")
	}
	logger.Info("Using SFTP_ROOT env variable", "path", root)

	abs, err := filepath.Abs(root)
	if err != nil {
		logger.Fatal("Failed to get absolute path", "path", root, "error", err)
	}
	SFTPRoot = filepath.Clean(abs)

	logger.Info("Checking SFTP root directory", "path", SFTPRoot)
	info, err := os.Stat(SFTPRoot)
	if os.IsNotExist(err) {
		logger.Fatal("Root directory does not exist", "path", SFTPRoot)
	}
	if err != nil || !info.IsDir() {
		logger.Fatal("Root is not a directory", "path", SFTPRoot, "error", err)
	}

	logger.Info("Root directory initialized", "path", SFTPRoot)
}
