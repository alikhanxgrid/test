package workflows

import (
	"time"

	"reference-app-wms-go/app/dwr/model"
)

// CheckOutParams contains parameters for checking out a worker
type CheckOutParams struct {
	WorkerID     model.WorkerID `json:"workerId"`
	CheckOutTime time.Time      `json:"checkOutTime"`
	JobSiteID    string         `json:"jobSiteId"`
	Date         time.Time      `json:"date"`
	Notes        string         `json:"notes,omitempty"`
}

// TaskUpdateParams contains parameters for updating a task's status
type TaskUpdateParams struct {
	TaskID     string           `json:"taskId"`
	NewStatus  model.TaskStatus `json:"newStatus"`
	UpdateTime time.Time        `json:"updateTime"`
	Notes      string           `json:"notes,omitempty"`
	UpdatedBy  model.WorkerID   `json:"updatedBy"`
}

// BreakParams contains parameters for managing worker breaks
type BreakParams struct {
	IsOnBreak bool      `json:"isOnBreak"`
	StartTime time.Time `json:"startTime"`
}
