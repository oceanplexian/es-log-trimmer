package utils

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"
	"time"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},                // 1.5 * 1024
		{1048576, "1.0 MB"},             // 1024 * 1024
		{1073741824, "1.0 GB"},          // 1024 * 1024 * 1024
		{1099511627776, "1.0 TB"},       // 1024 * 1024 * 1024 * 1024
		{1125899906842624, "1.0 PB"},    // 1024^5
		{1152921504606846976, "1.0 EB"}, // 1024^6
		{5368709120, "5.0 GB"},          // 5 * 1024^3
		{549755813888, "512.0 GB"},      // 512 * 1024^3
		{1000, "1000 B"},
		{1500000, "1.4 MB"}, // ~1.43 MB
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("bytes_%d", tt.input), func(t *testing.T) {
			result := FormatBytes(tt.input)
			if result != tt.expected {
				t.Errorf("FormatBytes(%d) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{999, "999"},
		{1000, "1.0K"},
		{1500, "1.5K"},
		{999999, "1000.0K"},
		{1000000, "1.0M"},
		{1500000, "1.5M"},
		{999999999, "1000.0M"},
		{1000000000, "1.0B"},
		{1500000000, "1.5B"},
		{42, "42"},
		{123456, "123.5K"},
		{987654321, "987.7M"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("number_%d", tt.input), func(t *testing.T) {
			result := FormatNumber(tt.input)
			if result != tt.expected {
				t.Errorf("FormatNumber(%d) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{1 * time.Minute, "1m ago"},
		{30 * time.Minute, "30m ago"},
		{59 * time.Minute, "59m ago"},
		{1 * time.Hour, "1h 0m ago"},
		{1*time.Hour + 30*time.Minute, "1h 30m ago"},
		{2 * time.Hour, "2h 0m ago"},
		{23 * time.Hour, "23h 0m ago"},
		{24 * time.Hour, "1d 0h ago"},
		{25 * time.Hour, "1d 1h ago"},
		{48 * time.Hour, "2d 0h ago"},
		{72*time.Hour + 3*time.Hour, "3d 3h ago"},
		{7 * 24 * time.Hour, "7d 0h ago"},
		{30 * time.Second, "0m ago"}, // Less than a minute
		{90 * time.Second, "1m ago"}, // 1.5 minutes rounds to 1
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("duration_%v", tt.input), func(t *testing.T) {
			result := FormatDuration(tt.input)
			if result != tt.expected {
				t.Errorf("FormatDuration(%v) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestPrintBanner(t *testing.T) {
	// Since the banner function uses the color package which writes directly to stdout,
	// and may not be easily captured in tests, we'll just call it to ensure it doesn't panic
	version := "1.0.0"

	// This test ensures the function doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PrintBanner panicked: %v", r)
		}
	}()

	PrintBanner(version)

	// If we reach here, the function completed without panicking
	t.Log("PrintBanner executed successfully")
}

func TestPrintTableHeader(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	headers := []string{"Name", "Size", "Age"}
	widths := []int{20, 10, 15}
	PrintTableHeader(headers, widths)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read the output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Check that the output contains the headers
	expectedElements := []string{"Name", "Size", "Age", "---"}

	for _, element := range expectedElements {
		if !contains(output, element) {
			t.Errorf("Expected table header to contain '%s', but it was missing.\nActual output:\n%s", element, output)
		}
	}
}

func TestPrintTableRow(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	values := []string{"test-index", "100MB", "2d ago"}
	widths := []int{20, 10, 15}
	PrintTableRow(values, widths)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read the output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Check that the output contains all values
	for _, value := range values {
		if !contains(output, value) {
			t.Errorf("Expected table row to contain '%s', but it was missing.\nActual output:\n%s", value, output)
		}
	}
}

func TestPrintTableFooter(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	widths := []int{20, 10, 15}
	PrintTableFooter(widths)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read the output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Check that the output contains dashes (footer separator)
	if !contains(output, "---") {
		t.Errorf("Expected table footer to contain dashes, but it was missing.\nActual output:\n%s", output)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmarks
func BenchmarkFormatBytes(b *testing.B) {
	testSizes := []int64{1024, 1048576, 1073741824, 1099511627776}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, size := range testSizes {
			FormatBytes(size)
		}
	}
}

func BenchmarkFormatNumber(b *testing.B) {
	testNumbers := []int64{1000, 1000000, 1000000000}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, num := range testNumbers {
			FormatNumber(num)
		}
	}
}

func BenchmarkFormatDuration(b *testing.B) {
	testDurations := []time.Duration{
		1 * time.Hour,
		24 * time.Hour,
		7 * 24 * time.Hour,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, duration := range testDurations {
			FormatDuration(duration)
		}
	}
}

// Table-driven test for edge cases
func TestFormatBytesEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected string
	}{
		{"negative value", -1, "-1 B"},
		{"max int64", 9223372036854775807, "8.0 EB"},
		{"zero", 0, "0 B"},
		{"exactly 1KB", 1024, "1.0 KB"},
		{"just under 1KB", 1023, "1023 B"},
		{"just over 1KB", 1025, "1.0 KB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBytes(tt.input)
			if result != tt.expected {
				t.Errorf("FormatBytes(%d) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatNumberEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected string
	}{
		{"negative value", -1000, "-1000"},
		{"exactly 1K", 1000, "1.0K"},
		{"just under 1K", 999, "999"},
		{"just over 1K", 1001, "1.0K"},
		{"exactly 1M", 1000000, "1.0M"},
		{"exactly 1B", 1000000000, "1.0B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatNumber(tt.input)
			if result != tt.expected {
				t.Errorf("FormatNumber(%d) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}
