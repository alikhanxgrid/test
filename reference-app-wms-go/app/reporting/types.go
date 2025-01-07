package reporting

import (
	"time"
)

// ReportType defines the type of report to generate
type ReportType string

const (
	ReportTypeDaily     ReportType = "daily"
	ReportTypeDashboard ReportType = "dashboard"
)

// OutputFormat defines the format of the report output
type OutputFormat string

const (
	FormatJSON OutputFormat = "json"
	FormatHTML OutputFormat = "html"
)

// TimeRange represents a time period for the report
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// ReportFilters defines filters for the report data
type ReportFilters struct {
	SiteIDs   []string
	WorkerIDs []string
}

// ReportConfig defines the configuration for report generation
type ReportConfig struct {
	Type      ReportType
	TimeRange TimeRange
	Filters   ReportFilters
	Format    OutputFormat
}

// AttendanceMetrics represents attendance-related metrics
type AttendanceMetrics struct {
	TotalWorkers         int
	OnTimeWorkers        int
	LateWorkers          int
	AverageWorkDuration  time.Duration
	CheckInDistribution  map[string]int // Hour -> Count
	CheckOutDistribution map[string]int // Hour -> Count
}

// TaskMetrics represents task-related metrics
type TaskMetrics struct {
	TotalTasks          int
	CompletedTasks      int
	BlockedTasks        int
	CompletionRate      float64
	AverageTaskDuration time.Duration
	StatusDistribution  map[string]int
}

// BreakMetrics represents break-related metrics
type BreakMetrics struct {
	TotalBreaks          int
	AverageBreakDuration time.Duration
	BreakDistribution    map[string]int // Hour -> Count
	ComplianceRate       float64        // % of breaks within standard duration
}

// ProductivityMetrics represents productivity-related metrics
type ProductivityMetrics struct {
	TasksPerWorker     float64
	AvgCompletionTime  time.Duration
	BlockageResolution time.Duration // Average time to resolve blockages
	ScheduleAdherence  float64       // % of tasks completed within scheduled time
}

// SiteMetrics represents site-level metrics
type SiteMetrics struct {
	WorkerToTaskRatio float64
	PeakHours         []string
	Utilization       float64
	ScheduleCoverage  float64
}

// DailyReport represents a complete daily report
type DailyReport struct {
	Date         time.Time
	SiteID       string
	Attendance   AttendanceMetrics
	Tasks        TaskMetrics
	Breaks       BreakMetrics
	Productivity ProductivityMetrics
	Site         SiteMetrics
}

// DashboardData represents real-time dashboard metrics
type DashboardData struct {
	LastUpdated time.Time
	Sites       map[string]SiteDashboard
}

// SiteDashboard represents real-time metrics for a site
type SiteDashboard struct {
	ActiveWorkers      int
	TasksInProgress    int
	BlockedTasks       int
	CurrentUtilization float64
	RecentBreaks       int // Breaks in last hour
	AlertCount         int // Number of active alerts/issues
}
