package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/lupguo/linkstash/app/infra/config"
)

// Setup initializes the global slog logger based on the given LogConfig.
// It writes to stdout and optionally to a log file (when cfg.File is set).
// The returned cleanup function should be deferred to close the log file.
func Setup(cfg config.LogConfig) (cleanup func(), err error) {
	// Parse log level
	level := parseLevel(cfg.Level)

	// Build writers: always include stdout
	writers := []io.Writer{os.Stdout}
	var logFile *os.File

	if cfg.File != "" {
		dir := filepath.Dir(cfg.File)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create log directory %s: %w", dir, err)
		}
		logFile, err = os.OpenFile(cfg.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("open log file %s: %w", cfg.File, err)
		}
		writers = append(writers, logFile)
	}

	w := io.MultiWriter(writers...)

	// Build handler based on format
	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if strings.EqualFold(cfg.Format, "json") {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}

	slog.SetDefault(slog.New(handler))

	cleanup = func() {
		if logFile != nil {
			logFile.Close()
		}
	}
	return cleanup, nil
}

// parseLevel converts a string level name to slog.Level.
func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
