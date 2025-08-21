package utils

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
)

// FormatBytes converts bytes to human-readable format
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatNumber formats large numbers with K/M/B suffixes
func FormatNumber(num int64) string {
	if num < 1000 {
		return fmt.Sprintf("%d", num)
	}
	if num < 1000000 {
		return fmt.Sprintf("%.1fK", float64(num)/1000)
	}
	if num < 1000000000 {
		return fmt.Sprintf("%.1fM", float64(num)/1000000)
	}
	return fmt.Sprintf("%.1fB", float64(num)/1000000000)
}

// FormatDuration formats a duration in a human-readable way
func FormatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24

	if days > 0 {
		return fmt.Sprintf("%dd %dh ago", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm ago", hours, int(d.Minutes())%60)
	}
	return fmt.Sprintf("%dm ago", int(d.Minutes()))
}

// PrintBanner prints a formatted banner with the application name and version
func PrintBanner(version string) {
	header := color.New(color.FgBlue, color.Bold)
	header.Println("╔══════════════════════════════════════════╗")
	header.Println("║          Elasticsearch Log Trimmer      ║")
	header.Printf("║              Version %-8s         ║\n", version)
	header.Println("╚══════════════════════════════════════════╝")
	fmt.Println()
}

// PrintTableHeader prints a formatted table header
func PrintTableHeader(headers []string, widths []int) {
	for i, header := range headers {
		fmt.Printf("%-*s ", widths[i], header)
	}
	fmt.Println()

	totalWidth := 0
	for _, width := range widths {
		totalWidth += width + 1 // +1 for space
	}
	fmt.Println(strings.Repeat("-", totalWidth-1))
}

// PrintTableRow prints a formatted table row
func PrintTableRow(values []string, widths []int) {
	for i, value := range values {
		if i < len(widths) {
			fmt.Printf("%-*s ", widths[i], value)
		} else {
			fmt.Printf("%s ", value)
		}
	}
	fmt.Println()
}

// PrintTableFooter prints a table footer separator
func PrintTableFooter(widths []int) {
	totalWidth := 0
	for _, width := range widths {
		totalWidth += width + 1 // +1 for space
	}
	fmt.Println(strings.Repeat("-", totalWidth-1))
}
