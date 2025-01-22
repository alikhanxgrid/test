package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"reference-app-wms-go/app/analytics"
	apiv2 "reference-app-wms-go/app/dwr/api/v2/openapi"
)

// LoadTestConfig defines the parameters for the load test
type LoadTestConfig struct {
	NumWorkers          int           // Number of concurrent workers
	NumJobSites         int           // Number of job sites
	TasksPerWorker      int           // Tasks per worker per day
	SimulationDays      int           // Number of days to simulate
	BlockageRate        float64       // Probability of task blockage (0-1)
	BreakFrequency      time.Duration // Average time between breaks
	BreakDuration       time.Duration // Average break duration
	TaskDuration        time.Duration // Average task duration
	ConcurrentCheckins  int           // Max workers checking in simultaneously
	MinSchedulesPerSite int           // Minimum number of schedules per job site
	MaxSchedulesPerSite int           // Maximum number of schedules per job site
}

// DefaultLoadTestConfig provides reasonable defaults for load testing
var DefaultLoadTestConfig = LoadTestConfig{
	NumWorkers:          100,
	NumJobSites:         5,
	TasksPerWorker:      20,
	SimulationDays:      2,
	BlockageRate:        0.1,
	BreakFrequency:      2 * time.Hour,
	BreakDuration:       15 * time.Minute,
	TaskDuration:        30 * time.Minute,
	ConcurrentCheckins:  100,
	MinSchedulesPerSite: 2,
	MaxSchedulesPerSite: 5,
}

type jobSiteInfo struct {
	ID       string
	Name     string
	Location string
}

// Add these types for tracking operational state
type taskOperation struct {
	TaskID    string
	WorkerID  apiv2.WorkerID
	Operation string
	Time      time.Time
	Status    apiv2.TaskStatus
	IsBlocked bool
	OnBreak   bool
	JobSiteID string // Changed from Location to JobSiteID
}

type simulationContext struct {
	jobSites []jobSiteInfo
	workers  []apiv2.WorkerID
	config   LoadTestConfig
	t        *testing.T
	client   *apiv2.ClientWithResponses // Add client to context
	// Add operational tracking
	operations []taskOperation
	mu         sync.Mutex // For concurrent access to operations
}

// Add this method to record operations
func (ctx *simulationContext) recordOperation(op taskOperation) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.operations = append(ctx.operations, op)
}

func TestAnalyticsLoad(t *testing.T) {
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
	}

	// Use default config for now
	config := DefaultLoadTestConfig
	ctx := &simulationContext{
		config: config,
		t:      t,
		client: createJobExecutionClient(t),
	}

	// Setup phase
	t.Log("Setting up simulation environment...")
	setupSimulation(ctx)

	// Run simulation for each day
	for day := 0; day < config.SimulationDays; day++ {
		simulateDay(ctx, day)
	}

	// Basic validation using existing analytics methods
	validateAnalytics(ctx)
}

func setupSimulation(ctx *simulationContext) {
	// Create job sites
	ctx.jobSites = make([]jobSiteInfo, ctx.config.NumJobSites)
	for i := 0; i < ctx.config.NumJobSites; i++ {
		site := createJobSite(ctx.t, fmt.Sprintf("Load Test Site %d", i+1), fmt.Sprintf("%d Test Avenue", i+1))
		ctx.jobSites[i] = jobSiteInfo{
			ID:       site.ID,
			Name:     site.Name,
			Location: site.Location,
		}
	}

	// Generate worker IDs using UUID
	ctx.workers = make([]apiv2.WorkerID, ctx.config.NumWorkers)
	for i := 0; i < ctx.config.NumWorkers; i++ {
		ctx.workers[i] = apiv2.WorkerID(GenerateWorkerID())
	}
}

