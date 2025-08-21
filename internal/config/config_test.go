package config

import (
	"testing"

	"github.com/company/log-trimmer/internal/logger"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.IndexPattern != "vector-*" {
		t.Errorf("Expected default IndexPattern 'vector-*', got %s", cfg.IndexPattern)
	}
	if cfg.SkipTLS != true {
		t.Errorf("Expected SkipTLS to be true by default")
	}
	if cfg.Logger.Level != logger.LevelInfo {
		t.Errorf("Expected default log level 'info', got %s", cfg.Logger.Level)
	}
}

func TestValidate(t *testing.T) {
	// Valid config
	cfg := DefaultConfig()
	cfg.ESHost = "https://localhost:9200"
	cfg.MaxAge = "7d"

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Expected valid config to pass validation, got: %v", err)
	}

	// Missing host
	cfg2 := DefaultConfig()
	cfg2.MaxAge = "7d"
	err = cfg2.Validate()
	if err == nil {
		t.Error("Expected error for missing host")
	}

	// Missing criteria
	cfg3 := DefaultConfig()
	cfg3.ESHost = "https://localhost:9200"
	err = cfg3.Validate()
	if err == nil {
		t.Error("Expected error for missing max-age/max-size")
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		wantErr  bool
	}{
		{"1KB", 1024, false},
		{"1MB", 1024 * 1024, false},
		{"1GB", 1024 * 1024 * 1024, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		result, err := parseSize(tt.input)
		if tt.wantErr && err == nil {
			t.Errorf("Expected error for input '%s'", tt.input)
		}
		if !tt.wantErr && result != tt.expected {
			t.Errorf("For input '%s', expected %d, got %d", tt.input, tt.expected, result)
		}
	}
}
