package logger

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
)

// LogLevel represents the log level
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
	LevelFatal LogLevel = "fatal"
)

// Logger provides structured logging with both JSON and colorized console output
type Logger struct {
	structured *logrus.Logger
	console    *ConsoleLogger
	level      logrus.Level
}

// ConsoleLogger provides colorized console output
type ConsoleLogger struct {
	Info    *color.Color
	Success *color.Color
	Warning *color.Color
	Error   *color.Color
	Debug   *color.Color
	Header  *color.Color
}

// Config holds logger configuration
type Config struct {
	Level      LogLevel `json:"level" yaml:"level"`
	Format     string   `json:"format" yaml:"format"` // "json" or "console"
	Output     string   `json:"output" yaml:"output"` // "stdout", "stderr", or file path
	EnableFile bool     `json:"enable_file" yaml:"enable_file"`
	FilePath   string   `json:"file_path" yaml:"file_path"`
}

// DefaultConfig returns a default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:      LevelInfo,
		Format:     "console",
		Output:     "stdout",
		EnableFile: false,
		FilePath:   "/var/log/log-trimmer/app.log",
	}
}

// New creates a new logger instance
func New(config *Config) (*Logger, error) {
	// Create structured logger (always JSON for consistency)
	structured := logrus.New()
	structured.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
			logrus.FieldKeyFunc:  "function",
			logrus.FieldKeyFile:  "source",
		},
	})

	// Set log level
	logLevel, err := logrus.ParseLevel(string(config.Level))
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	structured.SetLevel(logLevel)

	// Set output destination
	if config.EnableFile && config.FilePath != "" {
		file, err := os.OpenFile(config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}

		if config.Format == "json" {
			structured.SetOutput(file)
		} else {
			// For console format, write JSON to file and console to stdout
			structured.SetOutput(file)
		}
	} else {
		if config.Format == "json" {
			if config.Output == "stderr" {
				structured.SetOutput(os.Stderr)
			} else {
				structured.SetOutput(os.Stdout)
			}
		} else {
			// For console format, we'll handle output in the console logger
			structured.SetOutput(io.Discard)
		}
	}

	// Create console logger for colorized output
	console := &ConsoleLogger{
		Info:    color.New(color.FgCyan),
		Success: color.New(color.FgGreen),
		Warning: color.New(color.FgYellow),
		Error:   color.New(color.FgRed, color.Bold),
		Debug:   color.New(color.FgMagenta),
		Header:  color.New(color.FgBlue, color.Bold),
	}

	logger := &Logger{
		structured: structured,
		console:    console,
		level:      logLevel,
	}

	return logger, nil
}

// getCallerInfo returns caller information for structured logging
func getCallerInfo() (string, string) {
	pc, file, _, ok := runtime.Caller(3)
	if !ok {
		return "", ""
	}

	fn := runtime.FuncForPC(pc)
	var funcName string
	if fn != nil {
		funcName = fn.Name()
		// Extract just the function name, not the full path
		if idx := strings.LastIndex(funcName, "."); idx != -1 {
			funcName = funcName[idx+1:]
		}
	}

	// Extract just the file name, not the full path
	if idx := strings.LastIndex(file, "/"); idx != -1 {
		file = file[idx+1:]
	}

	return funcName, file
}

// structuredLog logs to the structured logger with caller information
func (l *Logger) structuredLog(level logrus.Level, component, operation, message string, fields map[string]interface{}) {
	if l.structured.Level < level {
		return
	}

	funcName, fileName := getCallerInfo()

	entry := l.structured.WithFields(logrus.Fields{
		"component": component,
		"operation": operation,
		"function":  funcName,
		"source":    fileName,
		"service":   "log-trimmer",
		"version":   "1.0.0",
	})

	// Add any additional fields
	for k, v := range fields {
		entry = entry.WithField(k, v)
	}

	switch level {
	case logrus.DebugLevel:
		entry.Debug(message)
	case logrus.InfoLevel:
		entry.Info(message)
	case logrus.WarnLevel:
		entry.Warn(message)
	case logrus.ErrorLevel:
		entry.Error(message)
	case logrus.FatalLevel:
		entry.Fatal(message)
	}
}

