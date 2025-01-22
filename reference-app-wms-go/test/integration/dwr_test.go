package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	apiv2 "reference-app-wms-go/app/dwr/api/v2/openapi"

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

	// Create API client
	client := createJobExecutionClient(t)
	ctx := context.Background()

	// 1. Worker Check-in
	t.Log("Testing worker check-in...")
	checkInReq := apiv2.CheckInRequest{
		WorkerId:    apiv2.WorkerID(workerID),
		CheckInTime: time.Now(),
		JobSiteId:   data.jobSiteID,
		Date:        time.Now(),
	}

	checkInResp, err := client.WorkerCheckInWithResponse(ctx, checkInReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, checkInResp.StatusCode())
	require.NotEmpty(t, checkInResp.JSON200.WorkflowID)

	// Wait for workflow to initialize and retrieve tasks
	t.Log("Waiting for workflow to initialize and retrieve tasks...")
	time.Sleep(2 * time.Second)

	// 2. Start first task
	t.Log("Starting first task...")
	taskUpdate := apiv2.TaskUpdate{
		TaskId:     data.tasks[0], // Use actual task ID
		NewStatus:  apiv2.INPROGRESS,
		UpdateTime: time.Now(),
		UpdatedBy:  apiv2.WorkerID(workerID),
		Notes:      stringPtr("Starting work on windows"),
	}

	taskUpdateResp, err := client.UpdateTaskProgressWithResponse(ctx, taskUpdate)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, taskUpdateResp.StatusCode())

	// 3. Take a break
	t.Log("Starting break...")
	breakStartTime := time.Now()
	breakSignal := apiv2.BreakSignal{
		WorkerId:  apiv2.WorkerID(workerID),
		StartTime: breakStartTime,
		IsOnBreak: true,
	}

	breakResp, err := client.SignalBreakWithResponse(ctx, breakSignal)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, breakResp.StatusCode())

	// Wait for break duration
	time.Sleep(2 * time.Second)

	// 4. End break
	t.Log("Ending break...")
	breakSignal.IsOnBreak = false
	breakSignal.StartTime = time.Now()

	breakEndResp, err := client.SignalBreakWithResponse(ctx, breakSignal)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, breakEndResp.StatusCode())

	// 5. Complete first task
	t.Log("Completing first task...")
	taskUpdate.NewStatus = apiv2.COMPLETED
	taskUpdate.UpdateTime = time.Now()
	taskUpdate.Notes = stringPtr("Windows installation completed")

	completeResp, err := client.UpdateTaskProgressWithResponse(ctx, taskUpdate)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, completeResp.StatusCode())

	// 6. Start and complete second task
	t.Log("Working on second task...")
	taskUpdate.TaskId = data.tasks[1] // Use second task ID
	taskUpdate.NewStatus = apiv2.INPROGRESS
	taskUpdate.UpdateTime = time.Now()
	taskUpdate.Notes = stringPtr("Starting door handle replacement")

	startSecondResp, err := client.UpdateTaskProgressWithResponse(ctx, taskUpdate)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, startSecondResp.StatusCode())

	time.Sleep(2 * time.Second)

	taskUpdate.NewStatus = apiv2.COMPLETED
	taskUpdate.UpdateTime = time.Now()
	taskUpdate.Notes = stringPtr("Door handles replaced")

	completeSecondResp, err := client.UpdateTaskProgressWithResponse(ctx, taskUpdate)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, completeSecondResp.StatusCode())

	// 7. Worker Check-out
	t.Log("Testing worker check-out...")
	checkOutReq := apiv2.CheckOutRequest{
		WorkerId:     apiv2.WorkerID(workerID),
		CheckOutTime: time.Now(),
		JobSiteId:    data.jobSiteID,
		Notes:        stringPtr("Completed all tasks for the day"),
		Date:         time.Now(),
	}

	checkOutResp, err := client.WorkerCheckOutWithResponse(ctx, checkOutReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, checkOutResp.StatusCode())

	t.Log("Daily worker routine test completed successfully")
}
