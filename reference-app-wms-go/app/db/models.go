package db

import "time"

// JobSite represents a record in the job_sites table
type JobSite struct {
	ID        string    `db:"id"`
	Name      string    `db:"name"`
	Location  string    `db:"location"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Schedule represents a record in the schedules table
type Schedule struct {
	ID        string    `db:"id"`
	JobSiteID string    `db:"job_site_id"`
	StartDate time.Time `db:"start_date"`
	EndDate   time.Time `db:"end_date"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// TaskStatus represents the status of a task in the database
type TaskStatus string

const (
	// Task statuses
	TaskStatusPending    TaskStatus = "PENDING"
	TaskStatusInProgress TaskStatus = "IN_PROGRESS"
	TaskStatusCompleted  TaskStatus = "COMPLETED"
	TaskStatusBlocked    TaskStatus = "BLOCKED"
)

// Task represents a record in the tasks table
type Task struct {
	ID               string    `db:"id" json:"id"`
	ScheduleID       string    `db:"schedule_id" json:"schedule_id"`
	WorkerID         string    `db:"worker_id" json:"worker_id"`
	Name             string    `db:"name" json:"name"`
	Description      string    `db:"description" json:"description"`
	PlannedStartTime time.Time `db:"planned_start_time" json:"planned_start_time"`
	PlannedEndTime   time.Time `db:"planned_end_time" json:"planned_end_time"`
	Status           string    `db:"status" json:"status"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time `db:"updated_at" json:"updated_at"`
}

// WorkerAttendance represents a record in the worker_attendance table
type WorkerAttendance struct {
	WorkerID     string    `db:"worker_id"`
	JobSiteID    string    `db:"job_site_id"`
	Date         time.Time `db:"date"`
	CheckInTime  time.Time `db:"check_in_time"`
	CheckOutTime time.Time `db:"check_out_time"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

// Worker represents a record in the workers table
type Worker struct {
	ID        string    `db:"id"`
	FullName  string    `db:"full_name"`
	Role      string    `db:"role"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
