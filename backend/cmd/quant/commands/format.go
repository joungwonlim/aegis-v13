package commands

import (
	"fmt"
	"time"
)

// ═══════════════════════════════════════════════════════════
// Common Formatting Utilities
// 모든 커맨드가 동일한 출력 포맷을 사용하도록 통일
// ═══════════════════════════════════════════════════════════

// JobMetadata holds job execution metadata
type JobMetadata struct {
	JobID     int
	JobType   string
	Tag       string
	Timestamp string
	Period    *Period // Optional
	Symbols   string  // Optional
}

// Period represents a date range
type Period struct {
	StartDate string
	EndDate   string
}

// PrintJobHeader prints a formatted job header
func PrintJobHeader(meta JobMetadata) {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("  %s\n", meta.JobType)
	fmt.Println("───────────────────────────────────────────────────────────")
	fmt.Printf("  Job ID    : #%d\n", meta.JobID)

	// Optional period
	if meta.Period != nil {
		fmt.Printf("  Period    : %s ~ %s\n", meta.Period.StartDate, meta.Period.EndDate)
	}

	// Optional symbols
	if meta.Symbols != "" {
		fmt.Printf("  Symbols   : %s\n", meta.Symbols)
	}

	fmt.Println("───────────────────────────────────────────────────────────")
	fmt.Printf("[%s] Manual collection triggered at %s\n", meta.Tag, meta.Timestamp)
}

// PrintProgress prints a progress step with counter
// Example: [Ranking] Fetched trading/KOSPI: 100 items [1/8]
func PrintProgress(tag string, message string, current int, total int) {
	fmt.Printf("[%s] %s [%d/%d]\n", tag, message, current, total)
}

// PrintJobCompletion prints job completion message
func PrintJobCompletion(jobID int, duration float64) {
	fmt.Println()
	fmt.Printf("✅ Job #%d completed in %.2fs (100%%)\n", jobID, duration)
}

// PrintSeparator prints a visual separator
func PrintSeparator() {
	fmt.Println("───────────────────────────────────────────────────────────")
}

// PrintDoubleSeparator prints a double-line separator
func PrintDoubleSeparator() {
	fmt.Println("═══════════════════════════════════════════════════════════")
}

// PrintWarning prints a warning message
func PrintWarning(message string) {
	fmt.Println()
	fmt.Printf("⚠️  %s\n", message)
	fmt.Println()
}

// PrintSuccess prints a success message
func PrintSuccess(message string) {
	fmt.Printf("✅ %s\n", message)
}

// PrintError prints an error message
func PrintError(message string) {
	fmt.Printf("❌ %s\n", message)
}

// PrintInfo prints an info message
func PrintInfo(message string) {
	fmt.Printf("ℹ️  %s\n", message)
}

// GetCurrentPeriod returns the current month period (last 30 days)
func GetCurrentPeriod() *Period {
	now := time.Now()
	return &Period{
		StartDate: now.AddDate(0, -1, 0).Format("2006-01-02"),
		EndDate:   now.Format("2006-01-02"),
	}
}

// SimulateProcessing simulates processing time (for demo)
func SimulateProcessing(milliseconds int) {
	time.Sleep(time.Duration(milliseconds) * time.Millisecond)
}

// PrintTableHeader prints a table header
func PrintTableHeader(columns []string, widths []int) {
	for i, col := range columns {
		fmt.Printf("%-*s", widths[i], col)
		if i < len(columns)-1 {
			fmt.Print("  ")
		}
	}
	fmt.Println()

	// Separator line
	totalWidth := 0
	for i, width := range widths {
		totalWidth += width
		if i < len(widths)-1 {
			totalWidth += 2 // spacing
		}
	}
	for i := 0; i < totalWidth; i++ {
		fmt.Print("─")
	}
	fmt.Println()
}

// PrintTableRow prints a table row
func PrintTableRow(values []string, widths []int) {
	for i, val := range values {
		fmt.Printf("%-*s", widths[i], val)
		if i < len(values)-1 {
			fmt.Print("  ")
		}
	}
	fmt.Println()
}

// PrintList prints a bulleted list
func PrintList(items []string) {
	for _, item := range items {
		fmt.Printf("   • %s\n", item)
	}
}

// PrintNumberedList prints a numbered list
func PrintNumberedList(items []string) {
	for i, item := range items {
		fmt.Printf("   %d. %s\n", i+1, item)
	}
}

// PrintKeyValue prints key-value pairs
func PrintKeyValue(key string, value string, keyWidth int) {
	fmt.Printf("   %-*s : %s\n", keyWidth, key, value)
}
