package analytics

import "time"

// JobSite represents a job site
type JobSite struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Location string `json:"location"`
}

// Real-time monitoring types
type WorkerTaskStatus struct {
	IsSessionActive bool       `json:"isSessionActive"`
	IsOnBreak       bool       `json:"isOnBreak"`
	CurrentSite     string     `json:"currentSite"` // Job site ID where the worker is currently working
	CurrentTasks    []TaskInfo `json:"currentTasks"`
}

type TaskInfo struct {
	TaskID      string
	Status      TaskStatus
	StartedAt   int64
	IsBlocked   bool
	BlockReason string
}

type TaskBlockageInfo struct {
	IsBlocked   bool
	BlockReason string
	Since       int64
}

type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "PENDING"
	TaskStatusInProgress TaskStatus = "IN_PROGRESS"
	TaskStatusBlocked    TaskStatus = "BLOCKED"
	TaskStatusCompleted  TaskStatus = "COMPLETED"
)

// Historical analytics types

// TaskStatusDistribution represents the distribution of tasks by status
type TaskStatusDistribution struct {
	Date     time.Time
	JobSite  string
	Statuses map[string]int
}

// DailyTaskStats represents task statistics for a single day
type DailyTaskStats struct {
	StatusCounts map[string]int
	Durations    map[string]float64
}

// TaskTrendMetrics represents task trends over time
type TaskTrendMetrics struct {
	JobSite    string
	StartDate  time.Time
	EndDate    time.Time
	DailyStats map[time.Time]DailyTaskStats
}

// BlockageStats represents blockage statistics
type BlockageStats struct {
	Count       int
	AvgDuration time.Duration
}

// BlockageTrendMetrics represents blockage trends over time
type BlockageTrendMetrics struct {
	JobSite    string
	StartDate  time.Time
	EndDate    time.Time
	DailyStats map[time.Time]BlockageStats
}

// ProductivityStats represents worker productivity statistics
type ProductivityStats struct {
	TotalTasks        int
	CompletedTasks    int
	BlockedTasks      int
	AvgCompletionTime time.Duration
	CompletionRate    float64
}

// WorkerProductivityMetrics represents worker productivity over time
type WorkerProductivityMetrics struct {
	WorkerID   string
	StartDate  time.Time
	EndDate    time.Time
	DailyStats map[time.Time]ProductivityStats
}

// HistoricalTaskInfo represents detailed information about a task
type HistoricalTaskInfo struct {
	ID               string
	Name             string
	Description      string
	Status           string
	PlannedStartTime time.Time
	PlannedEndTime   time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
	JobSiteID        string
}

// WorkerTaskHistory represents a worker's task history
type WorkerTaskHistory struct {
	WorkerID  string
	StartDate time.Time
	EndDate   time.Time
	Tasks     []HistoricalTaskInfo
}

// UtilizationStats represents worker utilization statistics
type UtilizationStats struct {
	TotalTime       time.Duration
	TotalTasks      int
	CompletedTasks  int
	ActiveTime      time.Duration
	UtilizationRate float64
}

// WorkerUtilizationMetrics represents worker utilization metrics
type WorkerUtilizationMetrics struct {
	WorkerID string
	Date     time.Time
	Stats    UtilizationStats
}

// SiteProductivityStats represents site productivity statistics
type SiteProductivityStats struct {
	ActiveWorkers     int
	TotalTasks        int
	CompletedTasks    int
	BlockedTasks      int
	AvgCompletionTime time.Duration
	CompletionRate    float64
}

// SiteProductivityMetrics represents site productivity over time
type SiteProductivityMetrics struct {
	JobSite    string
	StartDate  time.Time
	EndDate    time.Time
	DailyStats map[time.Time]SiteProductivityStats
}

// SiteUtilizationStats represents site utilization statistics
type SiteUtilizationStats struct {
	TotalWorkers         int
	OnTimeWorkers        int
	LateWorkers          int
	AvgWorkDuration      time.Duration
	TotalTasks           int
	CompletedTasks       int
	TotalActiveTime      time.Duration
	SiteUtilizationRate  float64
	CheckInDistribution  map[int]int
	CheckOutDistribution map[int]int
}

// SiteUtilizationMetrics represents site utilization metrics
type SiteUtilizationMetrics struct {
	JobSite string
	Date    time.Time
	Stats   SiteUtilizationStats
}

// WorkerTaskStats represents task statistics for a worker
type WorkerTaskStats struct {
	TotalTasks      int
	CompletedTasks  int
	BlockedTasks    int
	AvgTaskDuration time.Duration
	CompletionRate  float64
}

// SiteTaskDistribution represents task distribution across workers
type SiteTaskDistribution struct {
	JobSite     string
	StartDate   time.Time
	EndDate     time.Time
	WorkerStats map[string]WorkerTaskStats
}

// BreakMetrics represents break-related metrics
type BreakMetrics struct {
	TotalBreaks          int
	AverageBreakDuration time.Duration
	BreakDistribution    map[string]int
	ComplianceRate       float64
}
