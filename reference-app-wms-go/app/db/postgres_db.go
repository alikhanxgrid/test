package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
)

type PostgresDB struct {
	*sqlx.DB
}

// NewPostgresDB initializes a new PostgreSQL DB connection
func NewPostgresDB(db *sqlx.DB) *PostgresDB {
	return &PostgresDB{DB: db}
}

// AddJobSite adds a new job site to the database
func (p *PostgresDB) AddJobSite(ctx context.Context, site *JobSite) error {
	query := `INSERT INTO job_sites (id, name, location, created_at, updated_at)
	          VALUES (:id, :name, :location, :created_at, :updated_at)`
	_, err := p.NamedExecContext(ctx, query, site)
	return err
}

// GetJobSite retrieves a job site by ID
func (p *PostgresDB) GetJobSite(ctx context.Context, id string) (*JobSite, error) {
	var site JobSite
	query := `SELECT id, name, location, created_at, updated_at FROM job_sites WHERE id = $1`
	err := p.GetContext(ctx, &site, query, id)
	if err != nil {
		return nil, err
	}
	return &site, nil
}

// AddSchedule adds a new schedule to the database
func (p *PostgresDB) AddSchedule(ctx context.Context, schedule *Schedule) error {
	query := `INSERT INTO schedules (id, job_site_id, start_date, end_date, created_at, updated_at)
	          VALUES (:id, :job_site_id, :start_date, :end_date, :created_at, :updated_at)`
	_, err := p.NamedExecContext(ctx, query, schedule)
	return err
}

// GetSchedulesByJobSite retrieves schedules for a specific job site
func (p *PostgresDB) GetSchedulesByJobSite(ctx context.Context, jobSiteID string) ([]Schedule, error) {
	var schedules []Schedule
	query := `SELECT id, job_site_id, start_date, end_date, created_at, updated_at 
	          FROM schedules WHERE job_site_id = $1`
	err := p.SelectContext(ctx, &schedules, query, jobSiteID)
	if err != nil {
		return nil, err
	}
	return schedules, nil
}

