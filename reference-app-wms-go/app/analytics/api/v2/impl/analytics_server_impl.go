package impl

import (
	"fmt"
	"net/http"
	"time"

	"reference-app-wms-go/app/analytics"
	apiv2 "reference-app-wms-go/app/analytics/api/v2/openapi"

	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// AnalyticsServerImpl implements apiv2.ServerInterface for the analytics API v2.
type AnalyticsServerImpl struct {
	analyticsService analytics.AnalyticsService
}

// NewAnalyticsServerImpl creates a new AnalyticsServerImpl.
func NewAnalyticsServerImpl(service analytics.AnalyticsService) *AnalyticsServerImpl {
	return &AnalyticsServerImpl{
		analyticsService: service,
	}
}

// GetSiteProductivity handles GET /sites/{siteID}/productivity
func (s *AnalyticsServerImpl) GetSiteProductivity(c *gin.Context, siteID string, params apiv2.GetSiteProductivityParams) {
	// Parse optional date range (start/end)
	startDate, endDate, err := parseDateRange(params.Start, params.End)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiv2.Error{Error: err.Error()})
		return
	}
	if startDate.IsZero() || endDate.IsZero() {
		c.JSON(http.StatusBadRequest, apiv2.Error{Error: "start and end dates are required"})
		return
	}

	// Call analytics service
	result, err := s.analyticsService.GetSiteProductivity(c.Request.Context(), siteID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiv2.Error{Error: err.Error()})
		return
	}

	// Convert to openapi model
	resp := convertSiteProductivityMetrics(result)
	c.JSON(http.StatusOK, resp)
}

// GetSiteTaskDistribution handles GET /sites/{siteID}/tasks
func (s *AnalyticsServerImpl) GetSiteTaskDistribution(c *gin.Context, siteID string, params apiv2.GetSiteTaskDistributionParams) {
	startDate, endDate, err := parseDateRange(params.Start, params.End)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiv2.Error{Error: err.Error()})
		return
	}
	if startDate.IsZero() || endDate.IsZero() {
		c.JSON(http.StatusBadRequest, apiv2.Error{Error: "start and end dates are required"})
		return
	}

	result, err := s.analyticsService.GetSiteTaskDistribution(c.Request.Context(), siteID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiv2.Error{Error: err.Error()})
		return
	}

	resp := convertSiteTaskDistribution(result)
	c.JSON(http.StatusOK, resp)
}

// GetSiteUtilization handles GET /sites/{siteID}/utilization
func (s *AnalyticsServerImpl) GetSiteUtilization(c *gin.Context, siteID string, params apiv2.GetSiteUtilizationParams) {
	date := time.Now() // default if no date
	if params.Date != nil {
		parsed, err := parseOptionalDate(*params.Date)
		if err != nil {
			c.JSON(http.StatusBadRequest, apiv2.Error{Error: "invalid date format"})
			return
		}
		date = parsed
	}

	result, err := s.analyticsService.GetSiteUtilization(c.Request.Context(), siteID, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiv2.Error{Error: err.Error()})
		return
	}

	resp := convertSiteUtilizationMetrics(result)
	c.JSON(http.StatusOK, resp)
}

// GetWorkerTaskHistory handles GET /workers/{workerID}/history
func (s *AnalyticsServerImpl) GetWorkerTaskHistory(c *gin.Context, workerID string, params apiv2.GetWorkerTaskHistoryParams) {
	startDate, endDate, err := parseDateRange(params.Start, params.End)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiv2.Error{Error: err.Error()})
		return
	}
	if startDate.IsZero() || endDate.IsZero() {
		c.JSON(http.StatusBadRequest, apiv2.Error{Error: "start and end dates are required"})
		return
	}

	result, err := s.analyticsService.GetWorkerTaskHistory(c.Request.Context(), workerID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiv2.Error{Error: err.Error()})
		return
	}

	resp := convertWorkerTaskHistory(result)
	c.JSON(http.StatusOK, resp)
}

// GetWorkerProductivity handles GET /workers/{workerID}/productivity
func (s *AnalyticsServerImpl) GetWorkerProductivity(c *gin.Context, workerID string, params apiv2.GetWorkerProductivityParams) {
	startDate, endDate, err := parseDateRange(params.Start, params.End)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiv2.Error{Error: err.Error()})
		return
	}
	if startDate.IsZero() || endDate.IsZero() {
		c.JSON(http.StatusBadRequest, apiv2.Error{Error: "start and end dates are required"})
		return
	}

	result, err := s.analyticsService.GetWorkerProductivity(c.Request.Context(), workerID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiv2.Error{Error: err.Error()})
		return
	}

	resp := convertWorkerProductivityMetrics(result)
	c.JSON(http.StatusOK, resp)
}

