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

	response, err := s.temporalClient.QueryWorkflow(c, workflowID, "", "getCurrentTasks")
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

	response, err := s.temporalClient.QueryWorkflow(c, workerID, "", "getTaskBlockage", taskID)
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
		c.JSON(http.StatusBadRequest, apiv2.Error{Error: "siteID is required"})
		return
	}

	s.logger.Printf("[GetSiteProductivity] Querying site %s for productivity metrics", siteID)
}
