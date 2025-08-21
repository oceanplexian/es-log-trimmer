package logger

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Level != LevelInfo {
		t.Errorf("Expected default level 'info', got %s", cfg.Level)
	}

	if cfg.Format != "console" {
		t.Errorf("Expected default format 'console', got %s", cfg.Format)
	}

	if cfg.Output != "stdout" {
		t.Errorf("Expected default output 'stdout', got %s", cfg.Output)
	}

	if cfg.EnableFile != false {
		t.Errorf("Expected EnableFile to be false by default")
	}

	if cfg.FilePath != "/var/log/log-trimmer/app.log" {
		t.Errorf("Expected default file path, got %s", cfg.FilePath)
	}
}

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   error
	}{
		{
			name:   "default config",
			config: DefaultConfig(),
			want:   nil,
		},
		{
			name: "json format config",
			config: &Config{
				Level:      LevelDebug,
				Format:     "json",
				Output:     "stdout",
				EnableFile: false,
			},
			want: nil,
		},
		{
			name: "console format config",
			config: &Config{
				Level:      LevelWarn,
				Format:     "console",
				Output:     "stderr",
				EnableFile: false,
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(tt.config)

			if (err != nil) != (tt.want != nil) {
				t.Errorf("New() error = %v, want %v", err, tt.want)
				return
			}

			if err == nil && logger == nil {
				t.Errorf("Expected logger instance, got nil")
			}
		})
	}
}

func TestLoggerLevels(t *testing.T) {
	// Create a logger with JSON format and capture output
	var buf bytes.Buffer

	cfg := &Config{
		Level:      LevelDebug,
		Format:     "json",
		Output:     "stdout",
		EnableFile: false,
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Redirect structured logger output to our buffer
	logger.structured.SetOutput(&buf)

	tests := []struct {
		name      string
		logFunc   func()
		level     string
		component string
		operation string
		message   string
	}{
		{
			name: "info level",
			logFunc: func() {
				logger.Info("test-component", "test-operation", "test info message")
			},
			level:     "info",
			component: "test-component",
			operation: "test-operation",
			message:   "test info message",
		},
		{
			name: "debug level",
			logFunc: func() {
				logger.Debug("debug-component", "debug-operation", "test debug message")
			},
			level:     "debug",
			component: "debug-component",
			operation: "debug-operation",
			message:   "test debug message",
		},
		{
			name: "warn level",
			logFunc: func() {
				logger.Warn("warn-component", "warn-operation", "test warning message")
			},
			level:     "warning",
			component: "warn-component",
			operation: "warn-operation",
			message:   "test warning message",
		},
		{
			name: "error level",
			logFunc: func() {
				logger.Error("error-component", "error-operation", "test error message", nil)
			},
			level:     "error",
			component: "error-component",
			operation: "error-operation",
			message:   "test error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear buffer
			buf.Reset()

			// Call the log function
			tt.logFunc()

			// Parse JSON output
			var logEntry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
				t.Fatalf("Failed to parse JSON log output: %v\nOutput: %s", err, buf.String())
			}

			// Verify log entry fields
			if logEntry["level"] != tt.level {
				t.Errorf("Expected level %s, got %v", tt.level, logEntry["level"])
			}

			if logEntry["component"] != tt.component {
				t.Errorf("Expected component %s, got %v", tt.component, logEntry["component"])
			}

			if logEntry["operation"] != tt.operation {
				t.Errorf("Expected operation %s, got %v", tt.operation, logEntry["operation"])
			}

			if logEntry["message"] != tt.message {
				t.Errorf("Expected message %s, got %v", tt.message, logEntry["message"])
			}

			// Verify standard fields
			if logEntry["service"] != "log-trimmer" {
				t.Errorf("Expected service 'log-trimmer', got %v", logEntry["service"])
			}

			if logEntry["version"] != "1.0.0" {
				t.Errorf("Expected version '1.0.0', got %v", logEntry["version"])
			}

			// Verify timestamp exists and is valid
			if _, ok := logEntry["timestamp"]; !ok {
				t.Errorf("Expected timestamp field in log entry")
			}
		})
	}
}

func TestLoggerWithFields(t *testing.T) {
	var buf bytes.Buffer

	cfg := &Config{
		Level:      LevelInfo,
		Format:     "json",
		Output:     "stdout",
		EnableFile: false,
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	logger.structured.SetOutput(&buf)

	// Test with additional fields
	fields := map[string]interface{}{
		"index_name": "test-index",
		"size_bytes": 1234567,
		"doc_count":  1000,
	}

	logger.Info("elasticsearch", "delete", "Deleting index", fields)

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse JSON log output: %v", err)
	}

	// Verify additional fields are present
	if logEntry["index_name"] != "test-index" {
		t.Errorf("Expected index_name 'test-index', got %v", logEntry["index_name"])
	}

	// Note: JSON unmarshaling converts numbers to float64
	if logEntry["size_bytes"] != float64(1234567) {
		t.Errorf("Expected size_bytes 1234567, got %v", logEntry["size_bytes"])
	}

	if logEntry["doc_count"] != float64(1000) {
		t.Errorf("Expected doc_count 1000, got %v", logEntry["doc_count"])
	}
}