// GetWorkerUtilization handles GET /workers/{workerID}/utilization
func (s *AnalyticsServerImpl) GetWorkerUtilization(c *gin.Context, workerID string, params apiv2.GetWorkerUtilizationParams) {
	date := time.Now()
	if params.Date != nil {
		parsed, err := parseOptionalDate(*params.Date)
		if err != nil {
			c.JSON(http.StatusBadRequest, apiv2.Error{Error: "invalid date format"})
			return
		}
		date = parsed
	}

	result, err := s.analyticsService.GetWorkerUtilization(c.Request.Context(), workerID, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiv2.Error{Error: err.Error()})
		return
	}

	resp := convertWorkerUtilizationMetrics(result)
	c.JSON(http.StatusOK, resp)
}

// GetWorkerCurrentTasks handles GET /workflow/{workflowID}/tasks
func (s *AnalyticsServerImpl) GetWorkerCurrentTasks(c *gin.Context, workflowID string) {
	// In analytics_api.go, the param is "workflowID" but we interpret it as the Worker’s workflow ID
	tasks, err := s.analyticsService.GetWorkerCurrentTasks(c.Request.Context(), workflowID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiv2.Error{Error: err.Error()})
		return
	}

	resp := convertWorkerTaskStatus(tasks)
	c.JSON(http.StatusOK, resp)
}

// GetTaskBlockageStatus handles GET /workflow/{workflowID}/tasks/{taskID}/blockage
func (s *AnalyticsServerImpl) GetTaskBlockageStatus(c *gin.Context, workflowID string, taskID string) {
	blockage, err := s.analyticsService.GetTaskBlockageStatus(c.Request.Context(), taskID, workflowID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiv2.Error{Error: err.Error()})
		return
	}

	resp := apiv2.TaskBlockageInfo{
		IsBlocked:   blockage.IsBlocked,
		BlockReason: &blockage.BlockReason,
		Since:       blockage.Since,
	}
	c.JSON(http.StatusOK, resp)
}

// --------------------------------------------------------------------------------------------
// Helper Functions for converting between domain (analytics.*) and openapi (apiv2.*) models
// --------------------------------------------------------------------------------------------

func parseDateRange(startParam, endParam *openapi_types.Date) (time.Time, time.Time, error) {
	var start, end time.Time
	var err error

	if startParam != nil {
		start, err = parseOptionalDate(*startParam)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid start date")
		}
	}
	if endParam != nil {
		end, err = parseOptionalDate(*endParam)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid end date")
		}
	}
	return start, end, nil
}

func parseOptionalDate(d openapi_types.Date) (time.Time, error) {
	// openapi_types.Date is typically YYYY-MM-DD => parse.
	return d.Time, nil
}

func convertSiteProductivityMetrics(src *analytics.SiteProductivityMetrics) apiv2.SiteProductivityMetrics {
	if src == nil {
		return apiv2.SiteProductivityMetrics{}
	}
	dailyStats := make(map[string]apiv2.SiteProductivityStats)
	for day, stats := range src.DailyStats {
		key := day.Format("2006-01-02")
		dailyStats[key] = apiv2.SiteProductivityStats{
			ActiveWorkers:     &stats.ActiveWorkers,
			TotalTasks:        &stats.TotalTasks,
			CompletedTasks:    &stats.CompletedTasks,
			BlockedTasks:      &stats.BlockedTasks,
			AvgCompletionTime: stringPtr(stats.AvgCompletionTime.String()),
			CompletionRate:    float32Ptr(float32(stats.CompletionRate)),
		}
	}

	return apiv2.SiteProductivityMetrics{
		JobSite:    &src.JobSite,
		DailyStats: &dailyStats,
	}
}

func convertSiteTaskDistribution(src *analytics.SiteTaskDistribution) apiv2.SiteTaskDistribution {
	if src == nil {
		return apiv2.SiteTaskDistribution{}
	}

	// The openapi model has WorkerStats as map[string]interface{} by default,
	// but we can use a more structured approach if desired. We'll keep it simple:
	workerStats := make(map[string]interface{})
	for workerID, stats := range src.WorkerStats {
		workerStats[workerID] = map[string]interface{}{
			"totalTasks":      stats.TotalTasks,
			"completedTasks":  stats.CompletedTasks,
			"blockedTasks":    stats.BlockedTasks,
			"avgTaskDuration": stats.AvgTaskDuration.String(),
			"completionRate":  stats.CompletionRate,
		}
	}

	return apiv2.SiteTaskDistribution{
		JobSite:     &src.JobSite,
		WorkerStats: &workerStats,
	}
}

