package analytics

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"reference-app-wms-go/app/db"

	"go.temporal.io/sdk/client"
)

// AnalyticsServiceImpl implements the AnalyticsService interface
type AnalyticsServiceImpl struct {
	temporalClient client.Client
	db             db.DB
	logger         *log.Logger
}

// NewAnalyticsService creates a new instance of the analytics service
func NewAnalyticsService(temporalClient client.Client, db db.DB) AnalyticsService {
	return &AnalyticsServiceImpl{
		temporalClient: temporalClient,
		db:             db,
		logger:         log.New(os.Stdout, "[AnalyticsService] ", log.LstdFlags),
	}
}

// Real-time monitoring methods

func (s *AnalyticsServiceImpl) GetWorkerCurrentTasks(ctx context.Context, workflowID string) (*WorkerTaskStatus, error) {
	s.logger.Printf("Getting current tasks for workflow: %s", workflowID)

	var result WorkerTaskStatus
	response, err := s.temporalClient.QueryWorkflow(ctx, workflowID, "", "getCurrentTasks")
	if err != nil {
		s.logger.Printf("Failed to query workflow for current tasks: %v", err)
		return nil, err
	}

	err = response.Get(&result)
	if err != nil {
		s.logger.Printf("Failed to decode query result: %v", err)
		return nil, err
	}

	s.logger.Printf("Successfully retrieved tasks for workflow %s: active=%v, site=%s, tasks=%d",
		workflowID, result.IsSessionActive, result.CurrentSite, len(result.CurrentTasks))
	return &result, nil
}

func (s *AnalyticsServiceImpl) GetTaskBlockageStatus(ctx context.Context, taskID string, workerID string) (*TaskBlockageInfo, error) {
	s.logger.Printf("Getting blockage status for task %s of worker %s", taskID, workerID)

	var result TaskBlockageInfo
	response, err := s.temporalClient.QueryWorkflow(ctx, workerID, "", "getTaskBlockage", taskID)
	if err != nil {
		s.logger.Printf("Failed to query workflow for task blockage: %v", err)
		return nil, fmt.Errorf("failed to query task blockage: %w", err)
	}

	err = response.Get(&result)
	if err != nil {
		s.logger.Printf("Failed to decode query result: %v", err)
		return nil, fmt.Errorf("failed to decode query result: %w", err)
	}

	return &result, nil
}

// Task analytics methods

func (s *AnalyticsServiceImpl) GetTasksByStatus(ctx context.Context, jobSiteID string, date time.Time) (*TaskStatusDistribution, error) {
	s.logger.Printf("Getting task status distribution for job site %s on %s", jobSiteID, date.Format("2006-01-02"))

	// First, let's check if we have any tasks at all for this site
	checkQuery := `
		SELECT COUNT(*) 
		FROM tasks t 
		JOIN schedules s ON t.schedule_id = s.id 
		WHERE s.job_site_id = $1`

	var totalCount int
	err := s.db.Get(ctx, &totalCount, checkQuery, jobSiteID)
	if err != nil {
		return nil, fmt.Errorf("failed to check total tasks: %w", err)
	}
	s.logger.Printf("Total tasks found for site: %d", totalCount)

	query := `
		WITH task_stats AS (
			SELECT 
				COALESCE(t.status, 'PENDING') as status,
				COUNT(*) as count
			FROM tasks t
			JOIN schedules s ON t.schedule_id = s.id
			WHERE s.job_site_id = $1
			AND DATE(t.planned_start_time) = DATE($2)
			GROUP BY t.status
		)
		SELECT 
			status,
			count
		FROM task_stats
	`

	type statusCount struct {
		Status string `db:"status"`
		Count  int    `db:"count"`
	}

	var counts []statusCount
	err = s.db.Select(ctx, &counts, query, jobSiteID, date)
	if err != nil {
		return nil, fmt.Errorf("failed to get task distribution: %w", err)
	}

	s.logger.Printf("Found %d different status types", len(counts))
	for _, count := range counts {
		s.logger.Printf("Status: %s, Count: %d", count.Status, count.Count)
	}

	// Let's also check tasks for this specific date
	dateCheckQuery := `
		SELECT COUNT(*) 
		FROM tasks t 
		JOIN schedules s ON t.schedule_id = s.id 
		WHERE s.job_site_id = $1 
		AND DATE(t.planned_start_time) = DATE($2)`

	var dateCount int
	err = s.db.Get(ctx, &dateCount, dateCheckQuery, jobSiteID, date)
	if err != nil {
		return nil, fmt.Errorf("failed to check date tasks: %w", err)
	}
	s.logger.Printf("Tasks found for date %s: %d", date.Format("2006-01-02"), dateCount)

	distribution := &TaskStatusDistribution{
		Date:     date,
		JobSite:  jobSiteID,
		Statuses: make(map[string]int),
	}

	// Initialize all possible statuses with 0 count
	distribution.Statuses["PENDING"] = 0
	distribution.Statuses["IN_PROGRESS"] = 0
	distribution.Statuses["COMPLETED"] = 0
	distribution.Statuses["BLOCKED"] = 0

	// Update counts for statuses that exist
	for _, count := range counts {
		distribution.Statuses[count.Status] = count.Count
	}

	return distribution, nil
}