// Info logs an info message
func (l *Logger) Info(component, operation, message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.structuredLog(logrus.InfoLevel, component, operation, message, f)

	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// Add context from fields for better console readability
	contextInfo := ""
	if f != nil && len(f) > 0 {
		var contexts []string
		for k, v := range f {
			if k == "index" || k == "count" || k == "size" || k == "pattern" || k == "status_code" || k == "method" {
				contexts = append(contexts, fmt.Sprintf("%s=%v", k, v))
			}
		}
		if len(contexts) > 0 {
			contextInfo = fmt.Sprintf(" (%s)", strings.Join(contexts, ", "))
		}
	}

	l.console.Info.Printf("%s [INFO] [%s:%s] %s%s\n", timestamp, component, operation, message, contextInfo)
}

// Success logs a success message (info level with green color)
func (l *Logger) Success(component, operation, message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.structuredLog(logrus.InfoLevel, component, operation, message, f)

	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// Add context from fields for better console readability
	contextInfo := ""
	if f != nil && len(f) > 0 {
		var contexts []string
		for k, v := range f {
			if k == "index" || k == "count" || k == "size" || k == "pattern" || k == "status_code" || k == "method" {
				contexts = append(contexts, fmt.Sprintf("%s=%v", k, v))
			}
		}
		if len(contexts) > 0 {
			contextInfo = fmt.Sprintf(" (%s)", strings.Join(contexts, ", "))
		}
	}

	l.console.Success.Printf("%s [SUCCESS] [%s:%s] %s%s\n", timestamp, component, operation, message, contextInfo)
}

// Warn logs a warning message
func (l *Logger) Warn(component, operation, message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.structuredLog(logrus.WarnLevel, component, operation, message, f)

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	l.console.Warning.Printf("%s [WARN] [%s:%s] %s\n", timestamp, component, operation, message)
}

// Error logs an error message
func (l *Logger) Error(component, operation, message string, err error, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	} else {
		f = make(map[string]interface{})
	}

	if err != nil {
		f["error"] = err.Error()
		message = message + ": " + err.Error()
	}

	l.structuredLog(logrus.ErrorLevel, component, operation, message, f)

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	l.console.Error.Printf("%s [ERROR] [%s:%s] %s\n", timestamp, component, operation, message)
}

// Debug logs a debug message
func (l *Logger) Debug(component, operation, message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.structuredLog(logrus.DebugLevel, component, operation, message, f)

	if l.level <= logrus.DebugLevel {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		l.console.Debug.Printf("%s [DEBUG] [%s:%s] %s\n", timestamp, component, operation, message)
	}
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(component, operation, message string, err error, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	} else {
		f = make(map[string]interface{})
	}

	if err != nil {
		f["error"] = err.Error()
		message = message + ": " + err.Error()
	}

	l.structuredLog(logrus.FatalLevel, component, operation, message, f)

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	l.console.Error.Printf("%s [FATAL] [%s:%s] %s\n", timestamp, component, operation, message)
	os.Exit(1)
}

// Header prints a header message (console only)
func (l *Logger) Header(message string) {
	l.console.Header.Println(message)
}

// Printf provides formatted console output for backward compatibility
func (l *Logger) Printf(format string, args ...interface{}) {
	l.console.Info.Printf(format, args...)
}

// Println provides console output for backward compatibility
func (l *Logger) Println(message string) {
	l.console.Info.Println(message)
}

// SetLevel changes the log level
func (l *Logger) SetLevel(level LogLevel) {
	logrusLevel, err := logrus.ParseLevel(string(level))
	if err != nil {
		return
	}
	l.structured.SetLevel(logrusLevel)
	l.level = logrusLevel
}
