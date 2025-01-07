package workflows

import (
	"fmt"
	"time"

	"reference-app-wms-go/app/analytics"
	"reference-app-wms-go/app/dwr/model"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// DailyWorkerRoutineParams contains the input parameters for the DailyWorkerRoutine workflow
type DailyWorkerRoutineParams struct {
	WorkerID    model.WorkerID `json:"workerId"`
	CheckInTime time.Time      `json:"checkInTime"`
	JobSiteID   string         `json:"jobSiteId"`
}

// DailyWorkerRoutineState maintains the workflow state
type DailyWorkerRoutineState struct {
	Schedule     model.DailySchedule
	WorkerState  model.WorkerState
	IsCheckedOut bool
}

// TaskUpdateSignal represents a signal for updating task status
type TaskUpdateSignal struct {
	Update model.TaskUpdate
}

// BreakSignal represents signals for break management
type BreakSignal struct {
	StartTime time.Time
	IsOnBreak bool
}

// DailyWorkerRoutine is the main workflow that manages a worker's daily activities
func DailyWorkerRoutine(ctx workflow.Context, params DailyWorkerRoutineParams) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting daily worker routine", "params", fmt.Sprintf("%+v", params))

	// Set activity options
	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute * 5,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	// Record check-in activity
	var schedule model.DailySchedule
	err := workflow.ExecuteActivity(ctx, "RecordCheckInActivity", params).Get(ctx, &schedule)
	if err != nil {
		logger.Error("Failed to record check-in", "error", err)
		return err
	}

	// Initialize workflow state
	state := &DailyWorkerRoutineState{
		Schedule: schedule,
		WorkerState: model.WorkerState{
			WorkerID:    params.WorkerID,
			Status:      model.WorkerStatusOnDuty,
			LastUpdated: params.CheckInTime,
			BreakState: model.BreakState{
				IsOnBreak:    false,
				TotalBreaks:  0,
				BreakMinutes: 0,
			},
		},
	}

	// Set up query handlers
	if err := setupQueryHandlers(ctx, state, params); err != nil {
		return err
	}

	// Set up signal channels
	taskProgressChan := workflow.GetSignalChannel(ctx, "task-progress-signal")
	breakChan := workflow.GetSignalChannel(ctx, "break-signal")
	checkOutChan := workflow.GetSignalChannel(ctx, "check-out")

	// Retrieve daily tasks
	var tasks []model.Task
	err = workflow.ExecuteActivity(ctx, "RetrieveDailyTasksActivity", params.WorkerID).Get(ctx, &tasks)
	if err != nil {
		logger.Error("Failed to retrieve daily tasks", "error", err)
		return err
	}
	state.Schedule.Tasks = tasks

	// Create selector for handling signals
	selector := workflow.NewSelector(ctx)

	selector.AddReceive(taskProgressChan, func(ch workflow.ReceiveChannel, more bool) {
		var signal model.TaskUpdate
		ch.Receive(ctx, &signal)
		logger.Info("Received task progress signal", "signal", fmt.Sprintf("%+v", signal))

		// Update local state first
		handleTaskProgressSignal(ctx, signal, &state.Schedule)

		// Then update the DB
		err := workflow.ExecuteActivity(ctx, "UpdateTaskProgressActivity", signal).Get(ctx, nil)
		if err != nil {
			logger.Error("Failed to update task progress", "error", err)
			return
		}
	})

	selector.AddReceive(breakChan, func(ch workflow.ReceiveChannel, more bool) {
		var signal BreakSignal
		ch.Receive(ctx, &signal)
		handleBreakSignal(ctx, state, signal)
	})

	selector.AddReceive(checkOutChan, func(ch workflow.ReceiveChannel, more bool) {
		var checkOutReq model.CheckOutRequest
		ch.Receive(ctx, &checkOutReq)
		logger.Info("Received check-out signal", "request", fmt.Sprintf("%+v", checkOutReq))

		// Update state with check-out information
		state.Schedule.CheckOut = checkOutReq.CheckOutTime
		state.WorkerState.Status = model.WorkerStatusCheckedOut
		state.WorkerState.LastUpdated = checkOutReq.CheckOutTime
		state.IsCheckedOut = true

		logger.Info("Updated workflow state for check-out",
			"workerID", state.WorkerState.WorkerID,
			"status", state.WorkerState.Status,
			"checkOutTime", state.Schedule.CheckOut)

		// Execute check-out activity immediately
		err := workflow.ExecuteActivity(ctx, "RecordCheckOutActivity", checkOutReq).Get(ctx, nil)
		if err != nil {
			logger.Error("Failed to record check-out", "error", err)
		}
	})

	// Continue processing signals until checked out
	for !state.IsCheckedOut {
		selector.Select(ctx)
	}

	// Workflow is complete
	return nil
}

