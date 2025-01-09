package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"reference-app-wms-go/app/analytics"
	"reference-app-wms-go/app/dwr/model"
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

	// Set up test data
	workerID := GenerateWorkerID()
	jobSite := setupJobSite(t)
	schedule := setupSchedule(t, jobSite.ID)
	tasks := setupTasks(t, jobSite.ID, workerID, schedule.ID)

	// 1. Worker Check-in
	t.Log("Testing worker check-in...")
	checkInReq := model.CheckInRequest{
		WorkerID:    model.WorkerID(workerID),
		CheckInTime: time.Now(),
		JobSiteID:   jobSite.ID,
	}

	resp, err := makeRequest("POST", "/check-in", checkInReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var checkInResp struct {
		WorkflowID string `json:"workflowID"`
		RunID      string `json:"runID"`
	}
	err = json.NewDecoder(resp.Body).Decode(&checkInResp)
	require.NoError(t, err)
	require.NotEmpty(t, checkInResp.WorkflowID)
	resp.Body.Close()

	// 2. Verify initial task state
	t.Log("Verifying initial task state...")
	var taskStatus analytics.WorkerTaskStatus
	RequireEventually(t, "initial task state verification", func() (bool, error) {
		resp, err := makeRequest("GET", fmt.Sprintf("/analytics/workflow/%s/tasks", checkInResp.WorkflowID), nil)
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
	resp, err = makeRequest("GET", fmt.Sprintf("/analytics/workflow/%s/tasks", checkInResp.WorkflowID), nil)
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
	taskUpdate := model.TaskUpdate{
		TaskID:     tasks[0],
		NewStatus:  model.TaskStatusInProgress,
		UpdateTime: time.Now(),
		UpdatedBy:  model.WorkerID(workerID),
		Notes:      "Starting first task",
	}

	resp, err = makeRequest("POST", "/task-progress", taskUpdate)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Verify task status after update
	t.Log("Verifying task status update...")
	var updatedTask *analytics.TaskInfo
	RequireEventually(t, "task status update verification", func() (bool, error) {
		resp, err := makeRequest("GET", fmt.Sprintf("/analytics/workflow/%s/tasks", checkInResp.WorkflowID), nil)
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
	blockageUpdate := model.TaskUpdate{
		TaskID:     tasks[0],
		NewStatus:  model.TaskStatusBlocked,
		UpdateTime: time.Now(),
		UpdatedBy:  model.WorkerID(workerID),
		Notes:      "Missing materials",
	}

	resp, err = makeRequest("POST", "/task-progress", blockageUpdate)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Verify task blockage
	t.Log("Verifying task blockage...")
	var blockageInfo analytics.TaskBlockageInfo
	RequireEventually(t, "task blockage verification", func() (bool, error) {
		resp, err := makeRequest("GET", fmt.Sprintf("/analytics/workflow/%s/tasks/%s/blockage", checkInResp.WorkflowID, tasks[0]), nil)
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
	breakSignal := model.BreakSignal{
		WorkerID:  model.WorkerID(workerID),
		IsOnBreak: true,
		StartTime: breakStartTime,
	}

	resp, err = makeRequest("POST", "/break", breakSignal)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Verify break status
	t.Log("Verifying break status...")
	RequireEventually(t, "break status verification", func() (bool, error) {
		resp, err := makeRequest("GET", fmt.Sprintf("/analytics/workflow/%s/tasks", checkInResp.WorkflowID), nil)
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

	resp, err = makeRequest("POST", "/break", breakSignal)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Verify break ended
	t.Log("Verifying break ended...")
	RequireEventually(t, "break end verification", func() (bool, error) {
		resp, err := makeRequest("GET", fmt.Sprintf("/analytics/workflow/%s/tasks", checkInResp.WorkflowID), nil)
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
	resumeUpdate := model.TaskUpdate{
		TaskID:     tasks[0],
		NewStatus:  model.TaskStatusInProgress,
		UpdateTime: time.Now(),
		UpdatedBy:  model.WorkerID(workerID),
		Notes:      "Materials available now",
	}

	resp, err = makeRequest("POST", "/task-progress", resumeUpdate)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Then complete it
	completeUpdate := model.TaskUpdate{
		TaskID:     tasks[0],
		NewStatus:  model.TaskStatusCompleted,
		UpdateTime: time.Now(),
		UpdatedBy:  model.WorkerID(workerID),
		Notes:      "Task completed successfully",
	}

	resp, err = makeRequest("POST", "/task-progress", completeUpdate)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// 9. Check-out
	t.Log("Testing worker check-out...")
	checkOutReq := model.CheckOutRequest{
		WorkerID:     model.WorkerID(workerID),
		CheckOutTime: time.Now(),
		JobSiteID:    jobSite.ID,
		Notes:        "Completed all tasks for the day",
	}

	resp, err = makeRequest("POST", "/check-out", checkOutReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// 10. Verify checked-out state
	t.Log("Verifying checked-out state...")
	RequireEventually(t, "check-out verification", func() (bool, error) {
		resp, err := makeRequest("GET", fmt.Sprintf("/analytics/workflow/%s/tasks", checkInResp.WorkflowID), nil)
		if err != nil {
			return false, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		var finalStatus analytics.WorkerTaskStatus
		if err := json.NewDecoder(resp.Body).Decode(&finalStatus); err != nil {
			return false, err
		}

		return !finalStatus.IsSessionActive, nil
	})

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
