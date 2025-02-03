package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"reference-app-wms-go/app/analytics"
	apiv2 "reference-app-wms-go/app/dwr/api/v2/openapi"
)

func TestTaskManagementAndWorkerState(t *testing.T) {
	// Setup test database
	testDB, dbCleanup := SetupTestDB(t)
	defer dbCleanup()

	var serverCleanup func()
	if !noServer {
		// Start test server
		_, serverCleanup = StartTestServer(t, testDB)
		defer func() {
			if serverCleanup != nil {
				serverCleanup()
			}
		}()
	} else {
		t.Log("Skipping server startup as -no-server flag is set")
	}

	// Create API client
	client := createJobExecutionClient(t)
	ctx := context.Background()

	// Set up test data
	workerID := GenerateWorkerID()
	jobSite := setupJobSite(t)
	schedule := setupSchedule(t, jobSite.ID)
	tasks := setupTasks(t, jobSite.ID, workerID, schedule.ID)

	// 1. Worker Check-in
	t.Log("Testing worker check-in...")
	checkInReq := apiv2.CheckInRequest{
		WorkerId:    apiv2.WorkerID(workerID),
		CheckInTime: time.Now(),
		JobSiteId:   jobSite.ID,
		Date:        time.Now(),
	}

	checkInResp, err := client.WorkerCheckInWithResponse(ctx, checkInReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, checkInResp.StatusCode())
	require.NotEmpty(t, checkInResp.JSON200.WorkflowID)

	// 2. Verify initial task state
	t.Log("Verifying initial task state...")
	var taskStatus analytics.WorkerTaskStatus
	RequireEventually(t, "initial task state verification", func() (bool, error) {
		resp, err := makeRequest("GET", fmt.Sprintf("/analytics/workflow/%s/tasks", checkInResp.JSON200.WorkflowID), nil)
		if err != nil {
			t.Logf("Error making request: %v", err)
			return false, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			t.Logf("Unexpected status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
			return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		if err := json.NewDecoder(resp.Body).Decode(&taskStatus); err != nil {
			t.Logf("Error decoding response: %v", err)
			return false, err
		}

		t.Logf("Received task status - Active: %v, Site: %s, Tasks count: %d",
			taskStatus.IsSessionActive, taskStatus.CurrentSite, len(taskStatus.CurrentTasks))

		return taskStatus.IsSessionActive &&
			taskStatus.CurrentSite == jobSite.ID &&
			len(taskStatus.CurrentTasks) == 2, nil
	})

	// Verify initial task statuses
	for _, task := range taskStatus.CurrentTasks {
		require.Equal(t, analytics.TaskStatusPending, task.Status, "Task should be in PENDING status initially")
	}

	// 3. Query current tasks
	t.Log("Querying worker's current tasks...")
	resp, err := makeRequest("GET", fmt.Sprintf("/analytics/workflow/%s/tasks", checkInResp.JSON200.WorkflowID), nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var currentStatus analytics.WorkerTaskStatus
	err = json.NewDecoder(resp.Body).Decode(&currentStatus)
	require.NoError(t, err)
	require.True(t, currentStatus.IsSessionActive, "Worker should be checked in")
	require.Equal(t, jobSite.ID, currentStatus.CurrentSite)
	require.NotEmpty(t, currentStatus.CurrentTasks, "Should have at least one task")
	resp.Body.Close()

	// 4. Start first task
	t.Log("Starting first task...")
	taskUpdate := apiv2.TaskUpdate{
		TaskId:     tasks[0],
		NewStatus:  apiv2.INPROGRESS,
		UpdateTime: time.Now(),
		UpdatedBy:  apiv2.WorkerID(workerID),
		Notes:      stringPtr("Starting first task"),
	}

	taskUpdateResp, err := client.UpdateTaskProgressWithResponse(ctx, taskUpdate)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, taskUpdateResp.StatusCode())

	// Verify task status after update
	t.Log("Verifying task status update...")
	var updatedTask *analytics.TaskInfo
	RequireEventually(t, "task status update verification", func() (bool, error) {
		resp, err := makeRequest("GET", fmt.Sprintf("/analytics/workflow/%s/tasks", checkInResp.JSON200.WorkflowID), nil)
		if err != nil {
			return false, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		var status analytics.WorkerTaskStatus
		if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
			return false, err
		}

		// Verify all tasks
		foundInProgress := false
		for _, task := range status.CurrentTasks {
			if task.TaskID == tasks[0] {
				updatedTask = &task
				foundInProgress = task.Status == analytics.TaskStatusInProgress
			} else {
				// Other tasks should still be pending
				if task.Status != analytics.TaskStatusPending {
					return false, fmt.Errorf("expected other tasks to be pending, got %s", task.Status)
				}
			}
		}
		return foundInProgress, nil
	})

	require.NotNil(t, updatedTask, "Updated task should be found")
	require.Equal(t, analytics.TaskStatusInProgress, updatedTask.Status, "Task should be IN_PROGRESS")

	// 5. Mark task as blocked
	t.Log("Marking task as blocked...")
	blockageUpdate := apiv2.TaskUpdate{
		TaskId:     tasks[0],
		NewStatus:  apiv2.BLOCKED,
		UpdateTime: time.Now(),
		UpdatedBy:  apiv2.WorkerID(workerID),
		Notes:      stringPtr("Missing materials"),
	}

	blockageUpdateResp, err := client.UpdateTaskProgressWithResponse(ctx, blockageUpdate)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, blockageUpdateResp.StatusCode())

	// Verify task blockage
	t.Log("Verifying task blockage...")
	var blockageInfo analytics.TaskBlockageInfo
	RequireEventually(t, "task blockage verification", func() (bool, error) {
		resp, err := makeRequest("GET", fmt.Sprintf("/analytics/workflow/%s/tasks/%s/blockage", checkInResp.JSON200.WorkflowID, tasks[0]), nil)
		if err != nil {
			return false, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		if err := json.NewDecoder(resp.Body).Decode(&blockageInfo); err != nil {
			return false, err
		}

		return blockageInfo.IsBlocked && blockageInfo.BlockReason == "Missing materials", nil
	})

	// 6. Take a break
	t.Log("Starting break...")
	breakStartTime := time.Now()
	breakSignal := apiv2.BreakSignal{
		WorkerId:  apiv2.WorkerID(workerID),
		IsOnBreak: true,
		StartTime: breakStartTime,
	}

	breakResp, err := client.SignalBreakWithResponse(ctx, breakSignal)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, breakResp.StatusCode())

	// Verify break status
	t.Log("Verifying break status...")
	RequireEventually(t, "break status verification", func() (bool, error) {
		resp, err := makeRequest("GET", fmt.Sprintf("/analytics/workflow/%s/tasks", checkInResp.JSON200.WorkflowID), nil)
		if err != nil {
			return false, err
		}
		defer resp.Body.Close()

		var status analytics.WorkerTaskStatus
		if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
			return false, err
		}

		return status.IsOnBreak, nil
	})

	// 7. End break
	t.Log("Ending break...")
	breakSignal.IsOnBreak = false
	breakSignal.StartTime = time.Now()

	breakEndResp, err := client.SignalBreakWithResponse(ctx, breakSignal)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, breakEndResp.StatusCode())

	// Verify break ended
	t.Log("Verifying break ended...")
	RequireEventually(t, "break end verification", func() (bool, error) {
		resp, err := makeRequest("GET", fmt.Sprintf("/analytics/workflow/%s/tasks", checkInResp.JSON200.WorkflowID), nil)
		if err != nil {
			return false, err
		}
		defer resp.Body.Close()

		var status analytics.WorkerTaskStatus
		if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
			return false, err
		}

		return !status.IsOnBreak, nil
	})

	// 8. Resume and complete task
	t.Log("Resuming and completing task...")
	// First resume the task
	resumeUpdate := apiv2.TaskUpdate{
		TaskId:     tasks[0],
		NewStatus:  apiv2.INPROGRESS,
		UpdateTime: time.Now(),
		UpdatedBy:  apiv2.WorkerID(workerID),
		Notes:      stringPtr("Materials available now"),
	}

	resumeResp, err := client.UpdateTaskProgressWithResponse(ctx, resumeUpdate)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resumeResp.StatusCode())

	// Then complete it
	completeUpdate := apiv2.TaskUpdate{
		TaskId:     tasks[0],
		NewStatus:  apiv2.COMPLETED,
		UpdateTime: time.Now(),
		UpdatedBy:  apiv2.WorkerID(workerID),
		Notes:      stringPtr("Task completed successfully"),
	}

	completeResp, err := client.UpdateTaskProgressWithResponse(ctx, completeUpdate)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, completeResp.StatusCode())

	// 9. Check-out
	t.Log("Testing worker check-out...")
	checkOutReq := apiv2.CheckOutRequest{
		WorkerId:     apiv2.WorkerID(workerID),
		CheckOutTime: time.Now(),
		JobSiteId:    jobSite.ID,
		Notes:        stringPtr("Completed all tasks for the day"),
		Date:         time.Now(),
	}

	checkOutResp, err := client.WorkerCheckOutWithResponse(ctx, checkOutReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, checkOutResp.StatusCode())

	t.Log("Task management test completed successfully")
}

