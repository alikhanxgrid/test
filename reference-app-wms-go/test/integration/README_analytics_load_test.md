# Analytics Load Test Documentation

## Overview
The Analytics Load Test is a comprehensive integration test that simulates a real-world workday environment with multiple workers, job sites, and tasks. It validates the system's ability to handle concurrent operations, maintain data consistency, and properly track worker activities throughout their workday.

## Test Configuration
The test uses a `LoadTestConfig` structure that defines key parameters:

```go
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
```

Default configuration values are provided for standard load testing scenarios.

## Test Flow

### 1. Setup Phase
1. Creates a test database with required schema
2. Starts a test server instance
3. Initializes the simulation context with configuration
4. Creates job sites with unique locations
5. Generates worker IDs for all simulated workers

### 2. Daily Simulation
For each simulated day, the test:

1. **Schedule Creation**
   - Creates multiple schedules for each job site
   - Number of schedules varies randomly between MinSchedulesPerSite and MaxSchedulesPerSite
   - Schedules are created with non-overlapping time slots

2. **Task Generation**
   - Assigns workers to specific job sites (worker_index % num_job_sites)
   - Creates tasks for each worker at their assigned job site
   - Tasks are linked to schedules at the job site
   - Each task has planned start and end times

3. **Worker Day Simulation**
   - Simulates concurrent worker check-ins (controlled by ConcurrentCheckins)
   - For each worker:
     - Checks in at their assigned job site
     - Processes their assigned tasks
     - Takes random breaks (30% chance before each task)
     - Handles random task blockages (based on BlockageRate)
     - Completes tasks
     - Checks out at end of day

## Operation Tracking
The test tracks all operations in chronological order:
```go
type taskOperation struct {
    TaskID    string
    WorkerID  model.WorkerID
    Operation string        // check-in, task-update, break, check-out
    Time      time.Time
    Status    model.TaskStatus
    IsBlocked bool
    OnBreak   bool
    Location  string
}
```

## Validations

### 1. Operational State Validation
The test performs comprehensive validation of the operational state:

1. **Worker Operation Sequence**
   - Verifies check-in is first operation
   - Verifies check-out is last operation
   - Validates chronological order of operations

2. **Location Consistency**
   - Ensures all tasks for a worker are at their checked-in job site
   - Validates location tracking through check-in/check-out

3. **Task State Transitions**
   - Validates legal state transitions:
     * PENDING → IN_PROGRESS
     * IN_PROGRESS → BLOCKED or COMPLETED
     * BLOCKED → IN_PROGRESS
   - Ensures no invalid state transitions

4. **Break Management**
   - Verifies no task updates during breaks
   - Validates break start/end sequences
   - Ensures no overlapping breaks

5. **Task Completion**
   - Verifies all tasks reach COMPLETED state
   - Validates task completion timestamps

### 2. Analytics Data Validation
After operational validation, the test verifies analytics data:

1. **Worker State**
   - Validates final worker state
   - Verifies session activity status
   - Checks current site information

2. **Task Analytics**
   - Verifies task completion rates
   - Validates break patterns
   - Checks blockage statistics

## Error Handling
The test includes robust error handling:
- Retries for eventual consistency
- Proper cleanup of resources
- Detailed error logging
- State verification at each step

## Usage
To run the analytics load test:

```bash
go test -v -run TestAnalyticsLoad
```

Options:
- `-persist-db`: Keeps the test database for debugging
- `-no-server`: Skips server startup (assumes external server)

## Common Issues and Debugging
1. **Location Validation Failures**
   - Check worker-to-site assignment logic
   - Verify task creation uses correct job site
   - Ensure location consistency in check-in/out

2. **Task State Inconsistencies**
   - Review state transition logic
   - Check for race conditions in concurrent operations
   - Verify task update signal handling

3. **Timing Issues**
   - Adjust concurrent check-in limits
   - Review schedule overlap validation
   - Check for proper operation sequencing

## Future Enhancements
1. Add validation for:
   - Task completion rates
   - Break patterns
   - Blockage statistics
   - Worker productivity metrics

2. Enhance simulation with:
   - Variable task durations
   - Dynamic worker availability
   - Complex break patterns
   - Multi-site worker scenarios 