func handleBreakSignal(ctx workflow.Context, state *DailyWorkerRoutineState, signal BreakSignal) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Received break signal",
		"signal", fmt.Sprintf("%+v", signal),
		"currentState", fmt.Sprintf("%+v", state.WorkerState))

	if signal.IsOnBreak {
		logger.Info("Starting break",
			"workerID", state.WorkerState.WorkerID,
			"currentStatus", state.WorkerState.Status,
			"totalBreaks", state.WorkerState.BreakState.TotalBreaks)

		state.WorkerState.Status = model.WorkerStatusOnBreak
		state.WorkerState.BreakState.IsOnBreak = true
		state.WorkerState.BreakState.StartTime = signal.StartTime
		state.WorkerState.BreakState.TotalBreaks++
	} else {
		logger.Info("Ending break",
			"workerID", state.WorkerState.WorkerID,
			"currentStatus", state.WorkerState.Status,
			"totalBreaks", state.WorkerState.BreakState.TotalBreaks)

		state.WorkerState.Status = model.WorkerStatusOnDuty
		state.WorkerState.BreakState.IsOnBreak = false
		// Calculate break duration and add to total
		duration := time.Since(state.WorkerState.BreakState.StartTime)
		state.WorkerState.BreakState.BreakMinutes += int(duration.Minutes())
	}
	state.WorkerState.LastUpdated = signal.StartTime

	logger.Info("Updated worker state after break",
		"status", state.WorkerState.Status,
		"totalBreaks", state.WorkerState.BreakState.TotalBreaks,
		"breakMinutes", state.WorkerState.BreakState.BreakMinutes,
		"lastUpdated", state.WorkerState.LastUpdated)
}

func setupQueryHandlers(ctx workflow.Context, state *DailyWorkerRoutineState, params DailyWorkerRoutineParams) error {
	// Handler for getCurrentTasks query
	if err := workflow.SetQueryHandler(ctx, "getCurrentTasks", func() (*analytics.WorkerTaskStatus, error) {
		logger := workflow.GetLogger(ctx)
		currentTasks := make([]analytics.TaskInfo, 0)

		logger.Info("Processing getCurrentTasks query",
			"totalTasks", len(state.Schedule.Tasks),
			"tasks", fmt.Sprintf("%+v", state.Schedule.Tasks))

		for _, task := range state.Schedule.Tasks {
			// TODO(Abdullah) : Revisit this logic, do we want to include completed tasks?
			// Only include tasks that are not completed
			// if task.Status != model.TaskStatusCompleted {
			taskInfo := analytics.TaskInfo{
				TaskID:      task.ID,
				Status:      convertTaskStatus(task.Status),
				IsBlocked:   task.Status == model.TaskStatusBlocked,
				BlockReason: task.BlockReason,
			}
			if !task.PlannedStartTime.IsZero() {
				taskInfo.StartedAt = task.PlannedStartTime.Unix()
			}
			currentTasks = append(currentTasks, taskInfo)
			logger.Info("Added task to current tasks",
				"taskID", task.ID,
				"status", task.Status,
				"isCompleted", task.Status == model.TaskStatusCompleted)
		}
		// }

		return &analytics.WorkerTaskStatus{
			CurrentTasks:    currentTasks,
			IsSessionActive: !state.IsCheckedOut,
			CurrentSite:     params.JobSiteID,
			IsOnBreak:       state.WorkerState.BreakState.IsOnBreak,
		}, nil
	}); err != nil {
		return fmt.Errorf("failed to set getCurrentTasks query handler: %w", err)
	}

	// Handler for getTaskBlockage query
	if err := workflow.SetQueryHandler(ctx, "getTaskBlockage", func(taskID string) (*analytics.TaskBlockageInfo, error) {
		logger := workflow.GetLogger(ctx)
		for _, task := range state.Schedule.Tasks {
			if task.ID == taskID {
				logger.Info("============Task found", "taskID", taskID, "status", task.Status, "isBlocked", task.IsBlocked)
				return &analytics.TaskBlockageInfo{
					IsBlocked:   task.Status == model.TaskStatusBlocked,
					BlockReason: task.BlockReason,
					Since:       task.BlockedAt.Unix(),
				}, nil
			}
		}
		return nil, fmt.Errorf("task %s not found", taskID)
	}); err != nil {
		return fmt.Errorf("failed to set getTaskBlockage query handler: %w", err)
	}

	return nil
}