func convertSiteUtilizationMetrics(src *analytics.SiteUtilizationMetrics) apiv2.SiteUtilizationMetrics {
	if src == nil {
		return apiv2.SiteUtilizationMetrics{}
	}

	return apiv2.SiteUtilizationMetrics{
		JobSite: &src.JobSite,
		Stats: &apiv2.SiteUtilizationStats{
			TotalWorkers:        intPtr(src.Stats.TotalWorkers),
			OnTimeWorkers:       intPtr(src.Stats.OnTimeWorkers),
			LateWorkers:         intPtr(src.Stats.LateWorkers),
			AvgWorkDuration:     stringPtr(src.Stats.AvgWorkDuration.String()),
			SiteUtilizationRate: float32Ptr(float32(src.Stats.SiteUtilizationRate)),
		},
	}
}

func convertWorkerTaskHistory(src *analytics.WorkerTaskHistory) apiv2.WorkerTaskHistory {
	if src == nil {
		return apiv2.WorkerTaskHistory{}
	}

	var tasks []apiv2.HistoricalTaskInfo
	for _, t := range src.Tasks {
		tasks = append(tasks, apiv2.HistoricalTaskInfo{
			Id:               stringPtr(t.ID),
			Name:             stringPtr(t.Name),
			Status:           stringPtr(t.Status),
			PlannedStartTime: &t.PlannedStartTime,
			PlannedEndTime:   &t.PlannedEndTime,
		})
	}

	return apiv2.WorkerTaskHistory{
		WorkerId: &src.WorkerID,
		Tasks:    &tasks,
	}
}

func convertWorkerProductivityMetrics(src *analytics.WorkerProductivityMetrics) apiv2.WorkerProductivityMetrics {
	if src == nil {
		return apiv2.WorkerProductivityMetrics{}
	}

	dailyStats := make(map[string]apiv2.ProductivityStats)
	for day, stats := range src.DailyStats {
		key := day.Format("2006-01-02")
		dailyStats[key] = apiv2.ProductivityStats{
			TotalTasks:        intPtr(stats.TotalTasks),
			CompletedTasks:    intPtr(stats.CompletedTasks),
			BlockedTasks:      intPtr(stats.BlockedTasks),
			AvgCompletionTime: stringPtr(stats.AvgCompletionTime.String()),
			CompletionRate:    float32Ptr(float32(stats.CompletionRate)),
		}
	}

	return apiv2.WorkerProductivityMetrics{
		WorkerId:   &src.WorkerID,
		DailyStats: &dailyStats,
	}
}

func convertWorkerUtilizationMetrics(src *analytics.WorkerUtilizationMetrics) apiv2.WorkerUtilizationMetrics {
	if src == nil {
		return apiv2.WorkerUtilizationMetrics{}
	}

	return apiv2.WorkerUtilizationMetrics{
		WorkerId: stringPtr(src.WorkerID),
		Stats: &apiv2.UtilizationStats{
			TotalTime:       stringPtr(src.Stats.TotalTime.String()),
			TotalTasks:      intPtr(src.Stats.TotalTasks),
			CompletedTasks:  intPtr(src.Stats.CompletedTasks),
			ActiveTime:      stringPtr(src.Stats.ActiveTime.String()),
			UtilizationRate: float32Ptr(float32(src.Stats.UtilizationRate)),
		},
	}
}

func convertWorkerTaskStatus(src *analytics.WorkerTaskStatus) apiv2.WorkerTaskStatus {
	if src == nil {
		return apiv2.WorkerTaskStatus{}
	}

	var tasks []apiv2.TaskInfo
	for _, t := range src.CurrentTasks {
		ti := apiv2.TaskInfo{
			TaskId:      stringPtr(t.TaskID),
			Status:      stringPtr(string(t.Status)),
			IsBlocked:   &t.IsBlocked,
			BlockReason: &t.BlockReason,
			StartedAt:   &t.StartedAt,
		}
		tasks = append(tasks, ti)
	}

	return apiv2.WorkerTaskStatus{
		IsSessionActive: src.IsSessionActive,
		IsOnBreak:       src.IsOnBreak,
		CurrentSite:     src.CurrentSite,
		CurrentTasks:    tasks,
	}
}

// Utility conversions
func intPtr(i int) *int {
	return &i
}
func float32Ptr(f float32) *float32 {
	return &f
}
func stringPtr(s string) *string {
	return &s
}
