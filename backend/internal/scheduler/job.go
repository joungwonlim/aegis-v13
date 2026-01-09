package scheduler

import (
	"context"
	"time"
)

// Job represents a scheduled job
// ⭐ SSOT: 스케줄 작업 인터페이스는 여기서만 정의
type Job interface {
	// Name returns the job name
	Name() string

	// Run executes the job
	Run(ctx context.Context) error

	// Schedule returns the cron schedule expression
	// Examples: "0 16 * * *" (every day at 4 PM)
	//           "@daily", "@hourly"
	Schedule() string
}

// JobResult represents the result of a job execution
type JobResult struct {
	JobName   string        `json:"job_name"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
	Success   bool          `json:"success"`
	Error     string        `json:"error,omitempty"`
}

// JobHistory stores job execution history
type JobHistory struct {
	Results []JobResult
}

// AddResult adds a job result to history
func (h *JobHistory) AddResult(result JobResult) {
	h.Results = append(h.Results, result)

	// Keep only last 100 results
	if len(h.Results) > 100 {
		h.Results = h.Results[len(h.Results)-100:]
	}
}

// GetLatestResults returns the latest N results
func (h *JobHistory) GetLatestResults(n int) []JobResult {
	if n > len(h.Results) {
		n = len(h.Results)
	}

	if n == 0 {
		return []JobResult{}
	}

	return h.Results[len(h.Results)-n:]
}

// GetFailedResults returns all failed results
func (h *JobHistory) GetFailedResults() []JobResult {
	failed := make([]JobResult, 0)
	for _, result := range h.Results {
		if !result.Success {
			failed = append(failed, result)
		}
	}
	return failed
}

// GetSuccessRate returns the success rate (0.0 - 1.0)
func (h *JobHistory) GetSuccessRate() float64 {
	if len(h.Results) == 0 {
		return 0.0
	}

	successCount := 0
	for _, result := range h.Results {
		if result.Success {
			successCount++
		}
	}

	return float64(successCount) / float64(len(h.Results))
}
