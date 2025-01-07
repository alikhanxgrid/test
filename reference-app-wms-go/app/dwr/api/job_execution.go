package api

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"

	"reference-app-wms-go/app/dwr/model"
	"reference-app-wms-go/app/dwr/workflows"
)

const (
	TemporalTaskQueue = "DWR-TASK-QUEUE"
)

type JobExecutionAPI struct {
	temporalClient client.Client
	logger         *log.Logger
}

func NewJobExecutionAPI(temporalClient client.Client) *JobExecutionAPI {
	return &JobExecutionAPI{
		temporalClient: temporalClient,
		logger:         log.New(os.Stdout, "[JobExecutionAPI] ", log.LstdFlags),
	}
}

func (api *JobExecutionAPI) SetupRoutes(router *gin.Engine) {
	router.POST("/check-in", api.HandleCheckIn)
	router.POST("/check-out", api.HandleCheckOut)
	router.POST("/task-progress", api.HandleTaskProgress)
	router.POST("/break", api.HandleBreak)
}

func (api *JobExecutionAPI) HandleCheckIn(c *gin.Context) {
	var req model.CheckInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	workflowID := model.FormatWorkflowID(req.WorkerID)

	// Check if worker is already checked in
	workflow, err := api.temporalClient.DescribeWorkflowExecution(c, workflowID, "")
	if err == nil && workflow.WorkflowExecutionInfo.Status == enums.WORKFLOW_EXECUTION_STATUS_RUNNING {
		api.logger.Printf("Worker %s is already checked in. Please check out first.", req.WorkerID)
		c.JSON(http.StatusConflict, gin.H{
			"error": fmt.Sprintf("Worker %s is already checked in. Please check out first.", req.WorkerID),
		})
		return
	}

	workflowOptions := client.StartWorkflowOptions{
		ID:                    workflowID,
		TaskQueue:             TemporalTaskQueue,
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
	}

	// Start workflow
	params := workflows.DailyWorkerRoutineParams{
		WorkerID:    req.WorkerID,
		CheckInTime: req.CheckInTime,
		JobSiteID:   req.JobSiteID,
	}

	we, err := api.temporalClient.ExecuteWorkflow(c, workflowOptions, workflows.DailyWorkerRoutine, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start workflow"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"workflowID": workflowID,
		"runID":      we.GetRunID(),
	})
}

func (api *JobExecutionAPI) HandleCheckOut(c *gin.Context) {
	var req model.CheckOutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.logger.Printf("Failed to parse check-out request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	api.logger.Printf("Received check-out request - Worker: %s, CheckOutTime: %v, JobSiteId: %s",
		req.WorkerID, req.CheckOutTime, req.JobSiteID)

	workflowID := model.FormatWorkflowID(req.WorkerID)
	api.logger.Printf("Sending check-out signal to workflow: %s", workflowID)

	// Verify workflow exists and is running
	workflow, err := api.temporalClient.DescribeWorkflowExecution(c, workflowID, "")
	if err != nil {
		api.logger.Printf("Failed to describe workflow: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to verify workflow: %v", err)})
		return
	}
	api.logger.Printf("Workflow status: %s", workflow.WorkflowExecutionInfo.Status)

	err = api.temporalClient.SignalWorkflow(c, workflowID, "", "check-out", req)
	if err != nil {
		api.logger.Printf("Failed to signal workflow: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to signal workflow: %v", err)})
		return
	}

	api.logger.Printf("Successfully sent check-out signal to workflow")
	c.JSON(http.StatusOK, gin.H{"status": "check-out signal sent"})
}

func (api *JobExecutionAPI) HandleTaskProgress(c *gin.Context) {
	var update model.TaskUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		api.logger.Printf("Failed to parse task progress update: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	api.logger.Printf("Received task progress update - TaskID: %s, Status: %s, Worker: %s",
		update.TaskID, update.NewStatus, update.UpdatedBy)

	workflowID := model.FormatWorkflowID(update.UpdatedBy)
	api.logger.Printf("Sending signal to workflow: %s", workflowID)

	// Verify workflow exists and is running
	workflow, err := api.temporalClient.DescribeWorkflowExecution(c, workflowID, "")
	if err != nil {
		api.logger.Printf("Failed to describe workflow: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to verify workflow: %v", err)})
		return
	}
	api.logger.Printf("Workflow status: %s", workflow.WorkflowExecutionInfo.Status)

	err = api.temporalClient.SignalWorkflow(c, workflowID, "", "task-progress-signal", update)
	if err != nil {
		api.logger.Printf("Failed to signal workflow: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to signal workflow: %v", err)})
		return
	}

	api.logger.Printf("Successfully sent task progress signal to workflow")
	c.JSON(http.StatusOK, gin.H{"status": "update signal sent"})
}

func (api *JobExecutionAPI) HandleBreak(c *gin.Context) {
	var req model.BreakSignal
	if err := c.ShouldBindJSON(&req); err != nil {
		api.logger.Printf("Failed to parse break request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	api.logger.Printf("Received break request - Worker: %s, IsOnBreak: %v, StartTime: %v",
		req.WorkerID, req.IsOnBreak, req.StartTime)

	workflowID := model.FormatWorkflowID(req.WorkerID)
	api.logger.Printf("Sending break signal to workflow: %s", workflowID)

	// Verify workflow exists and is running
	workflow, err := api.temporalClient.DescribeWorkflowExecution(c, workflowID, "")
	if err != nil {
		api.logger.Printf("Failed to describe workflow: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to verify workflow: %v", err)})
		return
	}
	api.logger.Printf("Workflow status: %s", workflow.WorkflowExecutionInfo.Status)

	err = api.temporalClient.SignalWorkflow(c, workflowID, "", "break-signal", req)
	if err != nil {
		api.logger.Printf("Failed to signal workflow: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to signal workflow: %v", err)})
		return
	}

	api.logger.Printf("Successfully sent break signal to workflow")
	c.JSON(http.StatusOK, gin.H{"status": "break signal sent"})
}
