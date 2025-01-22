package scheduling

import (
	"context"
	"reference-app-wms-go/app/db"
	"time"
)

// SchedulingService defines the interface for scheduling operations
type SchedulingService interface {
	CreateJobSite(ctx context.Context, name, location string) (*db.JobSite, error)
	GetJobSite(ctx context.Context, id string) (*db.JobSite, error)
	CreateSchedule(ctx context.Context, jobSiteID string, scheduleID string, startDate, endDate time.Time) (*db.Schedule, error)
	GetSchedulesByJobSite(ctx context.Context, jobSiteID string) ([]db.Schedule, error)
	GetScheduleByID(ctx context.Context, scheduleID string) (*db.Schedule, error)
	ValidateScheduleOverlap(ctx context.Context, jobSiteID string, startDate, endDate time.Time) error
	GetActiveSchedules(ctx context.Context, jobSiteID string) ([]db.Schedule, error)
	GetUpcomingSchedules(ctx context.Context, jobSiteID string) ([]db.Schedule, error)
	GetWorkerTasks(ctx context.Context, workerID string, date time.Time) ([]db.Task, error)
	UpdateTaskStatus(ctx context.Context, taskID string, status string) error
	CreateTask(ctx context.Context, task *db.Task) error
	GetTaskByID(ctx context.Context, taskID string) (*db.Task, error)
	RecordWorkerAttendance(ctx context.Context, workerID string, jobSiteID string, checkInTime time.Time) error
	UpdateWorkerAttendance(ctx context.Context, workerID string, jobSiteID string, date time.Time, checkOutTime time.Time) error
	GetSiteAttendance(ctx context.Context, jobSiteID string, date time.Time) ([]db.WorkerAttendance, error)
}
