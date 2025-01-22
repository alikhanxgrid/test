package scheduling

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"reference-app-wms-go/app/db"
)

// SchedulingAPIClient implements the SchedulingService interface using HTTP calls
type SchedulingAPIClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *log.Logger
}

// NewSchedulingAPIClient creates a new client for the scheduling API
func NewSchedulingAPIClient(baseURL string) SchedulingService {
	logger := log.New(log.Default().Writer(), "[SchedulingAPI] ", log.LstdFlags|log.Lshortfile)
	logger.Printf("Initializing Scheduling API client with base URL: %s", baseURL)
	return &SchedulingAPIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

// GetWorkerTasks retrieves tasks for a worker on a specific date
func (c *SchedulingAPIClient) GetWorkerTasks(ctx context.Context, workerID string, date time.Time) ([]db.Task, error) {
	url := fmt.Sprintf("%s/scheduling/workers/%s/tasks?date=%s", c.baseURL, workerID, date.Format("2006-01-02"))
	c.logger.Printf("Fetching worker tasks: worker=%s, date=%s, url=%s", workerID, date.Format("2006-01-02"), url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		c.logger.Printf("Failed to create request: worker=%s, error=%v", workerID, err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Printf("Failed to execute request: worker=%s, error=%v", workerID, err)
		return nil, fmt.Errorf("failed to get worker tasks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Printf("Received non-OK status code: worker=%s, status=%d", workerID, resp.StatusCode)
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var tasks []db.Task
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		c.logger.Printf("Failed to decode response: worker=%s, error=%v", workerID, err)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Printf("Successfully retrieved %d tasks for worker=%s", len(tasks), workerID)
	return tasks, nil
}

// UpdateTaskStatus updates the status of a task
func (c *SchedulingAPIClient) UpdateTaskStatus(ctx context.Context, taskID string, status string) error {
	url := fmt.Sprintf("%s/scheduling/tasks/%s/status", c.baseURL, taskID)
	c.logger.Printf("Updating task status: task=%s, status=%s, url=%s", taskID, status, url)

	payload := map[string]string{"status": status}
	body, err := json.Marshal(payload)
	if err != nil {
		c.logger.Printf("Failed to marshal request: task=%s, error=%v", taskID, err)
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(body))
	if err != nil {
		c.logger.Printf("Failed to create request: task=%s, error=%v", taskID, err)
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Printf("Failed to execute request: task=%s, error=%v", taskID, err)
		return fmt.Errorf("failed to update task status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Printf("Received non-OK status code: task=%s, status=%d", taskID, resp.StatusCode)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	c.logger.Printf("Successfully updated task status: task=%s, status=%s", taskID, status)
	return nil
}

// CreateJobSite creates a new job site
func (c *SchedulingAPIClient) CreateJobSite(ctx context.Context, name, location string) (*db.JobSite, error) {
	url := fmt.Sprintf("%s/scheduling/job-sites", c.baseURL)
	c.logger.Printf("Creating job site: name=%s, location=%s, url=%s", name, location, url)

	payload := map[string]string{
		"name":     name,
		"location": location,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		c.logger.Printf("Failed to marshal request: name=%s, error=%v", name, err)
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		c.logger.Printf("Failed to create request: name=%s, error=%v", name, err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Printf("Failed to execute request: name=%s, error=%v", name, err)
		return nil, fmt.Errorf("failed to create job site: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		c.logger.Printf("Received non-Created status code: name=%s, status=%d", name, resp.StatusCode)
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var jobSite db.JobSite
	if err := json.NewDecoder(resp.Body).Decode(&jobSite); err != nil {
		c.logger.Printf("Failed to decode response: name=%s, error=%v", name, err)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Printf("Successfully created job site: id=%s, name=%s", jobSite.ID, name)
	return &jobSite, nil
}

// GetJobSite retrieves a job site by ID
func (c *SchedulingAPIClient) GetJobSite(ctx context.Context, id string) (*db.JobSite, error) {
	url := fmt.Sprintf("%s/scheduling/job-sites/%s", c.baseURL, id)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get job site: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var jobSite db.JobSite
	if err := json.NewDecoder(resp.Body).Decode(&jobSite); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &jobSite, nil
}

// CreateSchedule creates a new schedule for a job site
func (c *SchedulingAPIClient) CreateSchedule(ctx context.Context, jobSiteID string, scheduleID string, startDate, endDate time.Time) (*db.Schedule, error) {
	url := fmt.Sprintf("%s/scheduling/job-sites/%s/schedules", c.baseURL, jobSiteID)

	payload := map[string]string{
		"id":        scheduleID,
		"startDate": startDate.Format(time.RFC3339),
		"endDate":   endDate.Format(time.RFC3339),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create schedule: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var schedule db.Schedule
	if err := json.NewDecoder(resp.Body).Decode(&schedule); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &schedule, nil
}

// GetSchedulesByJobSite retrieves all schedules for a job site
func (c *SchedulingAPIClient) GetSchedulesByJobSite(ctx context.Context, jobSiteID string) ([]db.Schedule, error) {
	url := fmt.Sprintf("%s/scheduling/job-sites/%s/schedules", c.baseURL, jobSiteID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedules: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var schedules []db.Schedule
	if err := json.NewDecoder(resp.Body).Decode(&schedules); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return schedules, nil
}

// ValidateScheduleOverlap checks for schedule overlaps
func (c *SchedulingAPIClient) ValidateScheduleOverlap(ctx context.Context, jobSiteID string, startDate, endDate time.Time) error {
	url := fmt.Sprintf("%s/scheduling/job-sites/%s/schedules/validate", c.baseURL, jobSiteID)

	payload := map[string]string{
		"startDate": startDate.Format(time.RFC3339),
		"endDate":   endDate.Format(time.RFC3339),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to validate schedule: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		return fmt.Errorf("schedule overlap detected")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// GetActiveSchedules retrieves active schedules for a job site
func (c *SchedulingAPIClient) GetActiveSchedules(ctx context.Context, jobSiteID string) ([]db.Schedule, error) {
	url := fmt.Sprintf("%s/scheduling/job-sites/%s/schedules/active", c.baseURL, jobSiteID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get active schedules: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var schedules []db.Schedule
	if err := json.NewDecoder(resp.Body).Decode(&schedules); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return schedules, nil
}

// GetUpcomingSchedules retrieves upcoming schedules for a job site
func (c *SchedulingAPIClient) GetUpcomingSchedules(ctx context.Context, jobSiteID string) ([]db.Schedule, error) {
	url := fmt.Sprintf("%s/scheduling/job-sites/%s/schedules/upcoming", c.baseURL, jobSiteID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get upcoming schedules: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var schedules []db.Schedule
	if err := json.NewDecoder(resp.Body).Decode(&schedules); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return schedules, nil
}

// CreateTask creates a new task
func (c *SchedulingAPIClient) CreateTask(ctx context.Context, task *db.Task) error {
	url := fmt.Sprintf("%s/scheduling/job-sites/%s/tasks", c.baseURL, task.ScheduleID)
	c.logger.Printf("Creating task: name=%s, schedule=%s, url=%s", task.Name, task.ScheduleID, url)

	body, err := json.Marshal(task)
	if err != nil {
		c.logger.Printf("Failed to marshal task: %v", err)
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		c.logger.Printf("Failed to create request: %v", err)
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Printf("Failed to create task: %v", err)
		return fmt.Errorf("failed to create task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		c.logger.Printf("Received non-Created status code: %d", resp.StatusCode)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// GetTaskByID retrieves task details by task ID
func (c *SchedulingAPIClient) GetTaskByID(ctx context.Context, taskID string) (*db.Task, error) {
	url := fmt.Sprintf("%s/scheduling/tasks/%s", c.baseURL, taskID)
	c.logger.Printf("Fetching task: id=%s, url=%s", taskID, url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		c.logger.Printf("Failed to create request: task=%s, error=%v", taskID, err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Printf("Failed to execute request: task=%s, error=%v", taskID, err)
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Printf("Received non-OK status code: task=%s, status=%d", taskID, resp.StatusCode)
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var task db.Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		c.logger.Printf("Failed to decode response: task=%s, error=%v", taskID, err)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Printf("Successfully retrieved task: id=%s", taskID)
	return &task, nil
}

// GetScheduleByID retrieves schedule details by ID
func (c *SchedulingAPIClient) GetScheduleByID(ctx context.Context, scheduleID string) (*db.Schedule, error) {
	url := fmt.Sprintf("%s/scheduling/schedules/%s", c.baseURL, scheduleID)
	c.logger.Printf("Fetching schedule: id=%s, url=%s", scheduleID, url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var schedule db.Schedule
	if err := json.NewDecoder(resp.Body).Decode(&schedule); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &schedule, nil
}

// GetSiteAttendance retrieves attendance records for a job site on a specific date
func (c *SchedulingAPIClient) GetSiteAttendance(ctx context.Context, jobSiteID string, date time.Time) ([]db.WorkerAttendance, error) {
	url := fmt.Sprintf("%s/scheduling/job-sites/%s/attendance?date=%s", c.baseURL, jobSiteID, date.Format("2006-01-02"))

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get site attendance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var attendance []db.WorkerAttendance
	if err := json.NewDecoder(resp.Body).Decode(&attendance); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return attendance, nil
}

// RecordWorkerAttendance records a worker's attendance
func (c *SchedulingAPIClient) RecordWorkerAttendance(ctx context.Context, workerID string, jobSiteID string, checkInTime time.Time) error {
	url := fmt.Sprintf("%s/scheduling/workers/%s/attendance", c.baseURL, workerID)

	payload := map[string]interface{}{
		"jobSiteId":   jobSiteID,
		"checkInTime": checkInTime,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to record worker attendance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to record worker attendance: unexpected status code %d", resp.StatusCode)
	}

	return nil
}

// UpdateWorkerAttendance updates a worker's attendance record with check-out time
func (c *SchedulingAPIClient) UpdateWorkerAttendance(ctx context.Context, workerID string, jobSiteID string, date time.Time, checkOutTime time.Time) error {
	url := fmt.Sprintf("%s/scheduling/workers/%s/attendance", c.baseURL, workerID)

	payload := map[string]interface{}{
		"jobSiteId":    jobSiteID,
		"date":         date.Format("2006-01-02"),
		"checkOutTime": checkOutTime,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update attendance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// Other interface methods would be implemented similarly
