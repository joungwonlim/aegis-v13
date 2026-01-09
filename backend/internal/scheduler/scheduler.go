package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// Scheduler manages scheduled jobs
// ⭐ SSOT: 스케줄 관리는 이 스케줄러에서만
type Scheduler struct {
	cron    *cron.Cron
	logger  *logger.Logger
	jobs    map[string]Job
	history map[string]*JobHistory
	mu      sync.RWMutex

	// Retry configuration
	maxRetries   int
	retryDelay   time.Duration
}

// New creates a new scheduler
func New(log *logger.Logger) *Scheduler {
	return &Scheduler{
		cron:         cron.New(cron.WithSeconds()),
		logger:       log,
		jobs:         make(map[string]Job),
		history:      make(map[string]*JobHistory),
		maxRetries:   3,
		retryDelay:   1 * time.Minute,
	}
}

// AddJob adds a job to the scheduler
func (s *Scheduler) AddJob(job Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobName := job.Name()

	// Check if job already exists
	if _, exists := s.jobs[jobName]; exists {
		return fmt.Errorf("job %s already exists", jobName)
	}

	// Add job to cron
	_, err := s.cron.AddFunc(job.Schedule(), func() {
		s.runJob(job)
	})
	if err != nil {
		return fmt.Errorf("failed to schedule job %s: %w", jobName, err)
	}

	// Store job
	s.jobs[jobName] = job
	s.history[jobName] = &JobHistory{}

	s.logger.WithFields(map[string]interface{}{
		"job":      jobName,
		"schedule": job.Schedule(),
	}).Info("Job added to scheduler")

	return nil
}

// RemoveJob removes a job from the scheduler
func (s *Scheduler) RemoveJob(jobName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[jobName]; !exists {
		return fmt.Errorf("job %s not found", jobName)
	}

	delete(s.jobs, jobName)
	s.logger.WithField("job", jobName).Info("Job removed from scheduler")

	return nil
}

// Start starts the scheduler
func (s *Scheduler) Start() {
	s.logger.Info("Starting scheduler")
	s.cron.Start()
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.logger.Info("Stopping scheduler")
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.logger.Info("Scheduler stopped")
}

// RunJob runs a specific job immediately (outside of schedule)
func (s *Scheduler) RunJob(jobName string) error {
	s.mu.RLock()
	job, exists := s.jobs[jobName]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("job %s not found", jobName)
	}

	go s.runJob(job)
	return nil
}

// runJob executes a job with retry logic
func (s *Scheduler) runJob(job Job) {
	jobName := job.Name()
	startTime := time.Now()

	s.logger.WithField("job", jobName).Info("Job started")

	var lastErr error
	var success bool

	// Try running the job with retries
	for attempt := 0; attempt <= s.maxRetries; attempt++ {
		ctx := context.Background()

		err := job.Run(ctx)
		if err == nil {
			success = true
			break
		}

		lastErr = err
		s.logger.WithFields(map[string]interface{}{
			"job":     jobName,
			"attempt": attempt + 1,
			"error":   err.Error(),
		}).Warn("Job execution failed, retrying")

		// Wait before retry (except on last attempt)
		if attempt < s.maxRetries {
			time.Sleep(s.retryDelay)
		}
	}

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	// Create job result
	result := JobResult{
		JobName:   jobName,
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  duration,
		Success:   success,
	}

	if !success && lastErr != nil {
		result.Error = lastErr.Error()
	}

	// Store result in history
	s.mu.Lock()
	if history, exists := s.history[jobName]; exists {
		history.AddResult(result)
	}
	s.mu.Unlock()

	// Log completion
	if success {
		s.logger.WithFields(map[string]interface{}{
			"job":      jobName,
			"duration": duration,
		}).Info("Job completed successfully")
	} else {
		s.logger.WithFields(map[string]interface{}{
			"job":      jobName,
			"duration": duration,
			"error":    lastErr.Error(),
		}).Error("Job failed after all retries")
	}
}

// GetJobHistory returns the history for a specific job
func (s *Scheduler) GetJobHistory(jobName string) (*JobHistory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history, exists := s.history[jobName]
	if !exists {
		return nil, fmt.Errorf("job %s not found", jobName)
	}

	return history, nil
}

// GetAllJobs returns all registered jobs
func (s *Scheduler) GetAllJobs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]string, 0, len(s.jobs))
	for jobName := range s.jobs {
		jobs = append(jobs, jobName)
	}

	return jobs
}

// GetJobStats returns statistics for all jobs
func (s *Scheduler) GetJobStats() map[string]JobStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]JobStats)

	for jobName, history := range s.history {
		latestResults := history.GetLatestResults(10)
		failedResults := history.GetFailedResults()

		var lastRun *time.Time
		var lastSuccess *time.Time
		var lastFailure *time.Time

		if len(latestResults) > 0 {
			lastResult := latestResults[len(latestResults)-1]
			lastRun = &lastResult.StartTime

			if lastResult.Success {
				lastSuccess = &lastResult.StartTime
			} else {
				lastFailure = &lastResult.StartTime
			}
		}

		stats[jobName] = JobStats{
			JobName:       jobName,
			Schedule:      s.jobs[jobName].Schedule(),
			TotalRuns:     len(history.Results),
			SuccessCount:  len(history.Results) - len(failedResults),
			FailureCount:  len(failedResults),
			SuccessRate:   history.GetSuccessRate(),
			LastRun:       lastRun,
			LastSuccess:   lastSuccess,
			LastFailure:   lastFailure,
		}
	}

	return stats
}

// JobStats represents statistics for a job
type JobStats struct {
	JobName      string     `json:"job_name"`
	Schedule     string     `json:"schedule"`
	TotalRuns    int        `json:"total_runs"`
	SuccessCount int        `json:"success_count"`
	FailureCount int        `json:"failure_count"`
	SuccessRate  float64    `json:"success_rate"`
	LastRun      *time.Time `json:"last_run,omitempty"`
	LastSuccess  *time.Time `json:"last_success,omitempty"`
	LastFailure  *time.Time `json:"last_failure,omitempty"`
}
