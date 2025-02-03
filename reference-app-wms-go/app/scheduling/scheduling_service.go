package scheduling

import (
	"context"
	"fmt"
	"strings"
	"time"

	"reference-app-wms-go/app/db"

	"github.com/google/uuid"
)

// SchedulingService defines the interface for scheduling operations
// schedulingServiceImpl implements SchedulingService
type schedulingServiceImpl struct {
	db db.DB
}

func NewSchedulingService(db db.DB) *schedulingServiceImpl {
	return &schedulingServiceImpl{
		db: db,
	}
}

// CreateJobSite creates a new job site with validation
func (s *schedulingServiceImpl) CreateJobSite(ctx context.Context, name, location string) (*db.JobSite, error) {
	// Validate inputs
	if name == "" {
		return nil, fmt.Errorf("job site name cannot be empty")
	}
	if location == "" {
		return nil, fmt.Errorf("job site location cannot be empty")
	}

	now := time.Now().UTC()
	jobSite := &db.JobSite{
		ID:        uuid.New().String(),
		Name:      name,
		Location:  location,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.db.AddJobSite(ctx, jobSite); err != nil {
		return nil, fmt.Errorf("failed to create job site: %w", err)
	}

	return jobSite, nil
}

// GetJobSite retrieves a job site by ID
func (s *schedulingServiceImpl) GetJobSite(ctx context.Context, id string) (*db.JobSite, error) {
	return s.db.GetJobSite(ctx, id)
}

func (s *schedulingServiceImpl) CreateSchedule(ctx context.Context, jobSiteID string, scheduleID string, startDate, endDate time.Time) (*db.Schedule, error) {
	// Validate dates
	if startDate.After(endDate) {
		return nil, fmt.Errorf("start date must be before end date")
	}

	// Validate IDs
	if scheduleID == "" {
		return nil, fmt.Errorf("schedule ID is required")
	}
	if jobSiteID == "" {
		return nil, fmt.Errorf("job site ID is required")
	}

	// Check for overlaps
	if err := s.ValidateScheduleOverlap(ctx, jobSiteID, startDate, endDate); err != nil {
		return nil, err
	}

	now := time.Now()
	schedule := &db.Schedule{
		ID:        scheduleID,
		JobSiteID: jobSiteID,
		StartDate: startDate,
		EndDate:   endDate,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.db.AddSchedule(ctx, schedule); err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, fmt.Errorf("schedule with ID %s already exists", scheduleID)
		}
		return nil, fmt.Errorf("failed to create schedule: %w", err)
	}

	return schedule, nil
}

func (s *schedulingServiceImpl) GetSchedulesByJobSite(ctx context.Context, jobSiteID string) ([]db.Schedule, error) {
	return s.db.GetSchedulesByJobSite(ctx, jobSiteID)
}

func (s *schedulingServiceImpl) ValidateScheduleOverlap(ctx context.Context, jobSiteID string, startDate, endDate time.Time) error {
	schedules, err := s.GetSchedulesByJobSite(ctx, jobSiteID)
	if err != nil {
		return fmt.Errorf("failed to get existing schedules: %w", err)
	}

	// Normalize times to UTC for comparison
	startUTC := startDate.UTC()
	endUTC := endDate.UTC()

	for _, schedule := range schedules {
		scheduleStartUTC := schedule.StartDate.UTC()
		scheduleEndUTC := schedule.EndDate.UTC()

		// Check if the schedules overlap in time
		// Two time ranges overlap if one starts before the other ends
		// and the other starts before the first one ends
		if startUTC.Before(scheduleEndUTC) && scheduleStartUTC.Before(endUTC) {
			return fmt.Errorf("schedule overlaps with existing schedule %s (new: %s-%s, existing: %s-%s)",
				schedule.ID,
				startUTC.Format(time.RFC3339),
				endUTC.Format(time.RFC3339),
				scheduleStartUTC.Format(time.RFC3339),
				scheduleEndUTC.Format(time.RFC3339))
		}
	}

	return nil
}

func (s *schedulingServiceImpl) GetActiveSchedules(ctx context.Context, jobSiteID string) ([]db.Schedule, error) {
	now := time.Now()
	schedules, err := s.GetSchedulesByJobSite(ctx, jobSiteID)
	if err != nil {
		return nil, err
	}

	var activeSchedules []db.Schedule
	for _, schedule := range schedules {
		if schedule.StartDate.Before(now) && schedule.EndDate.After(now) {
			activeSchedules = append(activeSchedules, schedule)
		}
	}

	return activeSchedules, nil
}

