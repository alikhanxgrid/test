package scheduling

import (
	"fmt"
	"log"
	"net/http"
	"reference-app-wms-go/app/db"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SchedulingAPI handles HTTP requests for job sites and schedules
type SchedulingAPI struct {
	schedulingService SchedulingService
	logger            *log.Logger
}

// NewSchedulingAPI creates a new SchedulingAPI instance
func NewSchedulingAPI(schedulingService SchedulingService, logger *log.Logger) *SchedulingAPI {
	return &SchedulingAPI{
		schedulingService: schedulingService,
		logger:            logger,
	}
}

// SetupRoutes registers all the scheduling API routes
func (api *SchedulingAPI) SetupRoutes(router *gin.Engine) {
	scheduling := router.Group("/scheduling")
	{
		// Job Site routes
		scheduling.POST("/job-sites", api.CreateJobSite)
		scheduling.GET("/job-sites/:id", api.GetJobSite)

		// Schedule routes
		scheduling.POST("/job-sites/:id/schedules", api.CreateSchedule)
		scheduling.GET("/job-sites/:id/schedules", api.GetSchedules)
		scheduling.GET("/job-sites/:id/schedules/active", api.GetActiveSchedules)
		scheduling.GET("/job-sites/:id/schedules/upcoming", api.GetUpcomingSchedules)
		scheduling.POST("/job-sites/:id/schedules/validate", api.ValidateScheduleOverlap)
		scheduling.GET("/schedules/:id", api.GetSchedule)

		// Task routes
		scheduling.POST("/job-sites/:id/tasks", api.CreateTask)
		scheduling.PUT("/tasks/:id/status", api.UpdateTaskStatus)
		scheduling.GET("/workers/:id/tasks", api.GetWorkerTasks)
		scheduling.GET("/tasks/:id", api.GetTask)

		// Worker attendance routes
		scheduling.POST("/workers/:workerID/attendance", api.RecordWorkerAttendance)
		scheduling.PUT("/workers/:workerID/attendance", api.UpdateWorkerAttendance)

		// Site attendance routes
		scheduling.GET("/job-sites/:id/attendance", api.GetSiteAttendance)
	}
}

// Request/Response structures
type createJobSiteRequest struct {
	Name     string `json:"name" binding:"required"`
	Location string `json:"location" binding:"required"`
}

type createScheduleRequest struct {
	ID        string `json:"id"`
	StartDate string `json:"startDate" binding:"required"`
	EndDate   string `json:"endDate" binding:"required"`
}

type validateScheduleRequest struct {
	StartDate string `json:"startDate" binding:"required"`
	EndDate   string `json:"endDate" binding:"required"`
}

type createTaskRequest struct {
	Name             string    `json:"name" binding:"required"`
	Description      string    `json:"description" binding:"required"`
	ScheduleID       string    `json:"schedule_id" binding:"required"`
	WorkerID         string    `json:"worker_id" binding:"required"`
	PlannedStartTime time.Time `json:"planned_start_time" binding:"required"`
	PlannedEndTime   time.Time `json:"planned_end_time" binding:"required"`
}

type recordAttendanceRequest struct {
	JobSiteID   string    `json:"jobSiteId" binding:"required"`
	CheckInTime time.Time `json:"checkInTime" binding:"required"`
}

type updateAttendanceRequest struct {
	JobSiteID    string    `json:"jobSiteId" binding:"required"`
	Date         string    `json:"date" binding:"required"`
	CheckOutTime time.Time `json:"checkOutTime" binding:"required"`
}

// CreateJobSite handles the creation of a new job site
func (api *SchedulingAPI) CreateJobSite(c *gin.Context) {
	api.logger.Printf("Received CreateJobSite request")

	var req createJobSiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.logger.Printf("Failed to parse CreateJobSite request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	api.logger.Printf("Creating job site with name: %s, location: %v", req.Name, req.Location)

	jobSite, err := api.schedulingService.CreateJobSite(c.Request.Context(), req.Name, req.Location)
	if err != nil {
		api.logger.Printf("Failed to create job site: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create job site: %v", err)})
		return
	}

	api.logger.Printf("Successfully created job site with ID: %s", jobSite.ID)
	c.JSON(http.StatusCreated, jobSite)
}

