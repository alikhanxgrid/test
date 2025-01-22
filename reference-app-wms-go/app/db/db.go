package db

import (
	"context"
	"time"
)

// DB defines the interface for database operations
type DB interface {
	Close() error
	AddJobSite(ctx context.Context, site *JobSite) error
	GetJobSite(ctx context.Context, id string) (*JobSite, error)
	AddSchedule(ctx context.Context, schedule *Schedule) error
	GetSchedulesByJobSite(ctx context.Context, jobSiteID string) ([]Schedule, error)
	GetScheduleByID(ctx context.Context, scheduleID string) (*Schedule, error)
	GetTasksByWorker(ctx context.Context, workerID string, date time.Time) ([]Task, error)
	UpdateTaskStatus(ctx context.Context, taskID string, status string) error
	AddTask(ctx context.Context, task *Task) error
	GetTaskByID(ctx context.Context, taskID string) (*Task, error)
	AddWorkerAttendance(ctx context.Context, attendance *WorkerAttendance) error
	UpdateWorkerAttendance(ctx context.Context, workerID string, jobSiteID string, date time.Time, checkOutTime time.Time) error
	GetWorkerAttendanceByDate(ctx context.Context, jobSiteID string, date time.Time) ([]WorkerAttendance, error)
	GetWorkerAttendanceByDateRange(ctx context.Context, jobSiteID string, startDate, endDate time.Time) ([]WorkerAttendance, error)

	// Analytics methods
	Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	Get(ctx context.Context, dest interface{}, query string, args ...interface{}) error
}
