package impl

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"

	apiv2 "reference-app-wms-go/app/dwr/api/v2/openapi"
	"reference-app-wms-go/app/dwr/model"
	"reference-app-wms-go/app/dwr/workflows"
)

const (
	TemporalTaskQueue = "DWR-TASK-QUEUE"
)

// JobExecutionAPIV2 implements the generated ServerInterface
type JobExecutionAPIV2 struct {
	temporalClient client.Client
	logger         *log.Logger
}

// NewJobExecutionAPIV2 creates a new instance of the v2 API
func NewJobExecutionAPIV2(temporalClient client.Client) *JobExecutionAPIV2 {
	return &JobExecutionAPIV2{
		temporalClient: temporalClient,
		logger:         log.New(os.Stdout, "[JobExecutionAPIV2] ", log.LstdFlags),
	}
}

// formatWorkflowID formats a workflow ID consistently
func formatWorkflowID(workerID apiv2.WorkerID) string {
	return fmt.Sprintf("DWR-%s", workerID)
}

// WorkerCheckIn implements the check-in endpoint
func (api *JobExecutionAPIV2) WorkerCheckIn(c *gin.Context) {
	var req apiv2.CheckInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiv2.Error{Error: err.Error()})
		return
	}

	workflowID := formatWorkflowID(req.WorkerId)

	// Check if worker is already checked in
	workflow, err := api.temporalClient.DescribeWorkflowExecution(c, workflowID, "")
	if err == nil && workflow.WorkflowExecutionInfo.Status == enums.WORKFLOW_EXECUTION_STATUS_RUNNING {
		api.logger.Printf("Worker %s is already checked in. Please check out first.", req.WorkerId)
		c.JSON(http.StatusConflict, apiv2.Error{
			Error: fmt.Sprintf("Worker %s is already checked in. Please check out first.", req.WorkerId),
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
		WorkerID:    model.WorkerID(req.WorkerId),
		CheckInTime: req.CheckInTime,
		JobSiteID:   req.JobSiteId,
	}

	we, err := api.temporalClient.ExecuteWorkflow(c, workflowOptions, workflows.DailyWorkerRoutine, params)
	if err != nil {
		api.logger.Printf("Failed to start workflow: %v", err)
		c.JSON(http.StatusInternalServerError, apiv2.Error{Error: "Failed to start workflow"})
		return
	}

	c.JSON(http.StatusOK, apiv2.CheckInResponse{
		WorkflowID: workflowID,
		RunID:      we.GetRunID(),
	})
}

// WorkerCheckOut implements the check-out endpoint
func (api *JobExecutionAPIV2) WorkerCheckOut(c *gin.Context) {
	var req apiv2.CheckOutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.logger.Printf("Failed to parse check-out request: %v", err)
		c.JSON(http.StatusBadRequest, apiv2.Error{Error: err.Error()})
		return
	}

	api.logger.Printf("Received check-out request - Worker: %s, CheckOutTime: %v, JobSiteId: %s",
		req.WorkerId, req.CheckOutTime, req.JobSiteId)

	workflowID := formatWorkflowID(req.WorkerId)
	api.logger.Printf("Sending check-out signal to workflow: %s", workflowID)

	// Convert to workflow params
	checkOutParams := workflows.CheckOutParams{
		WorkerID:     model.WorkerID(req.WorkerId),
		CheckOutTime: req.CheckOutTime,
		JobSiteID:    req.JobSiteId,
		Date:         req.Date,
	}
	if req.Notes != nil {
		checkOutParams.Notes = *req.Notes
	}

	// Verify workflow exists and is running
	workflow, err := api.temporalClient.DescribeWorkflowExecution(c, workflowID, "")
	if err != nil {
		api.logger.Printf("Failed to describe workflow: %v", err)
		c.JSON(http.StatusInternalServerError, apiv2.Error{Error: fmt.Sprintf("Failed to verify workflow: %v", err)})
		return
	}
	api.logger.Printf("Workflow status: %s", workflow.WorkflowExecutionInfo.Status)

	err = api.temporalClient.SignalWorkflow(c, workflowID, "", "check-out", checkOutParams)
	if err != nil {
		api.logger.Printf("Failed to signal workflow: %v", err)
		c.JSON(http.StatusInternalServerError, apiv2.Error{Error: fmt.Sprintf("Failed to signal workflow: %v", err)})
		return
	}

	api.logger.Printf("Successfully sent check-out signal to workflow")
	c.JSON(http.StatusOK, apiv2.GenericResponse{Status: "check-out signal sent"})
}

