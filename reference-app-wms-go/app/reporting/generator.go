package reporting

import (
	"context"
	"fmt"
	"log"
	"time"

	"reference-app-wms-go/app/analytics"
)

// ReportGenerator defines the interface for generating reports
type ReportGenerator interface {
	GenerateDaily(ctx context.Context, siteID string, date time.Time) (*DailyReport, error)
	GenerateDashboard(ctx context.Context, filters ReportFilters) (*DashboardData, error)
	GenerateAllSites(ctx context.Context, date time.Time) ([]*DailyReport, error)
}

// Generator implements the ReportGenerator interface
type Generator struct {
	analytics analytics.AnalyticsService
	logger    *log.Logger
}

// NewGenerator creates a new report generator
func NewGenerator(analytics analytics.AnalyticsService) ReportGenerator {
	return &Generator{
		analytics: analytics,
		logger:    log.New(log.Default().Writer(), "[Report Generator] ", log.LstdFlags),
	}
}

// GenerateDaily generates a daily report for a specific site
func (g *Generator) GenerateDaily(ctx context.Context, siteID string, date time.Time) (*DailyReport, error) {
	g.logger.Printf("Generating daily report for site %s on %s", siteID, date.Format("2006-01-02"))

	// Initialize report
	report := &DailyReport{
		Date:   date,
		SiteID: siteID,
	}

	// Get site utilization
	siteUtil, err := g.analytics.GetSiteUtilization(ctx, siteID, date)
	if err != nil {
		return nil, fmt.Errorf("failed to get site utilization: %w", err)
	}

	// Get site productivity
	siteProd, err := g.analytics.GetSiteProductivity(ctx, siteID, date, date)
	if err != nil {
		return nil, fmt.Errorf("failed to get site productivity: %w", err)
	}

	// Get task distribution
	taskDist, err := g.analytics.GetTasksByStatus(ctx, siteID, date)
	if err != nil {
		return nil, fmt.Errorf("failed to get task distribution: %w", err)
	}

	// Get break metrics
	breakMetrics, err := g.analytics.GetBreakMetrics(ctx, siteID, date)
	if err != nil {
		return nil, fmt.Errorf("failed to get break metrics: %w", err)
	}

	// Convert check-in/out distributions to string keys
	checkInDist := make(map[string]int)
	checkOutDist := make(map[string]int)
	for hour, count := range siteUtil.Stats.CheckInDistribution {
		checkInDist[fmt.Sprintf("%02d:00", hour)] = count
	}
	for hour, count := range siteUtil.Stats.CheckOutDistribution {
		checkOutDist[fmt.Sprintf("%02d:00", hour)] = count
	}

	// Populate attendance metrics
	report.Attendance = AttendanceMetrics{
		TotalWorkers:         siteUtil.Stats.TotalWorkers,
		OnTimeWorkers:        siteUtil.Stats.OnTimeWorkers,
		LateWorkers:          siteUtil.Stats.LateWorkers,
		AverageWorkDuration:  siteUtil.Stats.AvgWorkDuration,
		CheckInDistribution:  checkInDist,
		CheckOutDistribution: checkOutDist,
	}

	// Initialize task metrics with data from task distribution
	totalTasks := 0
	completedTasks := 0
	blockedTasks := 0
	for status, count := range taskDist.Statuses {
		totalTasks += count
		if status == "COMPLETED" {
			completedTasks = count
		} else if status == "BLOCKED" {
			blockedTasks = count
		}
	}

	// Calculate completion rate
	completionRate := 0.0
	if totalTasks > 0 {
		completionRate = float64(completedTasks) / float64(totalTasks) * 100
	}

	// Get average task duration from site productivity if available
	var avgTaskDuration time.Duration
	if stats, ok := siteProd.DailyStats[date]; ok {
		avgTaskDuration = stats.AvgCompletionTime
	}

	// Populate task metrics
	report.Tasks = TaskMetrics{
		TotalTasks:          totalTasks,
		CompletedTasks:      completedTasks,
		BlockedTasks:        blockedTasks,
		CompletionRate:      completionRate,
		AverageTaskDuration: avgTaskDuration,
		StatusDistribution:  taskDist.Statuses,
	}

	// Populate break metrics
	report.Breaks = BreakMetrics{
		TotalBreaks:          breakMetrics.TotalBreaks,
		AverageBreakDuration: breakMetrics.AverageBreakDuration,
		BreakDistribution:    breakMetrics.BreakDistribution,
		ComplianceRate:       breakMetrics.ComplianceRate,
	}

	// Populate site metrics
	workerToTaskRatio := 0.0
	if report.Attendance.TotalWorkers > 0 {
		workerToTaskRatio = float64(report.Tasks.TotalTasks) / float64(report.Attendance.TotalWorkers)
	}
	report.Site = SiteMetrics{
		WorkerToTaskRatio: workerToTaskRatio,
		Utilization:       siteUtil.Stats.SiteUtilizationRate,
	}

	// Calculate productivity metrics
	tasksPerWorker := 0.0
	if report.Attendance.TotalWorkers > 0 {
		tasksPerWorker = float64(report.Tasks.CompletedTasks) / float64(report.Attendance.TotalWorkers)
	}
	report.Productivity = ProductivityMetrics{
		TasksPerWorker:    tasksPerWorker,
		AvgCompletionTime: report.Tasks.AverageTaskDuration,
		ScheduleAdherence: report.Tasks.CompletionRate,
	}

	return report, nil
}

// GenerateDashboard generates real-time dashboard data
func (g *Generator) GenerateDashboard(ctx context.Context, filters ReportFilters) (*DashboardData, error) {
	g.logger.Printf("Generating dashboard data for %d sites", len(filters.SiteIDs))

	dashboard := &DashboardData{
		LastUpdated: time.Now(),
		Sites:       make(map[string]SiteDashboard),
	}

	for _, siteID := range filters.SiteIDs {
		// Get current site utilization
		siteUtil, err := g.analytics.GetSiteUtilization(ctx, siteID, time.Now())
		if err != nil {
			g.logger.Printf("Failed to get utilization for site %s: %v", siteID, err)
			continue
		}

		// Get task distribution
		taskDist, err := g.analytics.GetTasksByStatus(ctx, siteID, time.Now())
		if err != nil {
			g.logger.Printf("Failed to get task distribution for site %s: %v", siteID, err)
			continue
		}

		// Create site dashboard
		siteDash := SiteDashboard{
			ActiveWorkers:      siteUtil.Stats.TotalWorkers,
			CurrentUtilization: siteUtil.Stats.SiteUtilizationRate,
		}

		// Count tasks by status
		for status, count := range taskDist.Statuses {
			if status == string(analytics.TaskStatusInProgress) {
				siteDash.TasksInProgress = count
			} else if status == string(analytics.TaskStatusBlocked) {
				siteDash.BlockedTasks = count
			}
		}

		dashboard.Sites[siteID] = siteDash
	}

	return dashboard, nil
}

// GenerateAllSites generates daily reports for all available sites
func (g *Generator) GenerateAllSites(ctx context.Context, date time.Time) ([]*DailyReport, error) {
	g.logger.Printf("Generating reports for all sites on %s", date.Format("2006-01-02"))

	// Get all job sites
	sites, err := g.analytics.GetAllJobSites(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get job sites: %w", err)
	}

	var reports []*DailyReport
	for _, site := range sites {
		report, err := g.GenerateDaily(ctx, site.ID, date)
		if err != nil {
			g.logger.Printf("Failed to generate report for site %s: %v", site.ID, err)
			continue
		}
		reports = append(reports, report)
	}

	return reports, nil
}
