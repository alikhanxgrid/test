package analytics

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// AnalyticsAPI handles HTTP requests for analytics operations
type AnalyticsAPI struct {
	analyticsService AnalyticsService
	logger           *log.Logger
}

// NewAnalyticsAPI creates a new AnalyticsAPI instance
func NewAnalyticsAPI(analyticsService AnalyticsService, logger *log.Logger) *AnalyticsAPI {
	return &AnalyticsAPI{
		analyticsService: analyticsService,
		logger:           logger,
	}
}

// SetupRoutes registers all the analytics API routes
func (api *AnalyticsAPI) SetupRoutes(router *gin.Engine) {
	analytics := router.Group("/analytics")
	{
		// Real-time monitoring endpoints (using workflowID)
		workflow := analytics.Group("/workflow")
		{
			workflow.GET("/:workflowID/tasks", api.GetWorkerCurrentTasks)
			workflow.GET("/:workflowID/tasks/:taskID/blockage", api.GetTaskBlockageStatus)
		}

		// Historical analytics endpoints (using workerID)
		workers := analytics.Group("/workers")
		{
			workers.GET("/:workerID/productivity", api.GetWorkerProductivity)
			workers.GET("/:workerID/history", api.GetWorkerTaskHistory)
			workers.GET("/:workerID/utilization", api.GetWorkerUtilization)
		}

		// Site analytics endpoints
		sites := analytics.Group("/sites")
		{
			sites.GET("/:siteID/productivity", api.GetSiteProductivity)
			sites.GET("/:siteID/utilization", api.GetSiteUtilization)
			sites.GET("/:siteID/tasks", api.GetSiteTaskDistribution)
		}
	}
}

// Real-time monitoring handlers

func (api *AnalyticsAPI) GetWorkerCurrentTasks(c *gin.Context) {
	workflowID := c.Param("workflowID")
	if workflowID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workflowID is required"})
		return
	}

	tasks, err := api.analyticsService.GetWorkerCurrentTasks(c.Request.Context(), workflowID)
	if err != nil {
		api.logger.Printf("Failed to get worker tasks: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

func (api *AnalyticsAPI) GetTaskBlockageStatus(c *gin.Context) {
	workflowID := c.Param("workflowID")
	taskID := c.Param("taskID")

	if workflowID == "" || taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workflowID and taskID are required"})
		return
	}

	blockageInfo, err := api.analyticsService.GetTaskBlockageStatus(c.Request.Context(), taskID, workflowID)
	if err != nil {
		api.logger.Printf("Failed to get task blockage status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, blockageInfo)
}

// Task analytics handlers

func (api *AnalyticsAPI) GetTasksByStatus(c *gin.Context) {
	jobSiteID := c.Query("siteId")
	dateStr := c.Query("date")

	if jobSiteID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "siteId is required"})
		return
	}

	date := time.Now()
	if dateStr != "" {
		var err error
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format"})
			return
		}
	}

	distribution, err := api.analyticsService.GetTasksByStatus(c.Request.Context(), jobSiteID, date)
	if err != nil {
		api.logger.Printf("Failed to get tasks by status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, distribution)
}

func (api *AnalyticsAPI) GetTaskTrends(c *gin.Context) {
	jobSiteID := c.Query("siteId")
	startStr := c.Query("start")
	endStr := c.Query("end")

	if jobSiteID == "" || startStr == "" || endStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "siteId, start, and end dates are required"})
		return
	}

	startDate, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start date format"})
		return
	}

	endDate, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end date format"})
		return
	}

	trends, err := api.analyticsService.GetTaskTrends(c.Request.Context(), jobSiteID, startDate, endDate)
	if err != nil {
		api.logger.Printf("Failed to get task trends: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, trends)
}

func (api *AnalyticsAPI) GetBlockageTrends(c *gin.Context) {
	jobSiteID := c.Query("siteId")
	startStr := c.Query("start")
	endStr := c.Query("end")

	if jobSiteID == "" || startStr == "" || endStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "siteId, start, and end dates are required"})
		return
	}

	startDate, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start date format"})
		return
	}

	endDate, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end date format"})
		return
	}

	trends, err := api.analyticsService.GetBlockageTrends(c.Request.Context(), jobSiteID, startDate, endDate)
	if err != nil {
		api.logger.Printf("Failed to get blockage trends: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, trends)
}

// Worker analytics handlers

func (api *AnalyticsAPI) GetWorkerProductivity(c *gin.Context) {
	workerID := c.Param("workerID")
	startStr := c.Query("start")
	endStr := c.Query("end")

	if workerID == "" || startStr == "" || endStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workerID, start, and end dates are required"})
		return
	}

	startDate, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start date format"})
		return
	}

	endDate, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end date format"})
		return
	}

	metrics, err := api.analyticsService.GetWorkerProductivity(c.Request.Context(), workerID, startDate, endDate)
	if err != nil {
		api.logger.Printf("Failed to get worker productivity: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (api *AnalyticsAPI) GetWorkerTaskHistory(c *gin.Context) {
	workerID := c.Param("workerID")
	startStr := c.Query("start")
	endStr := c.Query("end")

	if workerID == "" || startStr == "" || endStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workerID, start, and end dates are required"})
		return
	}

	startDate, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start date format"})
		return
	}

	endDate, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end date format"})
		return
	}

	history, err := api.analyticsService.GetWorkerTaskHistory(c.Request.Context(), workerID, startDate, endDate)
	if err != nil {
		api.logger.Printf("Failed to get worker task history: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, history)
}

func (api *AnalyticsAPI) GetWorkerUtilization(c *gin.Context) {
	workerID := c.Param("workerID")
	dateStr := c.Query("date")

	if workerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workerID is required"})
		return
	}

	date := time.Now()
	if dateStr != "" {
		var err error
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format"})
			return
		}
	}

	metrics, err := api.analyticsService.GetWorkerUtilization(c.Request.Context(), workerID, date)
	if err != nil {
		api.logger.Printf("Failed to get worker utilization: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// Site analytics handlers

func (api *AnalyticsAPI) GetSiteProductivity(c *gin.Context) {
	siteID := c.Param("siteID")
	startStr := c.Query("start")
	endStr := c.Query("end")

	if siteID == "" || startStr == "" || endStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "siteID, start, and end dates are required"})
		return
	}

	startDate, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start date format"})
		return
	}

	endDate, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end date format"})
		return
	}

	metrics, err := api.analyticsService.GetSiteProductivity(c.Request.Context(), siteID, startDate, endDate)
	if err != nil {
		api.logger.Printf("Failed to get site productivity: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (api *AnalyticsAPI) GetSiteUtilization(c *gin.Context) {
	siteID := c.Param("siteID")
	dateStr := c.Query("date")

	if siteID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "siteID is required"})
		return
	}

	date := time.Now()
	if dateStr != "" {
		var err error
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format"})
			return
		}
	}

	metrics, err := api.analyticsService.GetSiteUtilization(c.Request.Context(), siteID, date)
	if err != nil {
		api.logger.Printf("Failed to get site utilization: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (api *AnalyticsAPI) GetSiteTaskDistribution(c *gin.Context) {
	siteID := c.Param("siteID")
	startStr := c.Query("start")
	endStr := c.Query("end")

	if siteID == "" || startStr == "" || endStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "siteID, start, and end dates are required"})
		return
	}

	startDate, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start date format"})
		return
	}

	endDate, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end date format"})
		return
	}

	distribution, err := api.analyticsService.GetSiteTaskDistribution(c.Request.Context(), siteID, startDate, endDate)
	if err != nil {
		api.logger.Printf("Failed to get site task distribution: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, distribution)
}
