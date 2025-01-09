package model

import (
	"fmt"
	"time"
)

// WorkerID represents a unique identifier for a worker
type WorkerID string

// TaskStatus represents the current state of a task
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "PENDING"
	TaskStatusInProgress TaskStatus = "IN_PROGRESS"
	TaskStatusCompleted  TaskStatus = "COMPLETED"
	TaskStatusBlocked    TaskStatus = "BLOCKED"
)

// Task represents a single task assigned to a worker
type Task struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Description      string     `json:"description"`
	Status           TaskStatus `json:"status"`
	PlannedStartTime time.Time  `json:"plannedStartTime"`
	PlannedEndTime   time.Time  `json:"plannedEndTime"`
	Location         string     `json:"location"`
	Priority         int        `json:"priority"`
	BlockReason      string     `json:"blockReason,omitempty"`
	BlockedAt        time.Time  `json:"blockedAt,omitempty"`
	IsBlocked        bool       `json:"isBlocked"`
}

// DailySchedule represents a worker's schedule for a specific day
type DailySchedule struct {
	WorkerID WorkerID  `json:"workerId"`
	Date     time.Time `json:"date"`
	Tasks    []Task    `json:"tasks"`
	CheckIn  time.Time `json:"checkIn,omitempty"`
	CheckOut time.Time `json:"checkOut,omitempty"`
}

// WorkerStatus represents the current status of a worker
type WorkerStatus string

const (
	WorkerStatusOffDuty     WorkerStatus = "OFF_DUTY"
	WorkerStatusOnDuty      WorkerStatus = "ON_DUTY"
	WorkerStatusOnBreak     WorkerStatus = "ON_BREAK"
	WorkerStatusUnavailable WorkerStatus = "UNAVAILABLE"
	WorkerStatusCheckedOut  WorkerStatus = "CHECKED_OUT"
)

// BreakState represents a worker's break state
type BreakState struct {
	IsOnBreak    bool      `json:"isOnBreak"`
	StartTime    time.Time `json:"startTime,omitempty"`
	TotalBreaks  int       `json:"totalBreaks"`
	BreakMinutes int       `json:"breakMinutes"`
}

// BreakSignal represents a break start/end signal
type BreakSignal struct {
	WorkerID  WorkerID  `json:"workerId"`
	IsOnBreak bool      `json:"isOnBreak"`
	StartTime time.Time `json:"startTime"`
}

// WorkerState represents the current state of a worker during their workday
type WorkerState struct {
	WorkerID    WorkerID     `json:"workerId"`
	Status      WorkerStatus `json:"status"`
	CurrentTask *Task        `json:"currentTask,omitempty"`
	LastUpdated time.Time    `json:"lastUpdated"`
	BreakState  BreakState   `json:"breakState"`
}

// TaskUpdate represents an update to a task's status
type TaskUpdate struct {
	TaskID     string     `json:"taskId"`
	NewStatus  TaskStatus `json:"newStatus"`
	UpdateTime time.Time  `json:"updateTime"`
	Notes      string     `json:"notes,omitempty"`
	UpdatedBy  WorkerID   `json:"updatedBy"`
}

// CheckInRequest represents the data needed when a worker checks in
type CheckInRequest struct {
	WorkerID    WorkerID  `json:"workerId"`
	CheckInTime time.Time `json:"checkInTime"`
	JobSiteID   string    `json:"jobSiteId"`
	Date        time.Time `json:"date"`
}

// CheckOutRequest represents the data needed when a worker checks out
type CheckOutRequest struct {
	WorkerID     WorkerID  `json:"workerId"`
	CheckOutTime time.Time `json:"checkOutTime"`
	JobSiteID    string    `json:"jobSiteId"`
	Notes        string    `json:"notes"`
	Date         time.Time `json:"date"`
}

// Add this function to model/worker.go
func FormatWorkflowID(workerID WorkerID) string {
	return fmt.Sprintf("DWR-%s", workerID)
}