func simulateDay(ctx *simulationContext, dayIndex int) {
	ctx.t.Logf("Simulating day %d...", dayIndex+1)

	// Create schedules for each job site
	schedules := make(map[string][]*scheduleResponse)
	for _, site := range ctx.jobSites {
		// Randomly determine number of schedules for this site
		numSchedules := rand.Intn(ctx.config.MaxSchedulesPerSite-ctx.config.MinSchedulesPerSite+1) + ctx.config.MinSchedulesPerSite
		ctx.t.Logf("Creating %d schedules for job site %s", numSchedules, site.ID)

		siteSchedules := make([]*scheduleResponse, numSchedules)
		for i := 0; i < numSchedules; i++ {
			schedule := createSchedule(ctx.t, site.ID, dayIndex)
			siteSchedules[i] = schedule
			ctx.t.Logf("Created schedule %d/%d with ID %s", i+1, numSchedules, schedule.ID)
		}
		schedules[site.ID] = siteSchedules
	}

	// Create tasks for each worker
	tasks := generateDailyTasks(ctx, schedules, dayIndex)

	// Simulate concurrent worker check-ins
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, ctx.config.ConcurrentCheckins)

	for workerIndex, workerID := range ctx.workers {
		wg.Add(1)
		go func(wID apiv2.WorkerID, wIndex int) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			// Assign worker to a job site
			siteIndex := wIndex % len(ctx.jobSites)
			site := ctx.jobSites[siteIndex]

			// Simulate worker day
			simulateWorkerDay(ctx, wID, site, tasks[wID], dayIndex)
		}(workerID, workerIndex)
	}

	wg.Wait()
}

func generateDailyTasks(ctx *simulationContext, schedules map[string][]*scheduleResponse, dayIndex int) map[apiv2.WorkerID][]string {
	tasks := make(map[apiv2.WorkerID][]string)
	baseTime := time.Now().AddDate(0, 0, dayIndex)

	// First, assign each worker to a job site
	workerSites := make(map[apiv2.WorkerID]int)
	for workerIndex, worker := range ctx.workers {
		// Consistently assign worker to a job site based on their index
		siteIndex := workerIndex % len(ctx.jobSites)
		workerSites[worker] = siteIndex
	}

	for _, worker := range ctx.workers {
		workerTasks := make([]string, ctx.config.TasksPerWorker)
		// Use the pre-assigned site for this worker
		siteIndex := workerSites[worker]
		site := ctx.jobSites[siteIndex]
		siteSchedules := schedules[site.ID]

		for i := 0; i < ctx.config.TasksPerWorker; i++ {
			// Randomly select a schedule for this task
			scheduleIndex := rand.Intn(len(siteSchedules))
			schedule := siteSchedules[scheduleIndex]
			ctx.t.Logf("Creating task for worker %s with schedule ID %s at site %s", worker, schedule.ID, site.ID)

			startTime := baseTime.Add(time.Duration(i) * ctx.config.TaskDuration)
			endTime := startTime.Add(ctx.config.TaskDuration)

			taskReq := map[string]interface{}{
				"name":               fmt.Sprintf("Task %d for %s", i+1, worker),
				"description":        fmt.Sprintf("Load test task %d", i+1),
				"worker_id":          string(worker),
				"schedule_id":        schedule.ID,
				"planned_start_time": startTime.Format(time.RFC3339),
				"planned_end_time":   endTime.Format(time.RFC3339),
			}

			resp, err := makeRequest("POST", fmt.Sprintf("/scheduling/job-sites/%s/tasks", site.ID), taskReq)
			require.NoError(ctx.t, err)
			require.Equal(ctx.t, 201, resp.StatusCode)

			var taskResp struct {
				ID         string `json:"id"`
				ScheduleID string `json:"schedule_id"`
			}
			err = json.NewDecoder(resp.Body).Decode(&taskResp)
			require.NoError(ctx.t, err)
			resp.Body.Close()

			require.NotEmpty(ctx.t, taskResp.ScheduleID, "Schedule ID should be set in task response")
			require.Equal(ctx.t, schedule.ID, taskResp.ScheduleID, "Task should have correct schedule ID")

			workerTasks[i] = taskResp.ID
		}

		tasks[worker] = workerTasks
	}

	return tasks
}

