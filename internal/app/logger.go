package app

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

var logger *slog.Logger
var loggerOnce sync.Once

func Logger() *slog.Logger {
	loggerOnce.Do(initLogger)
	return logger
}

func initLogger() {
	file, err := openLogFile()
	if err != nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug, AddSource: false, ReplaceAttr: nil}))
		return
	}

	handler := slog.NewTextHandler(file, &slog.HandlerOptions{Level: slog.LevelDebug, AddSource: false, ReplaceAttr: nil})
	logger = slog.New(handler)
}

func openLogFile() (*os.File, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return os.OpenFile("debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	}

	path := filepath.Join(cwd, "debug.log")
	return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
}
