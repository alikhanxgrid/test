package impl

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"reference-app-wms-go/app/db"
	"time"

	apiv2 "reference-app-wms-go/app/analytics/api/v2/openapi"

	"github.com/gin-gonic/gin"
	"go.temporal.io/sdk/client"
)

// AnalyticsServiceImpl implements the AnalyticsService interface
type AnalyticsServiceAPIV2 struct {
	temporalClient client.Client
	db             db.DB
	logger         *log.Logger
}

// NewAnalyticsServiceImpl creates a new instance of the analytics service
func NewAnalyticsServiceAPIV2(temporalClient client.Client, db db.DB) *AnalyticsServiceAPIV2 {
	return &AnalyticsServiceAPIV2{
		temporalClient: temporalClient,
		db:             db,
		logger:         log.New(os.Stdout, "[AnalyticsServiceAPIV2] ", log.LstdFlags),
	}
}

// ----------------- Implement the methods from server.gen.go ----------------- //

func (s *AnalyticsServiceAPIV2) GetWorkerCurrentTasks(c *gin.Context, workflowID string) {

	if workflowID == "" {
		s.logger.Println("[GetWorkerCurrentTasks] Missing workflowID parameter")
		c.JSON(http.StatusBadRequest, apiv2.Error{Error: "[GetWorkerCurrentTasks] workflowID is required"})
		return
	}

	s.logger.Printf("[GetWorkerCurrentTasks] Querying workflow %s for current tasks", workflowID)

	response, err := s.temporalClient.QueryWorkflow(c.Request.Context(), workflowID, "", "getCurrentTasks")
	if err != nil {
		s.logger.Printf("[GetWorkerCurrentTasks] Failed to query workflow for current tasks (workflowID=%s): %v", workflowID, err)
		c.JSON(http.StatusInternalServerError, apiv2.Error{Error: err.Error()})
		return
	}

	var workerTaskStatus apiv2.WorkerTaskStatus

	if err := response.Get(&workerTaskStatus); err != nil {
		s.logger.Printf("[GetWorkerCurrentTasks] Failed decoding query result: %v", err)
		c.JSON(http.StatusInternalServerError, apiv2.Error{
			Error: fmt.Sprintf("[GetWorkerCurrentTasks] failed to decode query result: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, workerTaskStatus)
}

func (s *AnalyticsServiceAPIV2) GetTaskBlockageStatus(c *gin.Context, taskID string, workerID string) {

	if taskID == "" || workerID == "" {
		s.logger.Println("[GetTaskBlockageStatus] Missing taskID or workerID parameter")
		c.JSON(http.StatusBadRequest, apiv2.Error{Error: "taskID and workerID are required"})
		return
	}

	s.logger.Printf("[GetTaskBlockageStatus] Querying task %s of worker %s for blockage status", taskID, workerID)

	response, err := s.temporalClient.QueryWorkflow(c.Request.Context(), workerID, "", "getTaskBlockage", taskID)
	if err != nil {
		s.logger.Printf("[GetTaskBlockageStatus] Failed to query workflow for task blockage (workerID=%s, taskID=%s): %v", workerID, taskID, err)
		c.JSON(http.StatusInternalServerError, apiv2.Error{Error: err.Error()})
		return
	}

	var taskBlockageStatus apiv2.TaskBlockageInfo
	if err := response.Get(&taskBlockageStatus); err != nil {
		s.logger.Printf("[GetTaskBlockageStatus] Failed decoding query result: %v", err)
		c.JSON(http.StatusInternalServerError, apiv2.Error{Error: fmt.Sprintf("[GetTaskBlockageStatus] failed to decode query result: %v", err)})
		return
	}

	c.JSON(http.StatusOK, taskBlockageStatus)
}

func (s *AnalyticsServiceAPIV2) GetSiteProductivity(c *gin.Context, siteID string, startDate time.Time, endDate time.Time) {
	if siteID == "" || startDate.IsZero() || endDate.IsZero() {
		s.logger.Println("[GetSiteProductivity] Missing siteID parameter")
		c.JSON(http.StatusBadRequest, apiv2.Error{Error: "[GetSiteProductivity] siteID, start, and end dates are required"})
		return
	}

	s.logger.Printf("[GetSiteProductivity] Querying site %s for productivity metrics from %s to %s",
		siteID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
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
		AvgCompletionTime float64   `db:"avg_completion_time"` // in seconds
		CompletionRate    float64   `db:"completion_rate"`
	}

	var rows []dailyMetrics
	err := s.db.Select(c.Request.Context(), &rows, query, siteID, startDate, endDate)
	if err != nil {
		s.logger.Printf("[GetSiteProductivity] DB query failed: %v", err)
		c.JSON(http.StatusInternalServerError, apiv2.Error{
			Error: fmt.Sprintf("[GetSiteProductivity] query error: %v", err),
		})
		return
	}

	dailyStats := make(map[string]apiv2.SiteProductivityStats)
	for _, dm := range rows {
		// Convert the float seconds to a Go duration, then format as a string
		duration := time.Duration(dm.AvgCompletionTime * float64(time.Second))
		avgTimeString := duration.String() // e.g., "2h34m0s"

		dateKey := dm.Date.Format("2006-01-02") // Use "YYYY-MM-DD" as the map key

		activeWorkers := int(dm.ActiveWorkers)
		totalTasks := int(dm.TotalTasks)
		completedTasks := int(dm.CompletedTasks)
		blockedTasks := int(dm.BlockedTasks)
		rate := dm.CompletionRate

		dailyStats[dateKey] = apiv2.SiteProductivityStats{
			ActiveWorkers:     &activeWorkers,
			TotalTasks:        &totalTasks,
			CompletedTasks:    &completedTasks,
			BlockedTasks:      &blockedTasks,
			AvgCompletionTime: &avgTimeString, // duration in "0s", "34m56s" form
			CompletionRate:    &rate,
		}
	}

	// Construct the top-level SiteProductivityMetrics
	// The openapi model fields are pointers; we set them accordingly.
	jobSiteID := siteID
	startT := startDate
	endT := endDate

	result := apiv2.SiteProductivityMetrics{
		JobSite:    &jobSiteID,
		StartDate:  &startT,
		EndDate:    &endT,
		DailyStats: &map[string]apiv2.SiteProductivityStats{
			// we already filled dailyStats above
		},
	}
	*result.DailyStats = dailyStats

	s.logger.Printf("[GetSiteProductivity] Successfully built productivity metrics for site %s", siteID)
	c.JSON(http.StatusOK, result)
}

func (s *AnalyticsServiceAPIV2) GetSiteTaskDistribution(c *gin.Context, siteID string, startDate time.Time, endDate time.Time) {

	if siteID == "" || startDate.IsZero() || endDate.IsZero() {
		s.logger.Println("[GetSiteTaskDistribution] Missing siteID parameter")
		c.JSON(http.StatusBadRequest, apiv2.Error{Error: "[GetSiteTaskDistribution] siteID, start, and end dates are required"})
		return
	}

	s.logger.Printf("[GetSiteTaskDistribution] Querying site %s for task distribution from %s to %s",
		siteID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

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
	err := s.db.Select(c, &metrics, query, siteID, startDate, endDate)
	if err != nil {
		s.logger.Printf("[GetSiteTaskDistribution] DB query failed: %v", err)
		c.JSON(http.StatusInternalServerError, apiv2.Error{
			Error: fmt.Sprintf("[GetSiteTaskDistribution] query error: %v", err),
		})
		return
	}

	distribution := &apiv2.SiteTaskDistribution{
		JobSite:     &siteID,
		StartDate:   &startDate,
		EndDate:     &endDate,
		WorkerStats: &map[string]apiv2.WorkerTaskStats{},
	}

	for _, m := range metrics {
		(*distribution.WorkerStats)[m.WorkerID] = apiv2.WorkerTaskStats{
			TotalTasks:     &m.TaskCount,
			CompletedTasks: &m.CompletedCount,
			BlockedTasks:   &m.BlockedCount,
			AvgTaskDuration: func() *string {
				duration := time.Duration(m.AvgTaskDuration * float64(time.Second))
				str := duration.String()
				return &str
			}(),
			CompletionRate: &m.CompletionRate,
		}
	}

	c.JSON(http.StatusOK, distribution)
}

func (s *AnalyticsServiceAPIV2) GetSiteUtilization(c *gin.Context, siteID string, startDate time.Time, endDate time.Time) {

	if siteID == "" || startDate.IsZero() || endDate.IsZero() {
		s.logger.Println("[GetSiteUtilization] Missing siteID parameter")
		c.JSON(http.StatusBadRequest, apiv2.Error{Error: "[GetSiteUtilization] siteID, start, and end dates are required"})
		return
	}

	s.logger.Printf("[GetSiteUtilization] Querying site %s for utilization metrics from %s to %s",
		siteID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

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
	err := s.db.Select(c.Request.Context(), &attendance, attendanceQuery, siteID, startDate)
	if err != nil {
		s.logger.Printf("[GetSiteUtilization] DB query failed: %v", err)
		c.JSON(http.StatusInternalServerError, apiv2.Error{
			Error: fmt.Sprintf("[GetSiteUtilization] query error: %v", err),
		})
		return
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
	err = s.db.Get(c.Request.Context(), &metrics, query, siteID, startDate, endDate)
	if err != nil {
		s.logger.Printf("[GetSiteUtilization] DB query failed: %v", err)
		c.JSON(http.StatusInternalServerError, apiv2.Error{
			Error: fmt.Sprintf("[GetSiteUtilization] query error: %v", err),
		})
		return
	}

	s.logger.Printf("Utilization metrics - Workers: %d, AvgDuration: %.2f hours, Tasks: %d, Completed: %d, ActiveTime: %.2f hours, Rate: %.2f%%",
		metrics.TotalWorkers,
		metrics.AvgWorkDuration/3600, // Convert seconds to hours
		metrics.TotalTasks,
		metrics.CompletedTasks,
		metrics.TotalActiveTime/3600, // Convert seconds to hours
		metrics.SiteUtilizationRate*100)

	// Build check-in/out distributions
	checkInDist := make(map[string]int)
	checkOutDist := make(map[string]int)
	for _, a := range attendance {
		hour := int(a.CheckInHour)
		checkInDist[fmt.Sprintf("%d:00", hour)] = a.TotalWorkers

		hour = int(a.CheckOutHour)
		checkOutDist[fmt.Sprintf("%d:00", hour)] = a.TotalWorkers
	}

	s.logger.Printf("Check-in distribution: %v", checkInDist)
	s.logger.Printf("Check-out distribution: %v", checkOutDist)

	utilization := &apiv2.SiteUtilizationMetrics{
		JobSite: &siteID,
		Date:    &startDate,
		Stats: &apiv2.SiteUtilizationStats{
			TotalWorkers:  &metrics.TotalWorkers,
			OnTimeWorkers: &attendance[0].OnTimeWorkers,
			LateWorkers:   &attendance[0].LateWorkers,
			AvgWorkDuration: func() *string {
				duration := time.Duration(metrics.AvgWorkDuration * float64(time.Second))
				str := duration.String()
				return &str
			}(),
			TotalTasks:     &metrics.TotalTasks,
			CompletedTasks: &metrics.CompletedTasks,
			TotalActiveTime: func() *string {
				duration := time.Duration(metrics.TotalActiveTime * float64(time.Second))
				str := duration.String()
				return &str
			}(),
			SiteUtilizationRate:  &metrics.SiteUtilizationRate,
			CheckInDistribution:  &checkInDist,
			CheckOutDistribution: &checkOutDist,
		},
	}

	c.JSON(http.StatusOK, utilization)
}

func (s *AnalyticsServiceAPIV2) GetWorkerTaskHistory(c *gin.Context, workerID string, startDate time.Time, endDate time.Time) {

	if workerID == "" || startDate.IsZero() || endDate.IsZero() {
		s.logger.Println("[GetWorkerTaskHistory] Missing workerID parameter")
		c.JSON(http.StatusBadRequest, apiv2.Error{Error: "[GetWorkerTaskHistory] workerID, start, and end dates are required"})
		return
	}

	s.logger.Printf("[GetWorkerTaskHistory] Querying worker %s for task history from %s to %s",
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

	var tasks []apiv2.HistoricalTaskInfo
	err := s.db.Select(c, &tasks, query, workerID, startDate, endDate)
	if err != nil {
		s.logger.Printf("[GetWorkerTaskHistory] DB query failed: %v", err)
		c.JSON(http.StatusInternalServerError, apiv2.Error{
			Error: fmt.Sprintf("[GetWorkerTaskHistory] query error: %v", err),
		})
		return
	}

	history := &apiv2.WorkerTaskHistory{
		WorkerId:  &workerID,
		StartDate: &startDate,
		EndDate:   &endDate,
		Tasks:     &tasks,
	}

	c.JSON(http.StatusOK, history)
}

func (s *AnalyticsServiceAPIV2) GetWorkerProductivity(c *gin.Context, workerID string, startDate time.Time, endDate time.Time) {

	if workerID == "" || startDate.IsZero() || endDate.IsZero() {
		s.logger.Println("[GetWorkerTaskHistory] Missing workerID parameter")
		c.JSON(http.StatusBadRequest, apiv2.Error{Error: "[GetWorkerTaskHistory] workerID, start, and end dates are required"})
		return
	}

	s.logger.Printf("[GetWorkerProductivity] Querying worker %s for productivity metrics from %s to %s",
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
	err := s.db.Select(c, &metrics, query, workerID, startDate, endDate)
	if err != nil {
		s.logger.Printf("[GetWorkerProductivity] DB query failed: %v", err)
		c.JSON(http.StatusInternalServerError, apiv2.Error{
			Error: fmt.Sprintf("[GetWorkerProductivity] query error: %v", err),
		})
		return
	}

	productivity := &apiv2.WorkerProductivityMetrics{
		WorkerId:   &workerID,
		StartDate:  &startDate,
		EndDate:    &endDate,
		DailyStats: &map[string]apiv2.ProductivityStats{},
	}

	for _, m := range metrics {
		(*productivity.DailyStats)[m.Date.Format("2006-01-02")] = apiv2.ProductivityStats{
			TotalTasks:     &m.TotalTasks,
			CompletedTasks: &m.CompletedTasks,
			BlockedTasks:   &m.BlockedTasks,
			AvgCompletionTime: func() *string {
				duration := time.Duration(m.AvgCompletionTime * float64(time.Second))
				str := duration.String()
				return &str
			}(),
			CompletionRate: &m.CompletionRate,
		}
	}

	c.JSON(http.StatusOK, productivity)
}

func (s *AnalyticsServiceAPIV2) GetWorkerUtilization(c *gin.Context, workerID string, date time.Time) {
	if workerID == "" || date.IsZero() {
		s.logger.Println("[GetWorkerUtilization] Missing workerID parameter")
		c.JSON(http.StatusBadRequest, apiv2.Error{Error: "[GetWorkerUtilization] workerID and date are required"})
		return
	}

	s.logger.Printf("[GetWorkerUtilization] Querying worker %s for utilization metrics from %s",
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
	err := s.db.Get(c, &metrics, query, workerID, date)
	if err != nil {
		s.logger.Printf("[GetWorkerUtilization] DB query failed: %v", err)
		c.JSON(http.StatusInternalServerError, apiv2.Error{
			Error: fmt.Sprintf("[GetWorkerUtilization] query error: %v", err),
		})
		return
	}

	utilization := &apiv2.WorkerUtilizationMetrics{
		WorkerId: workerID,
		Date:     date,
		Stats: &apiv2.UtilizationStats{
			TotalTime: func() *string {
				duration := time.Duration(metrics.TotalTime * float64(time.Second))
				str := duration.String()
				return &str
			}(),
			TotalTasks:     &metrics.TotalTasks,
			CompletedTasks: &metrics.CompletedTasks,
			ActiveTime: func() *string {
				duration := time.Duration(metrics.ActiveTime * float64(time.Second))
				str := duration.String()
				return &str
			}(),
			UtilizationRate: &metrics.UtilizationRate,
		},
	}

	c.JSON(http.StatusOK, utilization)
}