// Helper functions for setting up test data
func setupJobSite(t *testing.T) *jobSiteResponse {
	createJobSiteReq := map[string]string{
		"name":     "Task Management Test Site",
		"location": "456 Test Ave",
	}

	resp, err := makeRequest("POST", "/scheduling/job-sites", createJobSiteReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var jobSite jobSiteResponse
	err = json.NewDecoder(resp.Body).Decode(&jobSite)
	require.NoError(t, err)
	require.NotEmpty(t, jobSite.ID)
	resp.Body.Close()

	return &jobSite
}

func setupSchedule(t *testing.T, jobSiteID string) *scheduleResponse {
	now := time.Now()
	createScheduleReq := map[string]string{
		"startDate": now.Format(time.RFC3339),
		"endDate":   now.Add(24 * time.Hour).Format(time.RFC3339),
	}

	resp, err := makeRequest("POST", fmt.Sprintf("/scheduling/job-sites/%s/schedules", jobSiteID), createScheduleReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var schedule scheduleResponse
	err = json.NewDecoder(resp.Body).Decode(&schedule)
	require.NoError(t, err)
	require.NotEmpty(t, schedule.ID)
	resp.Body.Close()

	return &schedule
}

func setupTasks(t *testing.T, jobSiteID, workerID, scheduleID string) []string {
	now := time.Now()
	tasks := []map[string]interface{}{
		{
			"name":               "Task 1",
			"description":        "First test task",
			"worker_id":          workerID,
			"schedule_id":        scheduleID,
			"planned_start_time": now.Format(time.RFC3339),
			"planned_end_time":   now.Add(2 * time.Hour).Format(time.RFC3339),
		},
		{
			"name":               "Task 2",
			"description":        "Second test task",
			"worker_id":          workerID,
			"schedule_id":        scheduleID,
			"planned_start_time": now.Add(2 * time.Hour).Format(time.RFC3339),
			"planned_end_time":   now.Add(4 * time.Hour).Format(time.RFC3339),
		},
	}

	taskIDs := make([]string, len(tasks))
	for i, task := range tasks {
		resp, err := makeRequest("POST", fmt.Sprintf("/scheduling/job-sites/%s/tasks", jobSiteID), task)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		var taskResp struct {
			ID string `json:"id"`
		}
		err = json.NewDecoder(resp.Body).Decode(&taskResp)
		require.NoError(t, err)
		resp.Body.Close()

		taskIDs[i] = taskResp.ID
	}

	return taskIDs
}
