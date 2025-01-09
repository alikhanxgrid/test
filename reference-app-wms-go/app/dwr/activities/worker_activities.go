package activities

import (
	"context"
	"fmt"
	"log"
	"time"

	"reference-app-wms-go/app/db"
	"reference-app-wms-go/app/dwr/model"
	"reference-app-wms-go/app/scheduling"

	"go.temporal.io/sdk/activity"
)

// Activities interface defines all activities used in the DWR workflow
type Activities interface {
	RecordCheckInActivity(ctx context.Context, params model.CheckInRequest) (model.DailySchedule, error)
	RetrieveDailyTasksActivity(ctx context.Context, workerID model.WorkerID) ([]model.Task, error)
	UpdateTaskProgressActivity(ctx context.Context, update model.TaskUpdate) error
	RecordCheckOutActivity(ctx context.Context, state model.CheckOutRequest) error
}

// ActivitiesImpl implements the Activities interface
type ActivitiesImpl struct {
	schedulingService scheduling.SchedulingService
	logger            *log.Logger
}

// NewActivities creates a new instance of ActivitiesImpl
func NewActivities(schedulingService scheduling.SchedulingService) Activities {
	logger := log.New(log.Default().Writer(), "[Activities] ", log.LstdFlags|log.Lshortfile)
	logger.Printf("Initializing activities with Scheduling API client")
	return &ActivitiesImpl{
		schedulingService: schedulingService,
		logger:            logger,
	}
}

// RecordCheckInActivity records a worker's check-in and retrieves their daily schedule
func (a *ActivitiesImpl) RecordCheckInActivity(ctx context.Context, params model.CheckInRequest) (model.DailySchedule, error) {
	logger := activity.GetLogger(ctx)
	a.logger.Printf("Recording worker check-in via Scheduling API: worker=%s, jobSiteId=%s", params.WorkerID, params.JobSiteID)
	logger.Info("Recording worker check-in", "workerID", params.WorkerID)

	// Record attendance directly with job site ID
	err := a.schedulingService.RecordWorkerAttendance(ctx, string(params.WorkerID), params.JobSiteID, params.CheckInTime)
	if err != nil {
		return model.DailySchedule{}, fmt.Errorf("failed to record worker attendance: %w", err)
	}

	// Create initial daily schedule
	schedule := model.DailySchedule{
		WorkerID: params.WorkerID,
		Date:     params.CheckInTime,
		CheckIn:  params.CheckInTime,
	}

	return schedule, nil
}

// RetrieveDailyTasksActivity fetches tasks for a worker for the current day
func (a *ActivitiesImpl) RetrieveDailyTasksActivity(ctx context.Context, workerID model.WorkerID) ([]model.Task, error) {
	logger := activity.GetLogger(ctx)
	a.logger.Printf("Retrieving daily tasks via Scheduling API: worker=%s", workerID)
	logger.Info("Retrieving daily tasks", "workerID", workerID)

	// Get today's date in UTC
	today := time.Now().UTC()

	// Fetch tasks from the scheduling service
	dbTasks, err := a.schedulingService.GetWorkerTasks(ctx, string(workerID), today)
	if err != nil {
		a.logger.Printf("Failed to retrieve tasks from Scheduling API: worker=%s, error=%v", workerID, err)
		return nil, fmt.Errorf("failed to retrieve tasks: %w", err)
	}

	// Convert db.Task to model.Task
	tasks := make([]model.Task, len(dbTasks))
	for i, dbTask := range dbTasks {
		isBlocked := dbTask.Status == string(db.TaskStatusBlocked)
		tasks[i] = model.Task{
			ID:               dbTask.ID,
			Name:             dbTask.Name,
			Description:      dbTask.Description,
			Status:           model.TaskStatus(dbTask.Status),
			PlannedStartTime: dbTask.PlannedStartTime,
			PlannedEndTime:   dbTask.PlannedEndTime,
			IsBlocked:        isBlocked,
		}
	}

	a.logger.Printf("Successfully retrieved %d tasks for worker=%s", len(tasks), workerID)
	return tasks, nil
}

// UpdateTaskProgressActivity handles updates to task status and progress
func (a *ActivitiesImpl) UpdateTaskProgressActivity(ctx context.Context, update model.TaskUpdate) error {
	logger := activity.GetLogger(ctx)
	a.logger.Printf("Updating task progress via Scheduling API: task=%s, status=%s, worker=%s",
		update.TaskID, update.NewStatus, update.UpdatedBy)
	logger.Info("Updating task progress",
		"taskID", update.TaskID,
		"newStatus", update.NewStatus,
		"updatedBy", update.UpdatedBy)

	// Update task status in the scheduling service
	err := a.schedulingService.UpdateTaskStatus(ctx, update.TaskID, string(update.NewStatus))
	if err != nil {
		a.logger.Printf("Failed to update task status in Scheduling API: task=%s, error=%v", update.TaskID, err)
		return fmt.Errorf("failed to update task status: %w", err)
	}

	a.logger.Printf("Successfully updated task status: task=%s, status=%s", update.TaskID, update.NewStatus)
	return nil
}

// RecordCheckOutActivity processes a worker's check-out
func (a *ActivitiesImpl) RecordCheckOutActivity(ctx context.Context, req model.CheckOutRequest) error {
	logger := activity.GetLogger(ctx)
	a.logger.Printf("Recording worker check-out via Scheduling API: worker=%s, jobSiteId=%s", req.WorkerID, req.JobSiteID)
	logger.Info("Recording worker check-out",
		"workerID", req.WorkerID,
		"checkOutTime", req.CheckOutTime)

	// Update attendance record with check-out time
	err := a.schedulingService.UpdateWorkerAttendance(ctx, string(req.WorkerID), req.JobSiteID, req.CheckOutTime, req.CheckOutTime)
	if err != nil {
		return fmt.Errorf("failed to update worker attendance: %w", err)
	}

	return nil
}