func (s *AnalyticsServiceImpl) GetTaskTrends(ctx context.Context, jobSiteID string, startDate, endDate time.Time) (*TaskTrendMetrics, error) {
	s.logger.Printf("Getting task trends for job site %s from %s to %s",
		jobSiteID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	query := `
		SELECT 
			DATE(t.planned_start_time) as date,
			t.status,
			COUNT(*) as count,
			AVG(EXTRACT(EPOCH FROM (t.updated_at - t.created_at))) as avg_duration
		FROM tasks t
		JOIN schedules s ON t.schedule_id = s.id
		WHERE s.job_site_id = $1
		AND DATE(t.planned_start_time) BETWEEN DATE($2) AND DATE($3)
		GROUP BY DATE(t.planned_start_time), t.status
		ORDER BY DATE(t.planned_start_time), t.status
	`

	type dailyMetrics struct {
		Date        time.Time `db:"date"`
		Status      string    `db:"status"`
		Count       int       `db:"count"`
		AvgDuration float64   `db:"avg_duration"`
	}

	var metrics []dailyMetrics
	err := s.db.Select(ctx, &metrics, query, jobSiteID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get task trends: %w", err)
	}

	trends := &TaskTrendMetrics{
		JobSite:    jobSiteID,
		StartDate:  startDate,
		EndDate:    endDate,
		DailyStats: make(map[time.Time]DailyTaskStats),
	}

	for _, m := range metrics {
		stats, exists := trends.DailyStats[m.Date]
		if !exists {
			stats = DailyTaskStats{
				StatusCounts: make(map[string]int),
				Durations:    make(map[string]float64),
			}
		}
		stats.StatusCounts[m.Status] = m.Count
		stats.Durations[m.Status] = m.AvgDuration
		trends.DailyStats[m.Date] = stats
	}

	return trends, nil
}

func (s *AnalyticsServiceImpl) GetBlockageTrends(ctx context.Context, jobSiteID string, startDate, endDate time.Time) (*BlockageTrendMetrics, error) {
	s.logger.Printf("Getting blockage trends for job site %s from %s to %s",
		jobSiteID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	query := `
		WITH task_status_changes AS (
			SELECT 
				t.id,
				t.schedule_id,
				t.status,
				t.updated_at,
				LAG(t.status) OVER (PARTITION BY t.id ORDER BY t.updated_at) as prev_status,
				LAG(t.updated_at) OVER (PARTITION BY t.id ORDER BY t.updated_at) as prev_update
			FROM tasks t
			JOIN schedules s ON t.schedule_id = s.id
			WHERE s.job_site_id = $1
			AND DATE(t.planned_start_time) BETWEEN DATE($2) AND DATE($3)
		)
		SELECT 
			DATE(updated_at) as date,
			COUNT(*) as blockage_count,
			AVG(EXTRACT(EPOCH FROM (updated_at - prev_update))) as avg_blockage_duration
		FROM task_status_changes
		WHERE status = 'IN_PROGRESS' AND prev_status = 'BLOCKED'
		GROUP BY DATE(updated_at)
		ORDER BY DATE(updated_at)
	`

	type blockageMetrics struct {
		Date                time.Time `db:"date"`
		BlockageCount       int       `db:"blockage_count"`
		AvgBlockageDuration float64   `db:"avg_blockage_duration"`
	}

	var metrics []blockageMetrics
	err := s.db.Select(ctx, &metrics, query, jobSiteID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get blockage trends: %w", err)
	}

	trends := &BlockageTrendMetrics{
		JobSite:    jobSiteID,
		StartDate:  startDate,
		EndDate:    endDate,
		DailyStats: make(map[time.Time]BlockageStats),
	}

	for _, m := range metrics {
		trends.DailyStats[m.Date] = BlockageStats{
			Count:       m.BlockageCount,
			AvgDuration: time.Duration(m.AvgBlockageDuration * float64(time.Second)),
		}
	}

	return trends, nil
}

// Worker analytics methods

func (s *AnalyticsServiceImpl) GetWorkerProductivity(ctx context.Context, workerID string, startDate, endDate time.Time) (*WorkerProductivityMetrics, error) {
	s.logger.Printf("Getting productivity metrics for worker %s from %s to %s",
		workerID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	query := `
		WITH task_metrics AS (
			SELECT 
				DATE(t.planned_start_time) as date,
				COUNT(*) as total_tasks,
				COUNT(CASE WHEN t.status = 'COMPLETED' THEN 1 END) as completed_tasks,
				COUNT(CASE WHEN t.status = 'BLOCKED' THEN 1 END) as blocked_tasks,
				AVG(CASE WHEN t.status = 'COMPLETED' 
					THEN EXTRACT(EPOCH FROM (t.updated_at - t.created_at)) 
				END) as avg_completion_time
			FROM tasks t
			WHERE t.worker_id = $1
			AND DATE(t.planned_start_time) BETWEEN DATE($2) AND DATE($3)
			GROUP BY DATE(t.planned_start_time)
		)
		SELECT 
			date,
			total_tasks,
			completed_tasks,
			blocked_tasks,
			avg_completion_time,
			CASE WHEN total_tasks > 0 
				THEN (completed_tasks::float / total_tasks::float) 
				ELSE 0 
			END as completion_rate
		FROM task_metrics
		ORDER BY date
	`

	type dailyMetrics struct {
		Date              time.Time `db:"date"`
		TotalTasks        int       `db:"total_tasks"`
		CompletedTasks    int       `db:"completed_tasks"`
		BlockedTasks      int       `db:"blocked_tasks"`
		AvgCompletionTime float64   `db:"avg_completion_time"`
		CompletionRate    float64   `db:"completion_rate"`
	}

	var metrics []dailyMetrics
	err := s.db.Select(ctx, &metrics, query, workerID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get worker productivity: %w", err)
	}

	productivity := &WorkerProductivityMetrics{
		WorkerID:   workerID,
		StartDate:  startDate,
		EndDate:    endDate,
		DailyStats: make(map[time.Time]ProductivityStats),
	}

	for _, m := range metrics {
		productivity.DailyStats[m.Date] = ProductivityStats{
			TotalTasks:        m.TotalTasks,
			CompletedTasks:    m.CompletedTasks,
			BlockedTasks:      m.BlockedTasks,
			AvgCompletionTime: time.Duration(m.AvgCompletionTime * float64(time.Second)),
			CompletionRate:    m.CompletionRate,
		}
	}

	return productivity, nil
}

func (s *AnalyticsServiceImpl) GetWorkerTaskHistory(ctx context.Context, workerID string, startDate, endDate time.Time) (*WorkerTaskHistory, error) {
	s.logger.Printf("Getting task history for worker %s from %s to %s",
		workerID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	query := `
		SELECT 
			t.id,
			t.name,
			t.description,
			t.status,
			t.planned_start_time,
			t.planned_end_time,
			t.created_at,
			t.updated_at,
			s.job_site_id
		FROM tasks t
		JOIN schedules s ON t.schedule_id = s.id
		WHERE t.worker_id = $1
		AND DATE(t.planned_start_time) BETWEEN DATE($2) AND DATE($3)
		ORDER BY t.planned_start_time DESC
	`

	var tasks []HistoricalTaskInfo
	err := s.db.Select(ctx, &tasks, query, workerID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get worker task history: %w", err)
	}

	history := &WorkerTaskHistory{
		WorkerID:  workerID,
		StartDate: startDate,
		EndDate:   endDate,
		Tasks:     tasks,
	}

	return history, nil
}

func (s *AnalyticsServiceImpl) GetWorkerUtilization(ctx context.Context, workerID string, date time.Time) (*WorkerUtilizationMetrics, error) {
	s.logger.Printf("Getting utilization metrics for worker %s on %s",
		workerID, date.Format("2006-01-02"))

	query := `
		WITH time_metrics AS (
			SELECT 
				wa.worker_id,
				wa.check_in_time,
				wa.check_out_time,
				EXTRACT(EPOCH FROM (wa.check_out_time - wa.check_in_time)) as total_time,
				COUNT(t.id) as total_tasks,
				COUNT(CASE WHEN t.status = 'COMPLETED' THEN 1 END) as completed_tasks,
				SUM(CASE WHEN t.status = 'COMPLETED' 
					THEN EXTRACT(EPOCH FROM (t.updated_at - t.created_at))
				END) as active_time
			FROM worker_attendance wa
			LEFT JOIN tasks t ON t.worker_id = wa.worker_id 
				AND DATE(t.planned_start_time) = DATE(wa.date)
			WHERE wa.worker_id = $1
			AND DATE(wa.date) = DATE($2)
			GROUP BY wa.worker_id, wa.check_in_time, wa.check_out_time
		)
		SELECT 
			total_time,
			total_tasks,
			completed_tasks,
			active_time,
			CASE WHEN total_time > 0 
				THEN (active_time / total_time) 
				ELSE 0 
			END as utilization_rate
		FROM time_metrics
	`

	type utilizationMetrics struct {
		TotalTime       float64 `db:"total_time"`
		TotalTasks      int     `db:"total_tasks"`
		CompletedTasks  int     `db:"completed_tasks"`
		ActiveTime      float64 `db:"active_time"`
		UtilizationRate float64 `db:"utilization_rate"`
	}

	var metrics utilizationMetrics
	err := s.db.Get(ctx, &metrics, query, workerID, date)
	if err != nil {
		return nil, fmt.Errorf("failed to get worker utilization: %w", err)
	}

	utilization := &WorkerUtilizationMetrics{
		WorkerID: workerID,
		Date:     date,
		Stats: UtilizationStats{
			TotalTime:       time.Duration(metrics.TotalTime * float64(time.Second)),
			TotalTasks:      metrics.TotalTasks,
			CompletedTasks:  metrics.CompletedTasks,
			ActiveTime:      time.Duration(metrics.ActiveTime * float64(time.Second)),
			UtilizationRate: metrics.UtilizationRate,
		},
	}

	return utilization, nil
}

// Site analytics methods

func (s *AnalyticsServiceImpl) GetSiteProductivity(ctx context.Context, jobSiteID string, startDate, endDate time.Time) (*SiteProductivityMetrics, error) {
	s.logger.Printf("Getting productivity metrics for job site %s from %s to %s",
		jobSiteID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	query := `
		WITH daily_metrics AS (
			SELECT 
				DATE(t.planned_start_time) as date,
				COUNT(DISTINCT t.worker_id) as active_workers,
				COUNT(*) as total_tasks,
				COUNT(CASE WHEN t.status = 'COMPLETED' THEN 1 END) as completed_tasks,
				COUNT(CASE WHEN t.status = 'BLOCKED' THEN 1 END) as blocked_tasks,
				AVG(CASE WHEN t.status = 'COMPLETED' 
					THEN EXTRACT(EPOCH FROM (t.updated_at - t.created_at)) 
				END) as avg_completion_time
			FROM tasks t
			JOIN schedules s ON t.schedule_id = s.id
			WHERE s.job_site_id = $1
			AND DATE(t.planned_start_time) BETWEEN DATE($2) AND DATE($3)
			GROUP BY DATE(t.planned_start_time)
		)
		SELECT 
			date,
			active_workers,
			total_tasks,
			completed_tasks,
			blocked_tasks,
			avg_completion_time,
			CASE WHEN total_tasks > 0 
				THEN (completed_tasks::float / total_tasks::float) 
				ELSE 0 
			END as completion_rate
		FROM daily_metrics
		ORDER BY date
	`

	type dailyMetrics struct {
		Date              time.Time `db:"date"`
		ActiveWorkers     int       `db:"active_workers"`
		TotalTasks        int       `db:"total_tasks"`
		CompletedTasks    int       `db:"completed_tasks"`
		BlockedTasks      int       `db:"blocked_tasks"`
		AvgCompletionTime float64   `db:"avg_completion_time"`
		CompletionRate    float64   `db:"completion_rate"`
	}

	var metrics []dailyMetrics
	err := s.db.Select(ctx, &metrics, query, jobSiteID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get site productivity: %w", err)
	}

	productivity := &SiteProductivityMetrics{
		JobSite:    jobSiteID,
		StartDate:  startDate,
		EndDate:    endDate,
		DailyStats: make(map[time.Time]SiteProductivityStats),
	}

	for _, m := range metrics {
		productivity.DailyStats[m.Date] = SiteProductivityStats{
			ActiveWorkers:     m.ActiveWorkers,
			TotalTasks:        m.TotalTasks,
			CompletedTasks:    m.CompletedTasks,
			BlockedTasks:      m.BlockedTasks,
			AvgCompletionTime: time.Duration(m.AvgCompletionTime * float64(time.Second)),
			CompletionRate:    m.CompletionRate,
		}
	}

	return productivity, nil
}

func (s *AnalyticsServiceImpl) GetSiteUtilization(ctx context.Context, jobSiteID string, date time.Time) (*SiteUtilizationMetrics, error) {
	s.logger.Printf("Getting utilization metrics for job site %s on %s",
		jobSiteID, date.Format("2006-01-02"))

	// First, get attendance details
	attendanceQuery := `
		SELECT 
			COUNT(*) as total_workers,
			COUNT(CASE WHEN check_in_time <= date + interval '15 minutes' THEN 1 END) as on_time_workers,
			COUNT(CASE WHEN check_in_time > date + interval '15 minutes' THEN 1 END) as late_workers,
			EXTRACT(HOUR FROM check_in_time) as check_in_hour,
			EXTRACT(HOUR FROM check_out_time) as check_out_hour
		FROM worker_attendance
		WHERE job_site_id = $1 
		AND DATE(date) = DATE($2)
		GROUP BY EXTRACT(HOUR FROM check_in_time), EXTRACT(HOUR FROM check_out_time)`

	type attendanceStats struct {
		TotalWorkers  int     `db:"total_workers"`
		OnTimeWorkers int     `db:"on_time_workers"`
		LateWorkers   int     `db:"late_workers"`
		CheckInHour   float64 `db:"check_in_hour"`
		CheckOutHour  float64 `db:"check_out_hour"`
	}

	var attendance []attendanceStats
	err := s.db.Select(ctx, &attendance, attendanceQuery, jobSiteID, date)
	if err != nil {
		return nil, fmt.Errorf("failed to get attendance stats: %w", err)
	}

	s.logger.Printf("Found attendance records: %d", len(attendance))
	for _, a := range attendance {
		s.logger.Printf("Attendance stats - Total: %d, OnTime: %d, Late: %d, CheckIn: %.0f:00, CheckOut: %.0f:00",
			a.TotalWorkers, a.OnTimeWorkers, a.LateWorkers, a.CheckInHour, a.CheckOutHour)
	}

	// Get utilization metrics
	query := `
		WITH site_stats AS (
			SELECT 
				COUNT(DISTINCT wa.worker_id) as total_workers,
				AVG(EXTRACT(EPOCH FROM (COALESCE(wa.check_out_time, NOW()) - wa.check_in_time))) as avg_work_duration,
				COUNT(t.id) as total_tasks,
				SUM(CASE WHEN t.status = 'COMPLETED' THEN 1 ELSE 0 END) as completed_tasks,
				SUM(EXTRACT(EPOCH FROM (COALESCE(t.updated_at, NOW()) - t.created_at))) as total_active_time
			FROM worker_attendance wa
			LEFT JOIN tasks t ON t.worker_id = wa.worker_id
			JOIN schedules s ON t.schedule_id = s.id
			WHERE wa.job_site_id = $1
			AND DATE(wa.date) = DATE($2)
			AND s.job_site_id = wa.job_site_id
		)
		SELECT 
			total_workers,
			avg_work_duration,
			total_tasks,
			completed_tasks,
			total_active_time,
			CASE 
				WHEN total_workers > 0 AND total_tasks > 0 
				THEN (completed_tasks::float / total_tasks::float) * 
					 (total_active_time::float / (total_workers * GREATEST(avg_work_duration, 1))::float)
				ELSE 0 
			END as site_utilization_rate
		FROM site_stats`

	type utilizationMetrics struct {
		TotalWorkers        int     `db:"total_workers"`
		AvgWorkDuration     float64 `db:"avg_work_duration"`
		TotalTasks          int     `db:"total_tasks"`
		CompletedTasks      int     `db:"completed_tasks"`
		TotalActiveTime     float64 `db:"total_active_time"`
		SiteUtilizationRate float64 `db:"site_utilization_rate"`
	}

	var metrics utilizationMetrics
	err = s.db.Get(ctx, &metrics, query, jobSiteID, date)
	if err != nil {
		return nil, fmt.Errorf("failed to get site utilization: %w", err)
	}

	s.logger.Printf("Utilization metrics - Workers: %d, AvgDuration: %.2f hours, Tasks: %d, Completed: %d, ActiveTime: %.2f hours, Rate: %.2f%%",
		metrics.TotalWorkers,
		metrics.AvgWorkDuration/3600, // Convert seconds to hours
		metrics.TotalTasks,
		metrics.CompletedTasks,
		metrics.TotalActiveTime/3600, // Convert seconds to hours
		metrics.SiteUtilizationRate*100)

	// Build check-in/out distributions
	checkInDist := make(map[int]int)
	checkOutDist := make(map[int]int)
	for _, a := range attendance {
		hour := int(a.CheckInHour)
		checkInDist[hour] = a.TotalWorkers

		hour = int(a.CheckOutHour)
		checkOutDist[hour] = a.TotalWorkers
	}

	s.logger.Printf("Check-in distribution: %v", checkInDist)
	s.logger.Printf("Check-out distribution: %v", checkOutDist)

	utilization := &SiteUtilizationMetrics{
		JobSite: jobSiteID,
		Date:    date,
		Stats: SiteUtilizationStats{
			TotalWorkers:         metrics.TotalWorkers,
			OnTimeWorkers:        attendance[0].OnTimeWorkers,
			LateWorkers:          attendance[0].LateWorkers,
			AvgWorkDuration:      time.Duration(metrics.AvgWorkDuration * float64(time.Second)),
			TotalTasks:           metrics.TotalTasks,
			CompletedTasks:       metrics.CompletedTasks,
			TotalActiveTime:      time.Duration(metrics.TotalActiveTime * float64(time.Second)),
			SiteUtilizationRate:  metrics.SiteUtilizationRate,
			CheckInDistribution:  checkInDist,
			CheckOutDistribution: checkOutDist,
		},
	}

	return utilization, nil
}

func (s *AnalyticsServiceImpl) GetSiteTaskDistribution(ctx context.Context, jobSiteID string, startDate, endDate time.Time) (*SiteTaskDistribution, error) {
	s.logger.Printf("Getting task distribution for job site %s from %s to %s",
		jobSiteID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	query := `
		WITH worker_task_counts AS (
			SELECT 
				t.worker_id,
				COUNT(*) as task_count,
				COUNT(CASE WHEN t.status = 'COMPLETED' THEN 1 END) as completed_count,
				COUNT(CASE WHEN t.status = 'BLOCKED' THEN 1 END) as blocked_count,
				AVG(EXTRACT(EPOCH FROM (t.updated_at - t.created_at))) as avg_task_duration
			FROM tasks t
			JOIN schedules s ON t.schedule_id = s.id
			WHERE s.job_site_id = $1
			AND DATE(t.planned_start_time) BETWEEN DATE($2) AND DATE($3)
			GROUP BY t.worker_id
		)
		SELECT 
			worker_id,
			task_count,
			completed_count,
			blocked_count,
			avg_task_duration,
			CASE WHEN task_count > 0 
				THEN (completed_count::float / task_count::float) 
				ELSE 0 
			END as completion_rate
		FROM worker_task_counts
		ORDER BY task_count DESC
	`

	type workerMetrics struct {
		WorkerID        string  `db:"worker_id"`
		TaskCount       int     `db:"task_count"`
		CompletedCount  int     `db:"completed_count"`
		BlockedCount    int     `db:"blocked_count"`
		AvgTaskDuration float64 `db:"avg_task_duration"`
		CompletionRate  float64 `db:"completion_rate"`
	}

	var metrics []workerMetrics
	err := s.db.Select(ctx, &metrics, query, jobSiteID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get site task distribution: %w", err)
	}

	distribution := &SiteTaskDistribution{
		JobSite:     jobSiteID,
		StartDate:   startDate,
		EndDate:     endDate,
		WorkerStats: make(map[string]WorkerTaskStats),
	}

	for _, m := range metrics {
		distribution.WorkerStats[m.WorkerID] = WorkerTaskStats{
			TotalTasks:      m.TaskCount,
			CompletedTasks:  m.CompletedCount,
			BlockedTasks:    m.BlockedCount,
			AvgTaskDuration: time.Duration(m.AvgTaskDuration * float64(time.Second)),
			CompletionRate:  m.CompletionRate,
		}
	}

	return distribution, nil
}

// GetAllJobSites retrieves all job sites
func (s *AnalyticsServiceImpl) GetAllJobSites(ctx context.Context) ([]JobSite, error) {
	s.logger.Printf("Getting all job sites")

	query := `
		SELECT id, name, location
		FROM job_sites
		ORDER BY name
	`

	var sites []JobSite
	err := s.db.Select(ctx, &sites, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get job sites: %w", err)
	}

	return sites, nil
}

// GetBreakMetrics retrieves break-related metrics for a site
func (s *AnalyticsServiceImpl) GetBreakMetrics(ctx context.Context, jobSiteID string, date time.Time) (*BreakMetrics, error) {
	s.logger.Printf("Getting break metrics for job site %s on %s", jobSiteID, date.Format("2006-01-02"))

	// Simplified query to calculate breaks based on attendance records
	query := `
		WITH attendance_stats AS (
			SELECT 
				EXTRACT(HOUR FROM check_in_time) as hour,
				COUNT(DISTINCT worker_id) as worker_count,
				AVG(EXTRACT(EPOCH FROM (check_out_time - check_in_time))) as total_time
			FROM worker_attendance
			WHERE job_site_id = $1
			AND DATE(date) = DATE($2)
			GROUP BY EXTRACT(HOUR FROM check_in_time)
		)
		SELECT 
			hour,
			worker_count,
			CASE 
				WHEN total_time > 3600 THEN total_time - 3600 -- Assume 1 hour of active work per hour
				ELSE 0 
			END as break_time
		FROM attendance_stats
		ORDER BY hour`

	type breakStats struct {
		Hour        float64 `db:"hour"`
		WorkerCount int     `db:"worker_count"`
		BreakTime   float64 `db:"break_time"`
	}

	var stats []breakStats
	err := s.db.Select(ctx, &stats, query, jobSiteID, date)
	if err != nil {
		return nil, fmt.Errorf("failed to get break metrics: %w", err)
	}

	s.logger.Printf("Found attendance records for %d time slots", len(stats))

	// Calculate metrics
	totalBreaks := 0
	totalBreakTime := 0.0
	distribution := make(map[string]int)

	for _, stat := range stats {
		if stat.BreakTime > 0 {
			totalBreaks += stat.WorkerCount
			totalBreakTime += stat.BreakTime * float64(stat.WorkerCount)
			hour := fmt.Sprintf("%02.0f:00", stat.Hour)
			distribution[hour] = stat.WorkerCount
			s.logger.Printf("Hour %s: %d workers, break time: %.2f minutes",
				hour, stat.WorkerCount, stat.BreakTime/60)
		}
	}

	var avgDuration time.Duration
	if totalBreaks > 0 {
		avgDuration = time.Duration((totalBreakTime / float64(totalBreaks)) * float64(time.Second))
	}

	s.logger.Printf("Break summary - Total breaks: %d, Average duration: %v", totalBreaks, avgDuration)

	metrics := &BreakMetrics{
		TotalBreaks:          totalBreaks,
		AverageBreakDuration: avgDuration,
		BreakDistribution:    distribution,
		ComplianceRate:       100.0, // All breaks are considered compliant in this simplified model
	}

	return metrics, nil
}
