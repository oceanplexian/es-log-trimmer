package config

import (
	"os"
	"testing"
	"time"

	"github.com/company/log-trimmer/internal/logger"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.ESHost != "" {
		t.Errorf("Expected empty ESHost, got %s", cfg.ESHost)
	}

	if cfg.IndexPattern != "vector-*" {
		t.Errorf("Expected default IndexPattern 'vector-*', got %s", cfg.IndexPattern)
	}

	if cfg.SkipTLS != true {
		t.Errorf("Expected SkipTLS to be true by default")
	}

	if cfg.DeleteIndexes != false {
		t.Errorf("Expected DeleteIndexes to be false by default")
	}

	if cfg.Logger.Level != logger.LevelInfo {
		t.Errorf("Expected default log level 'info', got %s", cfg.Logger.Level)
	}

	if cfg.Logger.Format != "console" {
		t.Errorf("Expected default log format 'console', got %s", cfg.Logger.Format)
	}
}

func TestLoadFromEnv(t *testing.T) {
	// Save original env vars
	originalVars := map[string]string{
		"ES_HOST":        os.Getenv("ES_HOST"),
		"ES_USERNAME":    os.Getenv("ES_USERNAME"),
		"ES_PASSWORD":    os.Getenv("ES_PASSWORD"),
		"MAX_SIZE":       os.Getenv("MAX_SIZE"),
		"MAX_AGE":        os.Getenv("MAX_AGE"),
		"INDEX_PATTERN":  os.Getenv("INDEX_PATTERN"),
		"DELETE_INDEXES": os.Getenv("DELETE_INDEXES"),
		"VERBOSE":        os.Getenv("VERBOSE"),
		"LOG_LEVEL":      os.Getenv("LOG_LEVEL"),
		"LOG_FORMAT":     os.Getenv("LOG_FORMAT"),
	}

	// Clean up after test
	defer func() {
		for key, value := range originalVars {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	// Set test environment variables
	testEnvVars := map[string]string{
		"ES_HOST":        "https://test.elasticsearch.com:9200",
		"ES_USERNAME":    "testuser",
		"ES_PASSWORD":    "testpass",
		"MAX_SIZE":       "100GB",
		"MAX_AGE":        "7d",
		"INDEX_PATTERN":  "test-*",
		"DELETE_INDEXES": "true",
		"VERBOSE":        "true",
		"LOG_LEVEL":      "debug",
		"LOG_FORMAT":     "json",
	}

	for key, value := range testEnvVars {
		os.Setenv(key, value)
	}

	cfg := DefaultConfig()
	cfg.LoadFromEnv()

	if cfg.ESHost != "https://test.elasticsearch.com:9200" {
		t.Errorf("Expected ESHost from env, got %s", cfg.ESHost)
	}

	if cfg.Username != "testuser" {
		t.Errorf("Expected Username from env, got %s", cfg.Username)
	}

	if cfg.Password != "testpass" {
		t.Errorf("Expected Password from env, got %s", cfg.Password)
	}

	if cfg.MaxSize != "100GB" {
		t.Errorf("Expected MaxSize from env, got %s", cfg.MaxSize)
	}

	if cfg.MaxAge != "7d" {
		t.Errorf("Expected MaxAge from env, got %s", cfg.MaxAge)
	}

	if cfg.IndexPattern != "test-*" {
		t.Errorf("Expected IndexPattern from env, got %s", cfg.IndexPattern)
	}

	if !cfg.DeleteIndexes {
		t.Errorf("Expected DeleteIndexes to be true from env")
	}

	if !cfg.Verbose {
		t.Errorf("Expected Verbose to be true from env")
	}

	if cfg.Logger.Level != logger.LevelDebug {
		t.Errorf("Expected Logger.Level to be debug from env, got %s", cfg.Logger.Level)
	}

	if cfg.Logger.Format != "json" {
		t.Errorf("Expected Logger.Format to be json from env, got %s", cfg.Logger.Format)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  func() *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config with max-age",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.ESHost = "https://localhost:9200"
				cfg.MaxAge = "7d"
				return cfg
			},
			wantErr: false,
		},
		{
			name: "valid config with max-size",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.ESHost = "https://localhost:9200"
				cfg.MaxSize = "100GB"
				return cfg
			},
			wantErr: false,
		},
		{
			name: "valid config with both max-age and max-size",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.ESHost = "https://localhost:9200"
				cfg.MaxAge = "30d"
				cfg.MaxSize = "500GB"
				return cfg
			},
			wantErr: false,
		},
		{
			name: "missing host",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.MaxAge = "7d"
				return cfg
			},
			wantErr: true,
			errMsg:  "elasticsearch host is required",
		},
		{
			name: "missing both max-age and max-size",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.ESHost = "https://localhost:9200"
				return cfg
			},
			wantErr: true,
			errMsg:  "must specify at least one of --max-size/MAX_SIZE or --max-age/MAX_AGE",
		},
		{
			name: "invalid max-size format",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.ESHost = "https://localhost:9200"
				cfg.MaxSize = "invalid"
				return cfg
			},
			wantErr: true,
			errMsg:  "invalid max-size format",
		},
		{
			name: "invalid max-age format",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.ESHost = "https://localhost:9200"
				cfg.MaxAge = "invalid"
				return cfg
			},
			wantErr: true,
			errMsg:  "invalid max-age format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.config()
			err := cfg.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errMsg)
					return
				}
				if err.Error() == "" {
					t.Errorf("Expected error message, got empty string")
					return
				}
				// Check if error message contains expected substring
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		wantErr  bool
	}{
		{"100B", 100, false},
		{"1KB", 1024, false},
		{"1MB", 1024 * 1024, false},
		{"1GB", 1024 * 1024 * 1024, false},
		{"1TB", 1024 * 1024 * 1024 * 1024, false},
		{"1.5GB", int64(1.5 * 1024 * 1024 * 1024), false},
		{"500MB", 500 * 1024 * 1024, false},
		{"invalid", 0, true},
		{"", 0, true},
		{"100XB", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseSize(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for input '%s', got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for input '%s': %v", tt.input, err)
				return
			}

			if result != tt.expected {
				t.Errorf("For input '%s', expected %d, got %d", tt.input, tt.expected, result)
			}
		})
	}
}