func simulateWorkerDay(ctx *simulationContext, workerID apiv2.WorkerID, site jobSiteInfo, tasks []string, simulationDay int) {
	t := ctx.t

	// Calculate base time for this simulation day
	simulationDate := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, simulationDay)

	// Check in at 9 AM
	checkInTime := simulationDate.Add(9 * time.Hour)
	checkInReq := apiv2.CheckInRequest{
		WorkerId:    workerID,
		CheckInTime: checkInTime,
		JobSiteId:   site.ID,
		Date:        simulationDate,
	}

	checkInResp, err := ctx.client.WorkerCheckInWithResponse(context.Background(), checkInReq)
	require.NoError(t, err)
	require.Equal(t, 200, checkInResp.StatusCode())
	require.NotEmpty(t, checkInResp.JSON200.WorkflowID)

	ctx.recordOperation(taskOperation{
		WorkerID:  workerID,
		Operation: "check-in",
		Time:      checkInTime,
		JobSiteID: site.ID,
	})

	// Start first task 5 minutes after check-in
	taskStartBase := checkInTime.Add(5 * time.Minute)
	var lastTaskEndTime time.Time

	// Process each task
	for taskIndex, taskID := range tasks {
		// Calculate task times based on simulation date
		// Each task gets a 30-minute slot, with 1-minute gaps between operations
		taskSlotStart := taskStartBase.Add(time.Duration(taskIndex) * (ctx.config.TaskDuration + time.Minute))

		// Start task at the beginning of its slot
		taskStartTime := taskSlotStart
		taskUpdate := apiv2.TaskUpdate{
			TaskId:     taskID,
			NewStatus:  apiv2.INPROGRESS,
			UpdateTime: taskStartTime,
			UpdatedBy:  workerID,
			Notes:      stringPtr(fmt.Sprintf("Starting task %d", taskIndex+1)),
		}

		startResp, err := ctx.client.UpdateTaskProgressWithResponse(context.Background(), taskUpdate)
		require.NoError(t, err)
		require.Equal(t, 200, startResp.StatusCode())

		ctx.recordOperation(taskOperation{
			TaskID:    taskID,
			WorkerID:  workerID,
			Operation: "task-update",
			Time:      taskStartTime,
			Status:    apiv2.INPROGRESS,
			JobSiteID: site.ID,
		})

		// Verify task state was updated correctly
		RequireEventually(t, "task status verification", func() (bool, error) {
			resp, err := makeRequest("GET", fmt.Sprintf("/analytics/workflow/%s/tasks", checkInResp.JSON200.WorkflowID), nil)
			if err != nil {
				return false, err
			}
			defer resp.Body.Close()

			var taskStatus analytics.WorkerTaskStatus
			if err := json.NewDecoder(resp.Body).Decode(&taskStatus); err != nil {
				return false, err
			}

			// Find the specific task
			var foundTask *analytics.TaskInfo
			for _, task := range taskStatus.CurrentTasks {
				if task.TaskID == taskID {
					foundTask = &task
					break
				}
			}

			if foundTask == nil {
				return false, fmt.Errorf("task %s not found", taskID)
			}

			return foundTask.Status == analytics.TaskStatusInProgress, nil
		})

		// Randomly simulate task blockage
		if rand.Float64() < ctx.config.BlockageRate {
			// Block 10 minutes into the task
			blockTime := taskStartTime.Add(10 * time.Minute)
			taskUpdate = apiv2.TaskUpdate{
				TaskId:     taskID,
				NewStatus:  apiv2.BLOCKED,
				UpdateTime: blockTime,
				UpdatedBy:  workerID,
				Notes:      stringPtr("Task blocked for testing"),
			}

			blockResp, err := ctx.client.UpdateTaskProgressWithResponse(context.Background(), taskUpdate)
			require.NoError(t, err)
			require.Equal(t, 200, blockResp.StatusCode())

			ctx.recordOperation(taskOperation{
				TaskID:    taskID,
				WorkerID:  workerID,
				Operation: "task-update",
				Time:      blockTime,
				Status:    apiv2.BLOCKED,
				IsBlocked: true,
				JobSiteID: site.ID,
			})

			// Verify task blockage state
			RequireEventually(t, "task blockage verification", func() (bool, error) {
				resp, err := makeRequest("GET", fmt.Sprintf("/analytics/workflow/%s/tasks/%s/blockage", checkInResp.JSON200.WorkflowID, taskID), nil)
				if err != nil {
					return false, err
				}
				defer resp.Body.Close()

				var blockageInfo analytics.TaskBlockageInfo
				if err := json.NewDecoder(resp.Body).Decode(&blockageInfo); err != nil {
					return false, err
				}

				return blockageInfo.IsBlocked, nil
			})

			// Resolve after 15 minutes, adding 1 minute to ensure sequence
			resumeTime := blockTime.Add(15*time.Minute + time.Minute)
			taskUpdate = apiv2.TaskUpdate{
				TaskId:     taskID,
				NewStatus:  apiv2.INPROGRESS,
				UpdateTime: resumeTime,
				UpdatedBy:  workerID,
				Notes:      stringPtr("Task unblocked"),
			}

			resumeResp, err := ctx.client.UpdateTaskProgressWithResponse(context.Background(), taskUpdate)
			require.NoError(t, err)
			require.Equal(t, 200, resumeResp.StatusCode())

			ctx.recordOperation(taskOperation{
				TaskID:    taskID,
				WorkerID:  workerID,
				Operation: "task-update",
				Time:      resumeTime,
				Status:    apiv2.INPROGRESS,
				JobSiteID: site.ID,
			})

			// Verify task is no longer blocked
			RequireEventually(t, "task unblocked verification", func() (bool, error) {
				resp, err := makeRequest("GET", fmt.Sprintf("/analytics/workflow/%s/tasks/%s/blockage", checkInResp.JSON200.WorkflowID, taskID), nil)
				if err != nil {
					return false, err
				}
				defer resp.Body.Close()

				var blockageInfo analytics.TaskBlockageInfo
				if err := json.NewDecoder(resp.Body).Decode(&blockageInfo); err != nil {
					return false, err
				}

				return !blockageInfo.IsBlocked, nil
			})
		}

		// Randomly take breaks
		if rand.Float64() < float64(ctx.config.TaskDuration/ctx.config.BreakFrequency) {
			// Take break 20 minutes into the task
			breakStartTime := taskStartTime.Add(20 * time.Minute)
			breakSignal := apiv2.BreakSignal{
				WorkerId:  workerID,
				StartTime: breakStartTime,
				IsOnBreak: true,
			}

			breakResp, err := ctx.client.SignalBreakWithResponse(context.Background(), breakSignal)
			require.NoError(t, err)
			require.Equal(t, 200, breakResp.StatusCode())

			ctx.recordOperation(taskOperation{
				WorkerID:  workerID,
				Operation: "break",
				Time:      breakStartTime,
				OnBreak:   true,
				JobSiteID: site.ID,
			})

			// Verify break started
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

			// End break after 15 minutes, adding 1 minute to ensure sequence
			breakEndTime := breakStartTime.Add(15*time.Minute + time.Minute)
			breakSignal.StartTime = breakEndTime
			breakSignal.IsOnBreak = false

			breakEndResp, err := ctx.client.SignalBreakWithResponse(context.Background(), breakSignal)
			require.NoError(t, err)
			require.Equal(t, 200, breakEndResp.StatusCode())

			ctx.recordOperation(taskOperation{
				WorkerID:  workerID,
				Operation: "break",
				Time:      breakEndTime,
				OnBreak:   false,
				JobSiteID: site.ID,
			})

			// Verify break ended
			RequireEventually(t, "break ended verification", func() (bool, error) {
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
		}

		// Complete task at the end of its slot, 1 minute before next task starts
		taskEndTime := taskSlotStart.Add(ctx.config.TaskDuration - time.Minute)
		taskUpdate = apiv2.TaskUpdate{
			TaskId:     taskID,
			NewStatus:  apiv2.COMPLETED,
			UpdateTime: taskEndTime,
			UpdatedBy:  workerID,
			Notes:      stringPtr(fmt.Sprintf("Completed task %d", taskIndex+1)),
		}

		completeResp, err := ctx.client.UpdateTaskProgressWithResponse(context.Background(), taskUpdate)
		require.NoError(t, err)
		require.Equal(t, 200, completeResp.StatusCode())

		ctx.recordOperation(taskOperation{
			TaskID:    taskID,
			WorkerID:  workerID,
			Operation: "task-update",
			Time:      taskEndTime,
			Status:    apiv2.COMPLETED,
			JobSiteID: site.ID,
		})

		lastTaskEndTime = taskEndTime

		// Verify task completion
		RequireEventually(t, "task completion verification", func() (bool, error) {
			resp, err := makeRequest("GET", fmt.Sprintf("/analytics/workflow/%s/tasks", checkInResp.JSON200.WorkflowID), nil)
			if err != nil {
				return false, err
			}
			defer resp.Body.Close()

			var taskStatus analytics.WorkerTaskStatus
			if err := json.NewDecoder(resp.Body).Decode(&taskStatus); err != nil {
				return false, err
			}

			// Find the specific task
			var foundTask *analytics.TaskInfo
			for _, task := range taskStatus.CurrentTasks {
				if task.TaskID == taskID {
					foundTask = &task
					break
				}
			}

			if foundTask == nil {
				return false, fmt.Errorf("task %s not found", taskID)
			}

			return foundTask.Status == analytics.TaskStatusCompleted, nil
		})
	}

	// Check out 15 minutes after the last task is completed
	checkOutTime := lastTaskEndTime.Add(15 * time.Minute)
	checkOutReq := apiv2.CheckOutRequest{
		WorkerId:     workerID,
		CheckOutTime: checkOutTime,
		JobSiteId:    site.ID,
		Notes:        stringPtr("Completed all tasks"),
		Date:         simulationDate,
	}

	checkOutResp, err := ctx.client.WorkerCheckOutWithResponse(context.Background(), checkOutReq)
	require.NoError(t, err)
	require.Equal(t, 200, checkOutResp.StatusCode())

	ctx.recordOperation(taskOperation{
		WorkerID:  workerID,
		Operation: "check-out",
		Time:      checkOutTime,
		JobSiteID: site.ID,
	})
}

func validateAnalytics(ctx *simulationContext) {
	t := ctx.t
	t.Log("Validating analytics data...")

	// First validate operational correctness
	validateOperationalState(ctx)

	// TODO(TEAM) : Add validation for analytics data, right now the endpoint only queries the current tasks from
	// the workflow, we need to query the analytics service to get the historical data from the database, this will fail since
	// the workflow has ended because we check-out, no idea how it ever worked before :)
	// Then validate analytics data
	// for _, worker := range ctx.workers {
	// 	workflowID := string(worker)

	// 	// Validate final worker state
	// 	RequireEventually(t, "worker task status validation", func() (bool, error) {
	// 		resp, err := makeRequest("GET", fmt.Sprintf("/analytics/workflow/%s/tasks", workflowID), nil)
	// 		if err != nil {
	// 			return false, err
	// 		}
	// 		defer resp.Body.Close()

	// 		var status analytics.WorkerTaskStatus
	// 		if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
	// 			return false, err
	// 		}

	// 		return !status.IsSessionActive && status.CurrentSite != "", nil
	// 	})

	// 	// TODO: Add validation for historical analytics once implemented:
	// 	// - Task completion rates
	// 	// - Break patterns
	// 	// - Blockage statistics
	// 	// - Worker productivity metrics
	// }
}

func validateOperationalState(ctx *simulationContext) {
	t := ctx.t
	t.Log("Validating operational state...")

	// Group operations by worker
	workerOps := make(map[apiv2.WorkerID][]taskOperation)
	// Track task metadata for validation
	taskMeta := make(map[string]taskMetadata)

	// First pass: collect all task metadata
	for _, op := range ctx.operations {
		workerOps[op.WorkerID] = append(workerOps[op.WorkerID], op)
		if op.Operation == "task-update" {
			if meta, exists := taskMeta[op.TaskID]; exists {
				meta.Status = op.Status
				taskMeta[op.TaskID] = meta
			} else {
				taskMeta[op.TaskID] = taskMetadata{
					TaskID: op.TaskID,
					Status: op.Status,
				}
			}
		}
	}

	// Fetch and validate task metadata
	for taskID := range taskMeta {
		resp, err := makeRequest("GET", fmt.Sprintf("/scheduling/tasks/%s", taskID), nil)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode)

		var taskDetails struct {
			ID         string `json:"id"`
			ScheduleID string `json:"schedule_id"`
		}
		err = json.NewDecoder(resp.Body).Decode(&taskDetails)
		require.NoError(t, err)
		resp.Body.Close()

		// Add debug logging
		t.Logf("Task %s has schedule ID: %s", taskID, taskDetails.ScheduleID)
		require.NotEmpty(t, taskDetails.ScheduleID, "Schedule ID should not be empty for task %s", taskID)

		// Get schedule details to find job site ID
		resp, err = makeRequest("GET", fmt.Sprintf("/scheduling/schedules/%s", taskDetails.ScheduleID), nil)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode)

		var scheduleDetails struct {
			ID        string `json:"id"`
			JobSiteID string `json:"job_site_id"`
		}
		err = json.NewDecoder(resp.Body).Decode(&scheduleDetails)
		require.NoError(t, err)
		resp.Body.Close()

		meta := taskMeta[taskID]
		meta.ScheduleID = taskDetails.ScheduleID
		meta.JobSiteID = scheduleDetails.JobSiteID
		taskMeta[taskID] = meta
	}

	// Validate each worker's operations
	for workerID, ops := range workerOps {
		t.Logf("Validating operations for worker %s", workerID)

		// Verify check-in/check-out
		require.Equal(t, "check-in", ops[0].Operation, "First operation should be check-in")
		require.Equal(t, "check-out", ops[len(ops)-1].Operation, "Last operation should be check-out")

		// Track task states and worker's current job site
		taskStates := make(map[string]apiv2.TaskStatus)
		var isOnBreak bool
		var currentJobSite string

		// Validate operation sequence
		for i, op := range ops {
			switch op.Operation {
			case "check-in":
				currentJobSite = op.JobSiteID
			case "task-update":
				// Verify task belongs to worker's current job site
				meta := taskMeta[op.TaskID]
				require.Equal(t, currentJobSite, meta.JobSiteID,
					"Task %s belongs to job site %s but worker is at %s",
					op.TaskID, meta.JobSiteID, currentJobSite)

				// Verify valid state transitions
				if prevStatus, exists := taskStates[op.TaskID]; exists {
					validateTaskStateTransition(t, prevStatus, op.Status)
				}
				taskStates[op.TaskID] = op.Status

				// Verify no task updates during break
				require.False(t, isOnBreak, "No task updates should occur during break")

			case "break":
				if op.OnBreak {
					require.False(t, isOnBreak, "Break started while already on break")
				} else {
					require.True(t, isOnBreak, "Break ended without being on break")
				}
				isOnBreak = op.OnBreak
			}

			// Verify timestamps are in sequence
			if i > 0 {
				prevOp := ops[i-1]
				t.Logf("Comparing operations: Previous [%s at %v] -> Current [%s at %v]",
					prevOp.Operation, prevOp.Time.Format(time.RFC3339),
					op.Operation, op.Time.Format(time.RFC3339))

				if !op.Time.After(prevOp.Time) {
					t.Logf("ERROR: Operation sequence violation:")
					t.Logf("  Previous operation: %+v", prevOp)
					t.Logf("  Current operation: %+v", op)
					t.Logf("  Time difference: %v", op.Time.Sub(prevOp.Time))
				}
				require.True(t, op.Time.After(prevOp.Time),
					"Operations should be in chronological order: %s (%v) -> %s (%v)",
					prevOp.Operation, prevOp.Time.Format(time.RFC3339),
					op.Operation, op.Time.Format(time.RFC3339))
			}
		}

		// Verify all tasks are completed
		for taskID, status := range taskStates {
			require.Equal(t, apiv2.COMPLETED, status, "Task %s should be completed", taskID)
		}

		// Verify all tasks for this worker belong to the same job site
		workerTasks := make(map[string]bool)
		for _, op := range ops {
			if op.Operation == "task-update" {
				workerTasks[op.TaskID] = true
			}
		}

		var lastJobSiteID string
		for taskID := range workerTasks {
			meta := taskMeta[taskID]
			if lastJobSiteID == "" {
				lastJobSiteID = meta.JobSiteID
			} else {
				require.Equal(t, lastJobSiteID, meta.JobSiteID,
					"All tasks for worker should belong to the same job site")
			}
		}
	}
}

