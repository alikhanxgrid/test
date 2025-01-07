package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"reference-app-wms-go/app/analytics"
	"reference-app-wms-go/app/dwr/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type checkInResponse struct {
	WorkflowID string `json:"workflowID"`
	RunID      string `json:"runID"`
}

func TestWorkerMonitoring(t *testing.T) {
	// Generate a proper UUID for the worker
	workerID := model.WorkerID(uuid.New().String())

	// Set up test data
	t.Log("Setting up test data...")
	jobSiteReq := map[string]string{
		"name":     "Site B",
		"location": "456 Test St",
	}
	resp, err := makeRequest("POST", "/scheduling/job-sites", jobSiteReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var jobSite struct {
		ID string `json:"id"`
	}
	err = json.NewDecoder(resp.Body).Decode(&jobSite)
	require.NoError(t, err)
	resp.Body.Close()

	// Create a schedule
	now := time.Now()
	scheduleReq := map[string]string{
		"startDate": now.Format(time.RFC3339),
		"endDate":   now.Add(24 * time.Hour).Format(time.RFC3339),
	}
	resp, err = makeRequest("POST", fmt.Sprintf("/scheduling/job-sites/%s/schedules", jobSite.ID), scheduleReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var schedule struct {
		ID string `json:"id"`
	}
	err = json.NewDecoder(resp.Body).Decode(&schedule)
	require.NoError(t, err)
	resp.Body.Close()

	// Create a task
	taskReq := map[string]interface{}{
		"name":               "Monitoring Test Task",
		"description":        "Task for monitoring test",
		"worker_id":          string(workerID),
		"schedule_id":        schedule.ID,
		"planned_start_time": now.Format(time.RFC3339),
		"planned_end_time":   now.Add(2 * time.Hour).Format(time.RFC3339),
	}
	resp, err = makeRequest("POST", fmt.Sprintf("/scheduling/job-sites/%s/tasks", jobSite.ID), taskReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var task struct {
		ID string `json:"id"`
	}
	err = json.NewDecoder(resp.Body).Decode(&task)
	require.NoError(t, err)
	resp.Body.Close()

	// 1. Check in the worker and get workflow ID
	t.Log("Checking in worker...")
	checkInReq := model.CheckInRequest{
		WorkerID:    workerID,
		CheckInTime: time.Now(),
		JobSiteID:   jobSite.ID,
	}

	// First check-in should succeed
	resp, err = makeRequest("POST", "/check-in", checkInReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var checkInResp checkInResponse
	err = json.NewDecoder(resp.Body).Decode(&checkInResp)
	require.NoError(t, err)
	require.NotEmpty(t, checkInResp.WorkflowID)
	resp.Body.Close()

	// Attempt duplicate check-in
	t.Log("Attempting duplicate check-in...")
	resp, err = makeRequest("POST", "/check-in", checkInReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusConflict, resp.StatusCode)

	var errorResp struct {
		Error string `json:"error"`
	}
	err = json.NewDecoder(resp.Body).Decode(&errorResp)
	require.NoError(t, err)
	require.Contains(t, errorResp.Error, "already checked in")
	resp.Body.Close()

	workflowID := checkInResp.WorkflowID

	// After check-in
	time.Sleep(2 * time.Second) // Increased sleep after check-in

	// 2. Start a task
	t.Log("Starting a task...")
	taskUpdate := model.TaskUpdate{
		TaskID:     task.ID,
		NewStatus:  model.TaskStatusInProgress,
		UpdateTime: time.Now(),
		UpdatedBy:  workerID,
		Notes:      "Starting monitoring test task",
	}

	resp, err = makeRequest("POST", "/task-progress", taskUpdate)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	time.Sleep(2 * time.Second) // Added sleep after task update

	// 3. Query current tasks
	t.Log("Querying worker's current tasks...")
	resp, err = makeRequest("GET", fmt.Sprintf("/analytics/workflow/%s/tasks", workflowID), nil)
	require.NoError(t, err)

	if resp.StatusCode != http.StatusOK {
		// Read and log the error response
		var errorResponse map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&errorResponse)
		if err != nil {
			t.Logf("Failed to decode error response: %v", err)
		} else {
			t.Logf("Error response: %+v", errorResponse)
		}
		resp.Body.Close()
	}

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var taskStatus analytics.WorkerTaskStatus
	err = json.NewDecoder(resp.Body).Decode(&taskStatus)
	require.NoError(t, err)
	require.True(t, taskStatus.IsSessionActive, "Worker should be checked in")
	require.Equal(t, jobSite.ID, taskStatus.CurrentSite)
	require.NotEmpty(t, taskStatus.CurrentTasks, "Should have at least one task")
	resp.Body.Close()

	// 4. Mark task as blocked
	t.Log("Marking task as blocked...")
	taskUpdate = model.TaskUpdate{
		TaskID:     task.ID,
		NewStatus:  model.TaskStatusBlocked,
		UpdateTime: time.Now(),
		UpdatedBy:  workerID,
		Notes:      "Task blocked due to missing materials",
	}

	resp, err = makeRequest("POST", "/task-progress", taskUpdate)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	time.Sleep(2 * time.Second) // Added sleep after blocking task

	// 5. Query task blockage status
	t.Log("Querying task blockage status...")
	resp, err = makeRequest("GET", fmt.Sprintf("/analytics/workflow/%s/tasks/%s/blockage", workflowID, task.ID), nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var blockageInfo analytics.TaskBlockageInfo
	err = json.NewDecoder(resp.Body).Decode(&blockageInfo)
	require.NoError(t, err)
	require.True(t, blockageInfo.IsBlocked, "Task should be marked as blocked")
	require.NotEmpty(t, blockageInfo.BlockReason, "Block reason should be provided")
	resp.Body.Close()

	// 6. Complete the task
	t.Log("Completing the task...")
	taskUpdate = model.TaskUpdate{
		TaskID:     task.ID,
		NewStatus:  model.TaskStatusCompleted,
		UpdateTime: time.Now(),
		UpdatedBy:  workerID,
		Notes:      "Task completed after resolving blockage",
	}

	resp, err = makeRequest("POST", "/task-progress", taskUpdate)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// TODO(Abdullah): Lazy ! sleep is bad, should have a wrapper to wait for the desired state or timeout
	time.Sleep(2 * time.Second) // Add sleep after completing task

	// 7. Verify task completion through query
	t.Log("Verifying task completion...")
	resp, err = makeRequest("GET", fmt.Sprintf("/analytics/workflow/%s/tasks", workflowID), nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	err = json.NewDecoder(resp.Body).Decode(&taskStatus)
	require.NoError(t, err)
	resp.Body.Close()

	// 8. Check out the worker
	t.Log("Checking out worker...")
	checkOutReq := model.CheckOutRequest{
		WorkerID:     workerID,
		CheckOutTime: time.Now(),
		JobSiteID:    jobSite.ID,
		Notes:        "Completed monitoring test",
	}

	resp, err = makeRequest("POST", "/check-out", checkOutReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// 9. Verify checked out status
	t.Log("Verifying checked out status...")
	resp, err = makeRequest("GET", fmt.Sprintf("/analytics/workflow/%s/tasks", workflowID), nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	err = json.NewDecoder(resp.Body).Decode(&taskStatus)
	require.NoError(t, err)
	require.False(t, taskStatus.IsSessionActive, "Worker should be checked out")
	resp.Body.Close()

	t.Log("Worker monitoring test completed successfully")
}