func TestParseAge(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"1s", 1 * time.Second, false},
		{"30m", 30 * time.Minute, false},
		{"24h", 24 * time.Hour, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"2w", 2 * 7 * 24 * time.Hour, false},
		{"invalid", 0, true},
		{"", 0, true},
		{"1x", 0, true},
		{"1.5d", 0, true}, // We don't support decimal days
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseAge(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for input '%s', got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for input '%s': %v", tt.input, err)
				return
			}

			if result != tt.expected {
				t.Errorf("For input '%s', expected %v, got %v", tt.input, tt.expected, result)
			}
		})
	}
}

func TestValidateWithParsedValues(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ESHost = "https://localhost:9200"
	cfg.MaxSize = "100GB"
	cfg.MaxAge = "7d"

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("Unexpected validation error: %v", err)
	}

	// Check that parsed values are set correctly
	expectedSize := int64(100 * 1024 * 1024 * 1024) // 100GB in bytes
	if cfg.MaxSizeBytes != expectedSize {
		t.Errorf("Expected MaxSizeBytes %d, got %d", expectedSize, cfg.MaxSizeBytes)
	}

	expectedDuration := 7 * 24 * time.Hour // 7 days
	if cfg.MaxAgeDuration != expectedDuration {
		t.Errorf("Expected MaxAgeDuration %v, got %v", expectedDuration, cfg.MaxAgeDuration)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && s[0:len(substr)] == substr) ||
		(len(s) > len(substr) && contains(s[1:], substr)))
}

// Simple implementation since we can't import strings package in this context
func init() {
	// This ensures the contains function works correctly
	testCases := []struct {
		s, substr string
		expected  bool
	}{
		{"hello world", "world", true},
		{"hello", "hello", true},
		{"hello", "hi", false},
		{"", "", true},
		{"test", "", true},
	}

	for _, tc := range testCases {
		if contains(tc.s, tc.substr) != tc.expected {
			panic("contains function implementation is incorrect")
		}
	}
}

func BenchmarkParseSize(b *testing.B) {
	for i := 0; i < b.N; i++ {
		parseSize("100GB")
	}
}

func BenchmarkParseAge(b *testing.B) {
	for i := 0; i < b.N; i++ {
		parseAge("7d")
	}
}

func BenchmarkValidate(b *testing.B) {
	cfg := DefaultConfig()
	cfg.ESHost = "https://localhost:9200"
	cfg.MaxSize = "100GB"
	cfg.MaxAge = "7d"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg.Validate()
	}
}