// GetTasksByWorker retrieves tasks assigned to a worker for a specific date
func (p *PostgresDB) GetTasksByWorker(ctx context.Context, workerID string, date time.Time) ([]Task, error) {
	var tasks []Task
	query := `
		SELECT t.* 
		FROM tasks t
		INNER JOIN schedules s ON t.schedule_id = s.id
		WHERE t.worker_id = $1 
		AND DATE(t.planned_start_time) = DATE($2)
		ORDER BY t.planned_start_time ASC`

	err := p.SelectContext(ctx, &tasks, query, workerID, date)
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

// UpdateTaskStatus updates the status of a task
func (p *PostgresDB) UpdateTaskStatus(ctx context.Context, taskID string, status string) error {
	query := `
		UPDATE tasks 
		SET status = $1, updated_at = NOW()
		WHERE id = $2`

	_, err := p.ExecContext(ctx, query, status, taskID)
	return err
}

// AddTask adds a new task to the database
func (p *PostgresDB) AddTask(ctx context.Context, task *Task) error {
	query := `
		INSERT INTO tasks (
			id, schedule_id, worker_id, name, description, 
			planned_start_time, planned_end_time, status, created_at, updated_at
		) VALUES (
			:id, :schedule_id, :worker_id, :name, :description,
			:planned_start_time, :planned_end_time, :status, :created_at, :updated_at
		)`
	_, err := p.NamedExecContext(ctx, query, task)
	return err
}

// GetTaskByID retrieves a task by ID with its associated schedule and job site information
func (p *PostgresDB) GetTaskByID(ctx context.Context, taskID string) (*Task, error) {
	var task Task
	query := `
		SELECT t.id, t.schedule_id, t.worker_id, t.name, t.description,
			   t.planned_start_time, t.planned_end_time, t.status,
			   t.created_at, t.updated_at
		FROM tasks t
		WHERE t.id = $1`

	err := p.GetContext(ctx, &task, query, taskID)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// Close closes the database connection
func (p *PostgresDB) Close() error {
	return p.DB.Close()
}

// GetScheduleByID retrieves a schedule by ID
func (p *PostgresDB) GetScheduleByID(ctx context.Context, scheduleID string) (*Schedule, error) {
	var schedule Schedule
	query := `SELECT id, job_site_id, start_date, end_date, created_at, updated_at 
			  FROM schedules WHERE id = $1`
	err := p.GetContext(ctx, &schedule, query, scheduleID)
	if err != nil {
		return nil, err
	}
	return &schedule, nil
}

// AddWorkerAttendance adds a new worker attendance record
func (p *PostgresDB) AddWorkerAttendance(ctx context.Context, attendance *WorkerAttendance) error {
	// Ensure we're using the date part for the composite key
	attendanceDate := attendance.Date.UTC().Truncate(24 * time.Hour)

	query := `
		INSERT INTO worker_attendance (
			worker_id, job_site_id, date, check_in_time, created_at, updated_at
		) VALUES (
			:worker_id, :job_site_id, :date, :check_in_time, :created_at, :updated_at
		) ON CONFLICT (worker_id, date) DO UPDATE SET
			check_in_time = EXCLUDED.check_in_time,
			job_site_id = EXCLUDED.job_site_id,
			updated_at = EXCLUDED.updated_at`

	// Update the date to ensure we're using the date part only
	attendance.Date = attendanceDate

	result, err := p.NamedExecContext(ctx, query, attendance)
	if err != nil {
		return fmt.Errorf("database insert failed: worker_id=%s, job_site_id=%s, date=%s, error: %w",
			attendance.WorkerID, attendance.JobSiteID, attendance.Date.Format("2006-01-02"), err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("no rows affected when recording attendance for worker %s on date %s",
			attendance.WorkerID, attendance.Date.Format("2006-01-02"))
	}

	return nil
}

// UpdateWorkerAttendance updates the check-out time for a worker's attendance record
func (p *PostgresDB) UpdateWorkerAttendance(ctx context.Context, workerID string, jobSiteID string, date time.Time, checkOutTime time.Time) error {
	// Ensure we're using the date part for the composite key
	attendanceDate := date.UTC().Truncate(24 * time.Hour)

	log.Printf("[DB] Updating attendance record for worker %s at job site %s for date %s",
		workerID, jobSiteID, attendanceDate.Format("2006-01-02"))

	query := `
		UPDATE worker_attendance 
		SET check_out_time = $1, updated_at = NOW()
		WHERE worker_id = $2 AND job_site_id = $3 AND date = $4`

	result, err := p.ExecContext(ctx, query, checkOutTime, workerID, jobSiteID, attendanceDate)
	if err != nil {
		log.Printf("[DB] Error updating attendance record: %v", err)
		return fmt.Errorf("database update failed: worker_id=%s, job_site_id=%s, date=%s, error: %w",
			workerID, jobSiteID, attendanceDate.Format("2006-01-02"), err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		log.Printf("[DB] Error checking affected rows: %v", err)
		return fmt.Errorf("error checking affected rows: %w", err)
	}

	if rows == 0 {
		log.Printf("[DB] No attendance record found to update for worker %s at job site %s on date %s",
			workerID, jobSiteID, attendanceDate.Format("2006-01-02"))
		return fmt.Errorf("no attendance record found for worker_id=%s, job_site_id=%s, date=%s",
			workerID, jobSiteID, attendanceDate.Format("2006-01-02"))
	}

	log.Printf("[DB] Successfully updated attendance record for worker %s on date %s",
		workerID, attendanceDate.Format("2006-01-02"))
	return nil
}

// GetWorkerAttendanceByDate retrieves attendance records for a job site on a specific date
func (p *PostgresDB) GetWorkerAttendanceByDate(ctx context.Context, jobSiteID string, date time.Time) ([]WorkerAttendance, error) {
	query := `
		SELECT * FROM worker_attendance
		WHERE job_site_id = $1 AND DATE(date) = DATE($2)`

	var attendance []WorkerAttendance
	err := p.SelectContext(ctx, &attendance, query, jobSiteID, date)
	if err != nil {
		return nil, err
	}
	return attendance, nil
}

// GetWorkerAttendanceByDateRange retrieves attendance records for a job site within a date range
func (p *PostgresDB) GetWorkerAttendanceByDateRange(ctx context.Context, jobSiteID string, startDate, endDate time.Time) ([]WorkerAttendance, error) {
	query := `
		SELECT * FROM worker_attendance
		WHERE job_site_id = $1 
		AND DATE(date) >= DATE($2)
		AND DATE(date) <= DATE($3)
		ORDER BY date ASC`

	var attendance []WorkerAttendance
	err := p.SelectContext(ctx, &attendance, query, jobSiteID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	return attendance, nil
}

// Select executes a query that returns multiple rows
func (p *PostgresDB) Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return p.SelectContext(ctx, dest, query, args...)
}

// Get executes a query that returns a single row
func (p *PostgresDB) Get(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return p.GetContext(ctx, dest, query, args...)
}