func TestLoggerErrorWithError(t *testing.T) {
	var buf bytes.Buffer

	cfg := &Config{
		Level:      LevelInfo,
		Format:     "json",
		Output:     "stdout",
		EnableFile: false,
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	logger.structured.SetOutput(&buf)

	// Test error logging with actual error
	testErr := io.EOF
	logger.Error("test-component", "test-operation", "something failed", testErr)

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse JSON log output: %v", err)
	}

	// Verify error field is present
	if logEntry["error"] != "EOF" {
		t.Errorf("Expected error 'EOF', got %v", logEntry["error"])
	}

	// Verify message includes error
	message, ok := logEntry["message"].(string)
	if !ok {
		t.Fatalf("Expected message to be string, got %T", logEntry["message"])
	}

	if !strings.Contains(message, "EOF") {
		t.Errorf("Expected message to contain error, got %s", message)
	}
}

func TestLoggerSetLevel(t *testing.T) {
	logger, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test setting different levels
	levels := []LogLevel{LevelDebug, LevelInfo, LevelWarn, LevelError}

	for _, level := range levels {
		logger.SetLevel(level)

		expectedLogrusLevel, _ := logrus.ParseLevel(string(level))
		if logger.level != expectedLogrusLevel {
			t.Errorf("Expected level %v, got %v", expectedLogrusLevel, logger.level)
		}

		if logger.structured.Level != expectedLogrusLevel {
			t.Errorf("Expected structured logger level %v, got %v", expectedLogrusLevel, logger.structured.Level)
		}
	}
}

func TestLoggerConsoleOutput(t *testing.T) {
	cfg := &Config{
		Level:      LevelInfo,
		Format:     "console",
		Output:     "stdout",
		EnableFile: false,
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Just test that the method doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Logger.Info panicked: %v", r)
		}
	}()

	// Log a message (output goes to console, hard to capture in tests)
	logger.Info("test-component", "test-operation", "test console message")

	// If we reach here, the function completed without panicking
	t.Log("Console logging executed successfully")
}

func TestGetCallerInfo(t *testing.T) {
	funcName, fileName := getCallerInfo()

	// Just verify that the function returns non-empty strings
	// The actual caller info depends on runtime.Caller behavior which can vary
	if funcName == "" {
		t.Errorf("Expected non-empty function name, got empty string")
	}

	if fileName == "" {
		t.Errorf("Expected non-empty file name, got empty string")
	}

	t.Logf("Caller info: function=%s, file=%s", funcName, fileName)
}

// Benchmark tests
func BenchmarkLoggerInfo(b *testing.B) {
	cfg := &Config{
		Level:      LevelInfo,
		Format:     "json",
		Output:     "stdout",
		EnableFile: false,
	}

	logger, err := New(cfg)
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}

	// Discard output for benchmarking
	logger.structured.SetOutput(io.Discard)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark-component", "benchmark-operation", "benchmark message")
	}
}

func BenchmarkLoggerWithFields(b *testing.B) {
	cfg := &Config{
		Level:      LevelInfo,
		Format:     "json",
		Output:     "stdout",
		EnableFile: false,
	}

	logger, err := New(cfg)
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}

	logger.structured.SetOutput(io.Discard)

	fields := map[string]interface{}{
		"index":    "test-index",
		"size":     1234567,
		"docs":     1000,
		"duration": "500ms",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark-component", "benchmark-operation", "benchmark message", fields)
	}
}

func TestLoggerLevelFiltering(t *testing.T) {
	var buf bytes.Buffer

	cfg := &Config{
		Level:      LevelWarn, // Set to warn level to filter out info/debug
		Format:     "json",
		Output:     "stdout",
		EnableFile: false,
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	logger.structured.SetOutput(&buf)

	// Try to log at different levels
	logger.Debug("test", "test", "debug message")  // Should be filtered
	logger.Info("test", "test", "info message")    // Should be filtered
	logger.Warn("test", "test", "warning message") // Should be logged

	output := buf.String()

	// Only warning should be in output
	if strings.Contains(output, "debug message") {
		t.Errorf("Debug message should be filtered out")
	}

	if strings.Contains(output, "info message") {
		t.Errorf("Info message should be filtered out")
	}

	if !strings.Contains(output, "warning message") {
		t.Errorf("Warning message should be logged")
	}
}

// Test edge cases and error conditions
func TestLoggerEdgeCases(t *testing.T) {
	// Test with nil fields
	logger, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// These should not panic
	logger.Info("test", "test", "message with nil fields", nil)
	logger.Error("test", "test", "error with nil error", nil)

	// Test with empty strings
	logger.Info("", "", "")
	logger.Error("", "", "", nil)

	// Test Success method
	logger.Success("test", "test", "success message")
}

func TestLoggerFileOutput(t *testing.T) {
	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "logger_test_*.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	cfg := &Config{
		Level:      LevelInfo,
		Format:     "json",
		Output:     "stdout",
		EnableFile: true,
		FilePath:   tmpFile.Name(),
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger with file output: %v", err)
	}

	// Log a message
	logger.Info("file-test", "write", "test file logging")

	// Read the file contents
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Verify the message was written to file
	if !strings.Contains(string(content), "test file logging") {
		t.Errorf("Expected log file to contain 'test file logging', got: %s", string(content))
	}

	// Verify it's valid JSON
	var logEntry map[string]interface{}
	if err := json.Unmarshal(content, &logEntry); err != nil {
		t.Errorf("Log file should contain valid JSON: %v", err)
	}
}