// Helper function to convert between task status types
func convertTaskStatus(status model.TaskStatus) analytics.TaskStatus {
	switch status {
	case model.TaskStatusPending:
		return analytics.TaskStatusPending
	case model.TaskStatusInProgress:
		return analytics.TaskStatusInProgress
	case model.TaskStatusBlocked:
		return analytics.TaskStatusBlocked
	case model.TaskStatusCompleted:
		return analytics.TaskStatusCompleted
	default:
		return analytics.TaskStatusPending
	}
}

// handleTaskProgressSignal processes task progress updates
func handleTaskProgressSignal(ctx workflow.Context, signal model.TaskUpdate, state *model.DailySchedule) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Received task progress signal",
		"taskID", signal.TaskID,
		"newStatus", signal.NewStatus,
		"workerID", signal.UpdatedBy)

	// Log current state
	logger.Info("Current workflow state",
		"taskCount", len(state.Tasks),
		"tasks", fmt.Sprintf("%+v", state.Tasks))

	taskFound := false
	for i, task := range state.Tasks {
		logger.Info("Checking task", "taskID", task.ID, "currentTaskID", signal.TaskID)
		if task.ID == signal.TaskID {
			logger.Info("Found matching task", "taskID", task.ID, "currentStatus", task.Status)
			taskFound = true

			// Update task status
			state.Tasks[i].Status = signal.NewStatus
			if signal.NewStatus == model.TaskStatusBlocked {
				state.Tasks[i].IsBlocked = true
				state.Tasks[i].BlockedAt = signal.UpdateTime
				state.Tasks[i].BlockReason = signal.Notes
			} else {
				state.Tasks[i].IsBlocked = false
			}

			logger.Info("Updated task status",
				"taskID", task.ID,
				"newStatus", signal.NewStatus,
				"isBlocked", state.Tasks[i].IsBlocked)
			break
		}
	}

	if !taskFound {
		logger.Info("Task not found in current state, adding new task", "taskID", signal.TaskID)
		// If task doesn't exist, add it to the list
		newTask := model.Task{
			ID:        signal.TaskID,
			Status:    signal.NewStatus,
			IsBlocked: (signal.NewStatus == model.TaskStatusBlocked),
		}

		if signal.NewStatus == model.TaskStatusBlocked {
			newTask.BlockedAt = signal.UpdateTime
			newTask.BlockReason = signal.Notes
		}
		if signal.NewStatus == model.TaskStatusInProgress {
			startTime := signal.UpdateTime
			newTask.PlannedStartTime = startTime
		}

		state.Tasks = append(state.Tasks, newTask)
		logger.Info("Added new task to state", "taskID", newTask.ID, "status", newTask.Status)
	}

	// Log final state
	logger.Info("Updated workflow state",
		"taskCount", len(state.Tasks),
		"tasks", fmt.Sprintf("%+v", state.Tasks))
}