func (s *schedulingServiceImpl) GetUpcomingSchedules(ctx context.Context, jobSiteID string) ([]db.Schedule, error) {
	now := time.Now()
	schedules, err := s.GetSchedulesByJobSite(ctx, jobSiteID)
	if err != nil {
		return nil, err
	}

	var upcomingSchedules []db.Schedule
	for _, schedule := range schedules {
		if schedule.StartDate.After(now) {
			upcomingSchedules = append(upcomingSchedules, schedule)
		}
	}

	return upcomingSchedules, nil
}

func (s *schedulingServiceImpl) GetWorkerTasks(ctx context.Context, workerID string, date time.Time) ([]db.Task, error) {
	return s.db.GetTasksByWorker(ctx, workerID, date)
}

func (s *schedulingServiceImpl) UpdateTaskStatus(ctx context.Context, taskID string, status string) error {
	return s.db.UpdateTaskStatus(ctx, taskID, status)
}

// CreateTask creates a new task
func (s *schedulingServiceImpl) CreateTask(ctx context.Context, task *db.Task) error {
	if task.ScheduleID == "" {
		return fmt.Errorf("schedule ID is required")
	}
	if task.WorkerID == "" {
		return fmt.Errorf("worker ID is required")
	}
	if task.Name == "" {
		return fmt.Errorf("task name is required")
	}
	if task.PlannedStartTime.After(task.PlannedEndTime) {
		return fmt.Errorf("start time must be before end time")
	}
	if err := s.db.AddTask(ctx, task); err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	return nil
}

// GetTaskByID retrieves task details by task ID, including its associated schedule and job site information
func (s *schedulingServiceImpl) GetTaskByID(ctx context.Context, taskID string) (*db.Task, error) {
	task, err := s.db.GetTaskByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("error fetching task: %w", err)
	}
	return task, nil
}

// GetScheduleByID retrieves a schedule by ID
func (s *schedulingServiceImpl) GetScheduleByID(ctx context.Context, scheduleID string) (*db.Schedule, error) {
	if scheduleID == "" {
		return nil, fmt.Errorf("schedule ID is required")
	}

	schedule, err := s.db.GetScheduleByID(ctx, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	return schedule, nil
}

// RecordWorkerAttendance records a worker's attendance for a given job site
func (s *schedulingServiceImpl) RecordWorkerAttendance(ctx context.Context, workerID string, jobSiteID string, checkInTime time.Time) error {
	// Validate inputs
	if workerID == "" {
		return fmt.Errorf("worker ID is required")
	}
	if jobSiteID == "" {
		return fmt.Errorf("job site ID is required")
	}

	// Verify job site exists
	jobSite, err := s.GetJobSite(ctx, jobSiteID)
	if err != nil {
		return fmt.Errorf("failed to verify job site exists: %w", err)
	}
	if jobSite == nil {
		return fmt.Errorf("job site not found: %s", jobSiteID)
	}

	// Create attendance record
	attendance := &db.WorkerAttendance{
		WorkerID:    workerID,
		JobSiteID:   jobSiteID,
		Date:        checkInTime,
		CheckInTime: checkInTime,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.db.AddWorkerAttendance(ctx, attendance); err != nil {
		return fmt.Errorf("failed to record worker attendance in database: %w", err)
	}

	return nil
}

// GetSiteAttendance retrieves attendance records for a job site on a specific date
func (s *schedulingServiceImpl) GetSiteAttendance(ctx context.Context, jobSiteID string, date time.Time) ([]db.WorkerAttendance, error) {
	if jobSiteID == "" {
		return nil, fmt.Errorf("job site ID is required")
	}

	attendance, err := s.db.GetWorkerAttendanceByDate(ctx, jobSiteID, date)
	if err != nil {
		return nil, fmt.Errorf("failed to get site attendance: %w", err)
	}

	return attendance, nil
}

// UpdateWorkerAttendance updates a worker's attendance record with check-out time
func (s *schedulingServiceImpl) UpdateWorkerAttendance(ctx context.Context, workerID string, jobSiteID string, date time.Time, checkOutTime time.Time) error {
	// Validate inputs
	if workerID == "" {
		return fmt.Errorf("worker ID is required")
	}
	if jobSiteID == "" {
		return fmt.Errorf("job site ID is required")
	}

	// Verify job site exists
	jobSite, err := s.GetJobSite(ctx, jobSiteID)
	if err != nil {
		return fmt.Errorf("failed to verify job site exists: %w", err)
	}
	if jobSite == nil {
		return fmt.Errorf("job site not found: %s", jobSiteID)
	}

	if err := s.db.UpdateWorkerAttendance(ctx, workerID, jobSiteID, date, checkOutTime); err != nil {
		return fmt.Errorf("failed to update attendance: %w", err)
	}

	return nil
}
