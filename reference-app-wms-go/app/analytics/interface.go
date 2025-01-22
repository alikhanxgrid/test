package analytics

import (
	"context"
	"time"
)

// AnalyticsService defines the interface for analytics operations
type AnalyticsService interface {
	// Real-time monitoring methods
	GetWorkerCurrentTasks(ctx context.Context, workflowID string) (*WorkerTaskStatus, error)
	GetTaskBlockageStatus(ctx context.Context, taskID string, workerID string) (*TaskBlockageInfo, error)

	// Task analytics methods
	GetTasksByStatus(ctx context.Context, jobSiteID string, date time.Time) (*TaskStatusDistribution, error)
	GetTaskTrends(ctx context.Context, jobSiteID string, startDate, endDate time.Time) (*TaskTrendMetrics, error)
	GetBlockageTrends(ctx context.Context, jobSiteID string, startDate, endDate time.Time) (*BlockageTrendMetrics, error)

	// Worker analytics methods
	GetWorkerProductivity(ctx context.Context, workerID string, startDate, endDate time.Time) (*WorkerProductivityMetrics, error)
	GetWorkerTaskHistory(ctx context.Context, workerID string, startDate, endDate time.Time) (*WorkerTaskHistory, error)
	GetWorkerUtilization(ctx context.Context, workerID string, date time.Time) (*WorkerUtilizationMetrics, error)

	// Site analytics methods
	GetSiteProductivity(ctx context.Context, jobSiteID string, startDate, endDate time.Time) (*SiteProductivityMetrics, error)
	GetSiteUtilization(ctx context.Context, jobSiteID string, date time.Time) (*SiteUtilizationMetrics, error)
	GetSiteTaskDistribution(ctx context.Context, jobSiteID string, startDate, endDate time.Time) (*SiteTaskDistribution, error)
	GetBreakMetrics(ctx context.Context, jobSiteID string, date time.Time) (*BreakMetrics, error)
	GetAllJobSites(ctx context.Context) ([]JobSite, error)
}
