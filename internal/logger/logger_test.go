package logger

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Level != LevelInfo {
		t.Errorf("Expected default level 'info', got %s", cfg.Level)
	}
	if cfg.Format != "console" {
		t.Errorf("Expected default format 'console', got %s", cfg.Format)
	}
}

func TestNewLogger(t *testing.T) {
	logger, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	if logger == nil {
		t.Fatal("Expected logger instance, got nil")
	}
}

func TestLoggerMethods(t *testing.T) {
	logger, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test that logging methods don't panic
	logger.Info("test", "operation", "test message")
	logger.Debug("test", "operation", "debug message")
	logger.Warn("test", "operation", "warn message")
	logger.Error("test", "operation", "error message", nil)
	logger.Success("test", "operation", "success message")
}

func TestSetLevel(t *testing.T) {
	logger, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test setting level doesn't panic
	logger.SetLevel(LevelDebug)
	logger.SetLevel(LevelError)
}