// Add this type to track relationships
type taskMetadata struct {
	TaskID     string
	ScheduleID string
	JobSiteID  string
	Status     apiv2.TaskStatus
}

func validateTaskStateTransition(t *testing.T, from, to apiv2.TaskStatus) {
	// Define valid state transitions
	validTransitions := map[apiv2.TaskStatus][]apiv2.TaskStatus{
		apiv2.PENDING:    {apiv2.INPROGRESS},
		apiv2.INPROGRESS: {apiv2.BLOCKED, apiv2.COMPLETED},
		apiv2.BLOCKED:    {apiv2.INPROGRESS},
	}

	valid := false
	for _, allowedStatus := range validTransitions[from] {
		if to == allowedStatus {
			valid = true
			break
		}
	}

	require.True(t, valid, "Invalid task state transition from %s to %s", from, to)
}

func createJobSite(t *testing.T, name, location string) *jobSiteResponse {
	createJobSiteReq := map[string]string{
		"name":     name,
		"location": location,
	}

	resp, err := makeRequest("POST", "/scheduling/job-sites", createJobSiteReq)
	require.NoError(t, err)
	require.Equal(t, 201, resp.StatusCode)

	var jobSite jobSiteResponse
	err = json.NewDecoder(resp.Body).Decode(&jobSite)
	require.NoError(t, err)
	require.NotEmpty(t, jobSite.ID)
	resp.Body.Close()

	return &jobSite
}