// GetJobSite handles retrieving a job site by ID
func (api *SchedulingAPI) GetJobSite(c *gin.Context) {
	id := c.Param("id")
	jobSite, err := api.schedulingService.GetJobSite(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Explicitly structure the response
	c.JSON(http.StatusOK, gin.H{
		"id":       jobSite.ID,
		"name":     jobSite.Name,
		"location": jobSite.Location,
	})
}

// CreateSchedule handles the creation of a new schedule
func (api *SchedulingAPI) CreateSchedule(c *gin.Context) {
	jobSiteID := c.Param("id")
	var req createScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.logger.Printf("Failed to parse CreateSchedule request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	api.logger.Printf("Creating schedule for job site %s with ID %s", jobSiteID, req.ID)

	startDate, err := time.Parse(time.RFC3339, req.StartDate)
	if err != nil {
		api.logger.Printf("Invalid start date format: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start date format"})
		return
	}

	endDate, err := time.Parse(time.RFC3339, req.EndDate)
	if err != nil {
		api.logger.Printf("Invalid end date format: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end date format"})
		return
	}

	// Generate schedule ID if not provided
	scheduleID := req.ID
	if scheduleID == "" {
		scheduleID = uuid.New().String()
		api.logger.Printf("Generated new schedule ID: %s", scheduleID)
	}

	// Validate schedule overlap before creating
	if err := api.schedulingService.ValidateScheduleOverlap(c.Request.Context(), jobSiteID, startDate, endDate); err != nil {
		api.logger.Printf("Schedule overlap detected: %v", err)
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	schedule, err := api.schedulingService.CreateSchedule(c.Request.Context(), jobSiteID, scheduleID, startDate, endDate)
	if err != nil {
		api.logger.Printf("Failed to create schedule: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	api.logger.Printf("Successfully created schedule with ID: %s", schedule.ID)
	c.JSON(http.StatusCreated, schedule)
}

// GetSchedules handles retrieving all schedules for a job site
func (api *SchedulingAPI) GetSchedules(c *gin.Context) {
	jobSiteID := c.Param("id")
	schedules, err := api.schedulingService.GetSchedulesByJobSite(c.Request.Context(), jobSiteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, schedules)
}

// GetActiveSchedules handles retrieving active schedules for a job site
func (api *SchedulingAPI) GetActiveSchedules(c *gin.Context) {
	jobSiteID := c.Param("id")
	schedules, err := api.schedulingService.GetActiveSchedules(c.Request.Context(), jobSiteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, schedules)
}

// GetUpcomingSchedules handles retrieving upcoming schedules for a job site
func (api *SchedulingAPI) GetUpcomingSchedules(c *gin.Context) {
	jobSiteID := c.Param("id")
	schedules, err := api.schedulingService.GetUpcomingSchedules(c.Request.Context(), jobSiteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, schedules)
}

// ValidateScheduleOverlap handles schedule overlap validation
func (api *SchedulingAPI) ValidateScheduleOverlap(c *gin.Context) {
	jobSiteID := c.Param("id")
	var req validateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	startDate, err := time.Parse(time.RFC3339, req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start date format"})
		return
	}

	endDate, err := time.Parse(time.RFC3339, req.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end date format"})
		return
	}

	if err := api.schedulingService.ValidateScheduleOverlap(c.Request.Context(), jobSiteID, startDate, endDate); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true})
}

// CreateTask handles the creation of a new task
func (api *SchedulingAPI) CreateTask(c *gin.Context) {
	api.logger.Printf("Received CreateTask request")

	var req createTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.logger.Printf("Failed to parse CreateTask request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	api.logger.Printf("Creating task with name: %s, schedule: %s, worker: %s", req.Name, req.ScheduleID, req.WorkerID)

	now := time.Now()
	task := &db.Task{
		ID:               uuid.New().String(),
		ScheduleID:       req.ScheduleID,
		WorkerID:         req.WorkerID,
		Name:             req.Name,
		Description:      req.Description,
		PlannedStartTime: req.PlannedStartTime,
		PlannedEndTime:   req.PlannedEndTime,
		Status:           string(db.TaskStatusPending),
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := api.schedulingService.CreateTask(c.Request.Context(), task); err != nil {
		api.logger.Printf("Failed to create task: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	api.logger.Printf("Successfully created task with ID: %s", task.ID)
	// Return both task ID and schedule ID in response
	c.JSON(http.StatusCreated, gin.H{
		"id":          task.ID,
		"schedule_id": task.ScheduleID,
	})
}

// UpdateTaskStatus handles updating a task's status
func (api *SchedulingAPI) UpdateTaskStatus(c *gin.Context) {
	taskID := c.Param("id")
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := api.schedulingService.UpdateTaskStatus(c.Request.Context(), taskID, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

// GetWorkerTasks handles retrieving tasks for a worker
func (api *SchedulingAPI) GetWorkerTasks(c *gin.Context) {
	api.logger.Printf("Received GetWorkerTasks request")

	workerID := c.Param("id")
	dateStr := c.Query("date")
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		api.logger.Printf("Failed to parse date: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format"})
		return
	}

	api.logger.Printf("Fetching tasks for worker: %s, date: %s", workerID, dateStr)
	tasks, err := api.schedulingService.GetWorkerTasks(c.Request.Context(), workerID, date)
	if err != nil {
		api.logger.Printf("Failed to get worker tasks: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to get worker tasks: %v", err)})
		return
	}

	api.logger.Printf("Successfully retrieved %d tasks for worker: %s", len(tasks), workerID)
	c.JSON(http.StatusOK, tasks)
}

// GetTask handles retrieving a task by ID
func (api *SchedulingAPI) GetTask(c *gin.Context) {
	api.logger.Printf("Received GetTask request")

	taskID := c.Param("id")
	task, err := api.schedulingService.GetTaskByID(c.Request.Context(), taskID)
	if err != nil {
		api.logger.Printf("Failed to get task: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("failed to get task: %v", err)})
		return
	}

	api.logger.Printf("Successfully retrieved task with ID: %s, schedule ID: %s", task.ID, task.ScheduleID)
	// Explicitly structure the response to ensure schedule_id is included
	c.JSON(http.StatusOK, gin.H{
		"id":                 task.ID,
		"schedule_id":        task.ScheduleID,
		"worker_id":          task.WorkerID,
		"name":               task.Name,
		"description":        task.Description,
		"planned_start_time": task.PlannedStartTime,
		"planned_end_time":   task.PlannedEndTime,
		"status":             task.Status,
		"created_at":         task.CreatedAt,
		"updated_at":         task.UpdatedAt,
	})
}

// GetSchedule handles retrieving a schedule by ID
func (api *SchedulingAPI) GetSchedule(c *gin.Context) {
	scheduleID := c.Param("id")
	api.logger.Printf("Fetching schedule: id=%s", scheduleID)

	schedule, err := api.schedulingService.GetScheduleByID(c.Request.Context(), scheduleID)
	if err != nil {
		api.logger.Printf("Failed to get schedule: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("failed to get schedule: %v", err)})
		return
	}

	api.logger.Printf("Successfully retrieved schedule with ID: %s", schedule.ID)
	c.JSON(http.StatusOK, gin.H{
		"id":          schedule.ID,
		"job_site_id": schedule.JobSiteID,
		"start_date":  schedule.StartDate,
		"end_date":    schedule.EndDate,
		"created_at":  schedule.CreatedAt,
		"updated_at":  schedule.UpdatedAt,
	})
}

// RecordWorkerAttendance handles recording a worker's attendance
func (api *SchedulingAPI) RecordWorkerAttendance(c *gin.Context) {
	api.logger.Printf("Received RecordWorkerAttendance request")

	workerID := c.Param("workerID")
	api.logger.Printf("Worker ID from URL: %s", workerID)

	var req recordAttendanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.logger.Printf("Failed to parse RecordWorkerAttendance request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	api.logger.Printf("Recording attendance for worker ID: %s, job site ID: %s, check-in time: %v",
		workerID, req.JobSiteID, req.CheckInTime)

	err := api.schedulingService.RecordWorkerAttendance(c.Request.Context(), workerID, req.JobSiteID, req.CheckInTime)
	if err != nil {
		api.logger.Printf("Failed to record worker attendance: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	api.logger.Printf("Successfully recorded attendance for worker ID: %s", workerID)
	c.Status(http.StatusOK)
}

// UpdateWorkerAttendance handles updating a worker's attendance record
func (api *SchedulingAPI) UpdateWorkerAttendance(c *gin.Context) {
	workerID := c.Param("workerID")
	log.Printf("[Scheduling API] Updating attendance for worker %s", workerID)

	var req updateAttendanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[Scheduling API] Error binding request for worker %s: %v", workerID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[Scheduling API] Request data for worker %s: jobSiteID=%s, date=%s, checkOutTime=%v",
		workerID, req.JobSiteID, req.Date, req.CheckOutTime)

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		log.Printf("[Scheduling API] Error parsing date for worker %s: %v", workerID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format"})
		return
	}

	err = api.schedulingService.UpdateWorkerAttendance(c.Request.Context(), workerID, req.JobSiteID, date, req.CheckOutTime)
	if err != nil {
		log.Printf("[Scheduling API] Error updating attendance for worker %s: %v", workerID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[Scheduling API] Successfully updated attendance for worker %s", workerID)
	c.Status(http.StatusOK)
}

// GetSiteAttendance handles retrieving attendance records for a job site
func (api *SchedulingAPI) GetSiteAttendance(c *gin.Context) {
	jobSiteID := c.Param("id")
	dateStr := c.Query("date")
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format"})
		return
	}

	attendance, err := api.schedulingService.GetSiteAttendance(c.Request.Context(), jobSiteID, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, attendance)
}
