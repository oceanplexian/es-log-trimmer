package utils

import (
	"testing"
	"time"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{1024, "1.0 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		result := FormatBytes(tt.input)
		if result != tt.expected {
			t.Errorf("FormatBytes(%d) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1.0K"},
		{1000000, "1.0M"},
	}

	for _, tt := range tests {
		result := FormatNumber(tt.input)
		if result != tt.expected {
			t.Errorf("FormatNumber(%d) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{1 * time.Minute, "1m ago"},
		{1 * time.Hour, "1h 0m ago"},
		{24 * time.Hour, "1d 0h ago"},
	}

	for _, tt := range tests {
		result := FormatDuration(tt.input)
		if result != tt.expected {
			t.Errorf("FormatDuration(%v) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestPrintFunctions(t *testing.T) {
	// Test that print functions don't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Print function panicked: %v", r)
		}
	}()

	PrintBanner("1.0.0")
	PrintTableHeader([]string{"Name", "Size"}, []int{20, 10})
	PrintTableRow([]string{"test", "100MB"}, []int{20, 10})
	PrintTableFooter([]int{20, 10})
}
