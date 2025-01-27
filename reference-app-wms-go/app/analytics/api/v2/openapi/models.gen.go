// Package apiv2 provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/oapi-codegen/oapi-codegen/v2 version v2.4.1 DO NOT EDIT.
package apiv2

import (
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Error defines model for Error.
type Error struct {
	Error string `json:"error"`
}

// HistoricalTaskInfo defines model for HistoricalTaskInfo.
type HistoricalTaskInfo struct {
	Id               *string             `json:"id,omitempty"`
	Name             *string             `json:"name,omitempty"`
	PlannedEndTime   *openapi_types.Date `json:"plannedEndTime,omitempty"`
	PlannedStartTime *openapi_types.Date `json:"plannedStartTime,omitempty"`
	Status           *string             `json:"status,omitempty"`
}

// ProductivityStats defines model for ProductivityStats.
type ProductivityStats struct {
	// AvgCompletionTime Duration (e.g., 1h23m)
	AvgCompletionTime *string `json:"avgCompletionTime,omitempty"`
	BlockedTasks      *int    `json:"blockedTasks,omitempty"`
	CompletedTasks    *int    `json:"completedTasks,omitempty"`

	// CompletionRate 0.0 to 1.0
	CompletionRate *float32 `json:"completionRate,omitempty"`
	TotalTasks     *int     `json:"totalTasks,omitempty"`
}

// SiteProductivityMetrics defines model for SiteProductivityMetrics.
type SiteProductivityMetrics struct {
	DailyStats *map[string]SiteProductivityStats `json:"dailyStats,omitempty"`
	JobSite    *string                           `json:"jobSite,omitempty"`
}

// SiteProductivityStats defines model for SiteProductivityStats.
type SiteProductivityStats struct {
	ActiveWorkers     *int     `json:"activeWorkers,omitempty"`
	AvgCompletionTime *string  `json:"avgCompletionTime,omitempty"`
	BlockedTasks      *int     `json:"blockedTasks,omitempty"`
	CompletedTasks    *int     `json:"completedTasks,omitempty"`
	CompletionRate    *float32 `json:"completionRate,omitempty"`
	TotalTasks        *int     `json:"totalTasks,omitempty"`
}

// SiteTaskDistribution defines model for SiteTaskDistribution.
type SiteTaskDistribution struct {
	JobSite *string `json:"jobSite,omitempty"`

	// WorkerStats Key = workerID, Value = stats
	WorkerStats *map[string]interface{} `json:"workerStats,omitempty"`
}

// SiteUtilizationMetrics defines model for SiteUtilizationMetrics.
type SiteUtilizationMetrics struct {
	JobSite *string               `json:"jobSite,omitempty"`
	Stats   *SiteUtilizationStats `json:"stats,omitempty"`
}

// SiteUtilizationStats defines model for SiteUtilizationStats.
type SiteUtilizationStats struct {
	AvgWorkDuration     *string  `json:"avgWorkDuration,omitempty"`
	LateWorkers         *int     `json:"lateWorkers,omitempty"`
	OnTimeWorkers       *int     `json:"onTimeWorkers,omitempty"`
	SiteUtilizationRate *float32 `json:"siteUtilizationRate,omitempty"`
	TotalWorkers        *int     `json:"totalWorkers,omitempty"`
}

// TaskBlockageInfo defines model for TaskBlockageInfo.
type TaskBlockageInfo struct {
	BlockReason *string `json:"blockReason,omitempty"`
	IsBlocked   bool    `json:"isBlocked"`

	// Since Unix timestamp since the task got blocked
	Since int64 `json:"since"`
}

// TaskInfo defines model for TaskInfo.
type TaskInfo struct {
	BlockReason *string `json:"blockReason,omitempty"`
	IsBlocked   *bool   `json:"isBlocked,omitempty"`

	// StartedAt Unix timestamp when task started
	StartedAt *int64 `json:"startedAt,omitempty"`

	// Status Task status (PENDING, IN_PROGRESS, BLOCKED, COMPLETED)
	Status *string `json:"status,omitempty"`
	TaskId *string `json:"taskId,omitempty"`
}

// UtilizationStats defines model for UtilizationStats.
type UtilizationStats struct {
	// ActiveTime Duration
	ActiveTime     *string `json:"activeTime,omitempty"`
	CompletedTasks *int    `json:"completedTasks,omitempty"`
	TotalTasks     *int    `json:"totalTasks,omitempty"`

	// TotalTime Duration
	TotalTime       *string  `json:"totalTime,omitempty"`
	UtilizationRate *float32 `json:"utilizationRate,omitempty"`
}

// WorkerProductivityMetrics defines model for WorkerProductivityMetrics.
type WorkerProductivityMetrics struct {
	DailyStats *map[string]ProductivityStats `json:"dailyStats,omitempty"`
	WorkerId   *string                       `json:"workerId,omitempty"`
}

// WorkerTaskHistory defines model for WorkerTaskHistory.
type WorkerTaskHistory struct {
	Tasks    *[]HistoricalTaskInfo `json:"tasks,omitempty"`
	WorkerId *string               `json:"workerId,omitempty"`
}

// WorkerTaskStatus defines model for WorkerTaskStatus.
type WorkerTaskStatus struct {
	CurrentSite     string     `json:"currentSite"`
	CurrentTasks    []TaskInfo `json:"currentTasks"`
	IsOnBreak       bool       `json:"isOnBreak"`
	IsSessionActive bool       `json:"isSessionActive"`
}

// WorkerUtilizationMetrics defines model for WorkerUtilizationMetrics.
type WorkerUtilizationMetrics struct {
	Stats    *UtilizationStats `json:"stats,omitempty"`
	WorkerId *string           `json:"workerId,omitempty"`
}

// GetSiteProductivityParams defines parameters for GetSiteProductivity.
type GetSiteProductivityParams struct {
	Start *openapi_types.Date `form:"start,omitempty" json:"start,omitempty"`
	End   *openapi_types.Date `form:"end,omitempty" json:"end,omitempty"`
}

// GetSiteTaskDistributionParams defines parameters for GetSiteTaskDistribution.
type GetSiteTaskDistributionParams struct {
	Start *openapi_types.Date `form:"start,omitempty" json:"start,omitempty"`
	End   *openapi_types.Date `form:"end,omitempty" json:"end,omitempty"`
}

// GetSiteUtilizationParams defines parameters for GetSiteUtilization.
type GetSiteUtilizationParams struct {
	Date *openapi_types.Date `form:"date,omitempty" json:"date,omitempty"`
}

// GetWorkerTaskHistoryParams defines parameters for GetWorkerTaskHistory.
type GetWorkerTaskHistoryParams struct {
	Start *openapi_types.Date `form:"start,omitempty" json:"start,omitempty"`
	End   *openapi_types.Date `form:"end,omitempty" json:"end,omitempty"`
}

// GetWorkerProductivityParams defines parameters for GetWorkerProductivity.
type GetWorkerProductivityParams struct {
	Start *openapi_types.Date `form:"start,omitempty" json:"start,omitempty"`
	End   *openapi_types.Date `form:"end,omitempty" json:"end,omitempty"`
}

// GetWorkerUtilizationParams defines parameters for GetWorkerUtilization.
type GetWorkerUtilizationParams struct {
	Date *openapi_types.Date `form:"date,omitempty" json:"date,omitempty"`
}