func createSchedule(t *testing.T, jobSiteID string, dayOffset int) *scheduleResponse {
	baseTime := time.Now().AddDate(0, 0, dayOffset)
	startOfDay := time.Date(baseTime.Year(), baseTime.Month(), baseTime.Day(), 0, 0, 0, 0, baseTime.Location())
	shiftDuration := 4 * time.Hour
	maxRetries := 10

	// Try different time slots until we find one that works
	for retry := 0; retry < maxRetries; retry++ {
		// Try different parts of the day for each retry
		startHour := (retry * 4) % 20 // Divide day into 4-hour slots
		startTime := startOfDay.Add(time.Duration(startHour) * time.Hour)
		endTime := startTime.Add(shiftDuration)

		scheduleID := GenerateWorkerID()
		t.Logf("Attempt %d: Creating schedule with ID %s for job site %s (time: %s - %s)",
			retry+1, scheduleID, jobSiteID, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))

		createScheduleReq := map[string]string{
			"id":        scheduleID,
			"startDate": startTime.Format(time.RFC3339),
			"endDate":   endTime.Format(time.RFC3339),
		}

		resp, err := makeRequest("POST", fmt.Sprintf("/scheduling/job-sites/%s/schedules", jobSiteID), createScheduleReq)
		if err != nil {
			continue
		}

		if resp.StatusCode == 409 {
			t.Logf("Schedule conflict detected, retrying with different time slot")
			resp.Body.Close()
			continue
		}

		require.Equal(t, 201, resp.StatusCode)

		var s scheduleResponse
		err = json.NewDecoder(resp.Body).Decode(&s)
		resp.Body.Close()
		if err != nil {
			continue
		}

		require.NotEmpty(t, s.ID)
		require.Equal(t, scheduleID, s.ID, "Server returned different schedule ID than requested")
		t.Logf("Successfully created schedule with ID %s", s.ID)
		return &s
	}

	require.Fail(t, fmt.Sprintf("Failed to create schedule after %d attempts", maxRetries))
	return nil
}