// UpdateTaskProgress implements the task progress endpoint
func (api *JobExecutionAPIV2) UpdateTaskProgress(c *gin.Context) {
	var update apiv2.TaskUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		api.logger.Printf("Failed to parse task progress update: %v", err)
		c.JSON(http.StatusBadRequest, apiv2.Error{Error: err.Error()})
		return
	}

	api.logger.Printf("Received task progress update - TaskID: %s, Status: %s, Worker: %s",
		update.TaskId, update.NewStatus, update.UpdatedBy)

	workflowID := formatWorkflowID(update.UpdatedBy)
	api.logger.Printf("Sending signal to workflow: %s", workflowID)

	// Convert to workflow params
	taskUpdateParams := workflows.TaskUpdateParams{
		TaskID:     update.TaskId,
		NewStatus:  model.TaskStatus(update.NewStatus),
		UpdateTime: update.UpdateTime,
		UpdatedBy:  model.WorkerID(update.UpdatedBy),
	}
	if update.Notes != nil {
		taskUpdateParams.Notes = *update.Notes
	}

	// Verify workflow exists and is running
	workflow, err := api.temporalClient.DescribeWorkflowExecution(c, workflowID, "")
	if err != nil {
		api.logger.Printf("Failed to describe workflow: %v", err)
		c.JSON(http.StatusInternalServerError, apiv2.Error{Error: fmt.Sprintf("Failed to verify workflow: %v", err)})
		return
	}
	api.logger.Printf("Workflow status: %s", workflow.WorkflowExecutionInfo.Status)

	err = api.temporalClient.SignalWorkflow(c, workflowID, "", "task-progress-signal", taskUpdateParams)
	if err != nil {
		api.logger.Printf("Failed to signal workflow: %v", err)
		c.JSON(http.StatusInternalServerError, apiv2.Error{Error: fmt.Sprintf("Failed to signal workflow: %v", err)})
		return
	}

	api.logger.Printf("Successfully sent task progress signal to workflow")
	c.JSON(http.StatusOK, apiv2.GenericResponse{Status: "update signal sent"})
}

// SignalBreak implements the break signal endpoint
func (api *JobExecutionAPIV2) SignalBreak(c *gin.Context) {
	var req apiv2.BreakSignal
	if err := c.ShouldBindJSON(&req); err != nil {
		api.logger.Printf("Failed to parse break request: %v", err)
		c.JSON(http.StatusBadRequest, apiv2.Error{Error: err.Error()})
		return
	}

	api.logger.Printf("Received break request - Worker: %s, IsOnBreak: %v, StartTime: %v",
		req.WorkerId, req.IsOnBreak, req.StartTime)

	workflowID := formatWorkflowID(req.WorkerId)
	api.logger.Printf("Sending break signal to workflow: %s", workflowID)

	// Convert to workflow params
	breakParams := workflows.BreakParams{
		IsOnBreak: req.IsOnBreak,
		StartTime: req.StartTime,
	}

	// Verify workflow exists and is running
	workflow, err := api.temporalClient.DescribeWorkflowExecution(c, workflowID, "")
	if err != nil {
		api.logger.Printf("Failed to describe workflow: %v", err)
		c.JSON(http.StatusInternalServerError, apiv2.Error{Error: fmt.Sprintf("Failed to verify workflow: %v", err)})
		return
	}
	api.logger.Printf("Workflow status: %s", workflow.WorkflowExecutionInfo.Status)

	err = api.temporalClient.SignalWorkflow(c, workflowID, "", "break-signal", breakParams)
	if err != nil {
		api.logger.Printf("Failed to signal workflow: %v", err)
		c.JSON(http.StatusInternalServerError, apiv2.Error{Error: fmt.Sprintf("Failed to signal workflow: %v", err)})
		return
	}

	api.logger.Printf("Successfully sent break signal to workflow")
	c.JSON(http.StatusOK, apiv2.GenericResponse{Status: "break signal sent"})
}
