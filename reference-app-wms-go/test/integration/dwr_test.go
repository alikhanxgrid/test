package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"reference-app-wms-go/app/dwr/model"

	"github.com/stretchr/testify/require"
)

const workerID = "550e8400-e29b-41d4-a716-446655440000"

type testData struct {
	jobSiteID  string
	scheduleID string
	tasks      []string // task IDs
}

// setupTestData creates necessary test data in the database
func setupTestData(t *testing.T) testData {
	t.Log("Setting up test data...")

	// 1. Create a job site
	jobSiteReq := map[string]string{
		"name":     "Test Site A",
		"location": "123 Test St",
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

	// 2. Create a schedule for today
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

	// 3. Create test tasks
	taskIDs := make([]string, 2)
	tasks := []map[string]interface{}{
		{
			"name":               "Install Windows",
			"description":        "Install new windows in room 101",
			"worker_id":          workerID,
			"schedule_id":        schedule.ID,
			"planned_start_time": now.Format(time.RFC3339),
			"planned_end_time":   now.Add(2 * time.Hour).Format(time.RFC3339),
		},
		{
			"name":               "Replace Door Handles",
			"description":        "Replace all door handles in section A",
			"worker_id":          workerID,
			"schedule_id":        schedule.ID,
			"planned_start_time": now.Add(2 * time.Hour).Format(time.RFC3339),
			"planned_end_time":   now.Add(4 * time.Hour).Format(time.RFC3339),
		},
	}

	for i, task := range tasks {
		resp, err = makeRequest("POST", fmt.Sprintf("/scheduling/job-sites/%s/tasks", jobSite.ID), task)
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

	return testData{
		jobSiteID:  jobSite.ID,
		scheduleID: schedule.ID,
		tasks:      taskIDs,
	}
}

// TestDailyWorkerRoutine simulates a full day of worker activities
func TestDailyWorkerRoutine(t *testing.T) {
	// Set up test data
	data := setupTestData(t)
	t.Logf("Test data created - Job Site: %s, Schedule: %s, Tasks: %v", data.jobSiteID, data.scheduleID, data.tasks)

	// 1. Worker Check-in
	t.Log("Testing worker check-in...")
	checkInReq := model.CheckInRequest{
		WorkerID:    workerID,
		CheckInTime: time.Now(),
		JobSiteID:   data.jobSiteID,
	}

	resp, err := makeRequest("POST", "/check-in", checkInReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var workflowResp workflowResponse
	err = json.NewDecoder(resp.Body).Decode(&workflowResp)
	require.NoError(t, err)
	require.NotEmpty(t, workflowResp.WorkflowID)
	resp.Body.Close()

	// Wait for workflow to initialize and retrieve tasks
	t.Log("Waiting for workflow to initialize and retrieve tasks...")
	time.Sleep(2 * time.Second)

	// 2. Start first task
	t.Log("Starting first task...")
	taskUpdate := model.TaskUpdate{
		TaskID:     data.tasks[0], // Use actual task ID
		NewStatus:  model.TaskStatusInProgress,
		UpdateTime: time.Now(),
		UpdatedBy:  workerID,
		Notes:      "Starting work on windows",
	}

	resp, err = makeRequest("POST", "/task-progress", taskUpdate)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// 3. Take a break
	t.Log("Starting break...")
	breakStartTime := time.Now()
	breakSignal := struct {
		WorkerID  string    `json:"workerId"`
		StartTime time.Time `json:"startTime"`
		IsStart   bool      `json:"isStart"`
	}{
		WorkerID:  string(workerID),
		StartTime: breakStartTime,
		IsStart:   true,
	}

	resp, err = makeRequest("POST", "/break", breakSignal)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Wait for break duration
	time.Sleep(2 * time.Second)

	// 4. End break
	t.Log("Ending break...")
	breakSignal.IsStart = false
	breakSignal.StartTime = time.Now()

	resp, err = makeRequest("POST", "/break", breakSignal)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// 5. Complete first task
	t.Log("Completing first task...")
	taskUpdate.NewStatus = model.TaskStatusCompleted
	taskUpdate.UpdateTime = time.Now()
	taskUpdate.Notes = "Windows installation completed"

	resp, err = makeRequest("POST", "/task-progress", taskUpdate)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// 6. Start and complete second task
	t.Log("Working on second task...")
	taskUpdate.TaskID = data.tasks[1] // Use second task ID
	taskUpdate.NewStatus = model.TaskStatusInProgress
	taskUpdate.UpdateTime = time.Now()
	taskUpdate.Notes = "Starting door handle replacement"

	resp, err = makeRequest("POST", "/task-progress", taskUpdate)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	time.Sleep(2 * time.Second)

	taskUpdate.NewStatus = model.TaskStatusCompleted
	taskUpdate.UpdateTime = time.Now()
	taskUpdate.Notes = "Door handles replaced"

	resp, err = makeRequest("POST", "/task-progress", taskUpdate)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// 7. Worker Check-out
	t.Log("Testing worker check-out...")
	checkOutReq := model.CheckOutRequest{
		WorkerID:     workerID,
		CheckOutTime: time.Now(),
		JobSiteID:    data.jobSiteID,
		Notes:        "Completed all tasks for the day",
	}

	resp, err = makeRequest("POST", "/check-out", checkOutReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	t.Log("Daily worker routine test completed successfully")
}
