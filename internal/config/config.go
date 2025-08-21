package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/company/log-trimmer/internal/logger"
)

const (
	Version = "1.0.0"
)

// Config holds all application configuration
type Config struct {
	// Elasticsearch settings
	ESHost   string `json:"es_host" yaml:"es_host"`
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
	SkipTLS  bool   `json:"skip_tls" yaml:"skip_tls"`

	// Trimming settings
	MaxSize        string        `json:"max_size" yaml:"max_size"`
	MaxAge         string        `json:"max_age" yaml:"max_age"`
	IndexPattern   string        `json:"index_pattern" yaml:"index_pattern"`
	DeleteIndexes  bool          `json:"delete_indexes" yaml:"delete_indexes"`
	MaxSizeBytes   int64         `json:"-" yaml:"-"`
	MaxAgeDuration time.Duration `json:"-" yaml:"-"`

	// Application settings
	Verbose bool           `json:"verbose" yaml:"verbose"`
	Logger  *logger.Config `json:"logger" yaml:"logger"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		ESHost:        "",
		Username:      "",
		Password:      "",
		SkipTLS:       true,
		MaxSize:       "",
		MaxAge:        "",
		IndexPattern:  "vector-*",
		DeleteIndexes: false,
		Verbose:       false,
		Logger:        logger.DefaultConfig(),
	}
}

// LoadFromEnv loads configuration from environment variables
func (c *Config) LoadFromEnv() {
	// Elasticsearch settings
	if host := os.Getenv("ES_HOST"); host != "" {
		c.ESHost = host
	}
	if username := os.Getenv("ES_USERNAME"); username != "" {
		c.Username = username
	}
	if password := os.Getenv("ES_PASSWORD"); password != "" {
		c.Password = password
	}
	if skipTLS := os.Getenv("ES_SKIP_TLS"); skipTLS != "" {
		c.SkipTLS = strings.ToLower(skipTLS) == "true"
	}

	// Trimming settings
	if maxSize := os.Getenv("MAX_SIZE"); maxSize != "" {
		c.MaxSize = maxSize
	}
	if maxAge := os.Getenv("MAX_AGE"); maxAge != "" {
		c.MaxAge = maxAge
	}
	if pattern := os.Getenv("INDEX_PATTERN"); pattern != "" {
		c.IndexPattern = pattern
	}
	if deleteIndexes := os.Getenv("DELETE_INDEXES"); deleteIndexes != "" {
		c.DeleteIndexes = strings.ToLower(deleteIndexes) == "true"
	}

	// Application settings
	if verbose := os.Getenv("VERBOSE"); verbose != "" {
		c.Verbose = strings.ToLower(verbose) == "true"
	}

	// Logger settings
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		c.Logger.Level = logger.LogLevel(strings.ToLower(logLevel))
	}
	if logFormat := os.Getenv("LOG_FORMAT"); logFormat != "" {
		c.Logger.Format = strings.ToLower(logFormat)
	}
	if logOutput := os.Getenv("LOG_OUTPUT"); logOutput != "" {
		c.Logger.Output = strings.ToLower(logOutput)
	}
	if logFile := os.Getenv("LOG_FILE"); logFile != "" {
		c.Logger.EnableFile = true
		c.Logger.FilePath = logFile
	}
}

// Validate validates the configuration and parses computed fields
func (c *Config) Validate() error {
	// Host is required
	if c.ESHost == "" {
		return fmt.Errorf("elasticsearch host is required (use --host flag or ES_HOST environment variable)")
	}

	// Parse max size if provided
	if c.MaxSize != "" {
		size, err := parseSize(c.MaxSize)
		if err != nil {
			return fmt.Errorf("invalid max-size format '%s': %v", c.MaxSize, err)
		}
		c.MaxSizeBytes = size
	}

	// Parse max age if provided
	if c.MaxAge != "" {
		duration, err := parseAge(c.MaxAge)
		if err != nil {
			return fmt.Errorf("invalid max-age format '%s': %v", c.MaxAge, err)
		}
		c.MaxAgeDuration = duration
	}

	// Must specify at least one constraint
	if c.MaxSize == "" && c.MaxAge == "" {
		return fmt.Errorf("must specify at least one of --max-size/MAX_SIZE or --max-age/MAX_AGE")
	}

	return nil
}

// parseSize parses a size string like "10GB" into bytes
func parseSize(sizeStr string) (int64, error) {
	re := regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*([KMGT]?B?)$`)
	matches := re.FindStringSubmatch(strings.ToUpper(sizeStr))
	if len(matches) != 3 {
		return 0, fmt.Errorf("invalid size format, expected format like '10GB', '500MB'")
	}

	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, err
	}

	unit := matches[2]
	multiplier := int64(1)

	switch unit {
	case "B", "":
		multiplier = 1
	case "KB":
		multiplier = 1024
	case "MB":
		multiplier = 1024 * 1024
	case "GB":
		multiplier = 1024 * 1024 * 1024
	case "TB":
		multiplier = 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unknown size unit: %s", unit)
	}

	return int64(value * float64(multiplier)), nil
}

// parseAge parses an age string like "7d" into a duration
func parseAge(ageStr string) (time.Duration, error) {
	re := regexp.MustCompile(`^(\d+)\s*([smhdw])$`)
	matches := re.FindStringSubmatch(strings.ToLower(ageStr))
	if len(matches) != 3 {
		return 0, fmt.Errorf("invalid age format, expected format like '7d', '24h', '30m'")
	}

	value, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, err
	}

	unit := matches[2]
	var duration time.Duration

	switch unit {
	case "s":
		duration = time.Duration(value) * time.Second
	case "m":
		duration = time.Duration(value) * time.Minute
	case "h":
		duration = time.Duration(value) * time.Hour
	case "d":
		duration = time.Duration(value) * 24 * time.Hour
	case "w":
		duration = time.Duration(value) * 7 * 24 * time.Hour
	default:
		return 0, fmt.Errorf("unknown age unit: %s", unit)
	}

	return duration, nil
}
