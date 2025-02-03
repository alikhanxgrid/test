# Technical Description

This document provides a detailed technical description of the Workflow Management System (WMS), including its implementation details, workflow processes, and code organization.

## System Architecture

The WMS is built using a microservices architecture with the following main components:

1. **API Server**: Handles HTTP requests and interfaces with Temporal
2. **Temporal Workers**: Execute workflow and activity code
3. **PostgreSQL Database**: Stores persistent data about job sites, schedules, and tasks
4. **Temporal Service**: Manages workflow state and orchestration

## Daily Worker Routine Workflow

The core of the system is the `DailyWorkerRoutine` workflow, which manages a worker's entire day from check-in to check-out.

### Workflow State

The workflow maintains state using the `DailyWorkerRoutineState` struct:
```go
type DailyWorkerRoutineState struct {
    Schedule     model.DailySchedule
    WorkerState  model.WorkerState
    IsCheckedOut bool
}
```

### Signal Channels

The workflow listens to three main signal channels:
- `task-progress-signal`: For task status updates
- `break-signal`: For break start/end events
- `check-out`: For worker check-out

### Query Handlers

Two query handlers provide real-time state information:
- `getCurrentTasks`: Returns current tasks and session status
- `getTaskBlockage`: Returns blockage information for specific tasks

## Typical Day Workflow

Here's how the system handles a typical worker's day:

1. **Check-in Process**:
   - Worker sends check-in request with location
   - System starts `DailyWorkerRoutine` workflow
   - `RetrieveDailyTasksActivity` fetches assigned tasks
   - Workflow initializes state with tasks and worker info

2. **Task Management**:
   - Worker starts a task → Task status updated to "IN_PROGRESS"
   - Task completion → Status updated to "COMPLETED"
   - Task blockage → Status updated to "BLOCKED" with reason
   - All updates handled through task-progress-signal

3. **Break Management**:
   - Break start → Updates worker status to "ON_BREAK"
   - Break end → Updates status to "ON_DUTY"
   - System tracks total break duration

4. **Check-out Process**:
   - Worker sends check-out request
   - Workflow updates state and marks session as inactive
   - System records final status of all tasks

## Task Status Management

Tasks can be in one of four states:
- `PENDING`: Initial state
- `IN_PROGRESS`: Worker actively working
- `BLOCKED`: Task blocked with reason
- `COMPLETED`: Task finished

The system ensures eventual consistency of task states through:
- Workflow state management
- Database updates
- Retry mechanisms in status checks

## Code Flow Example

Here's a detailed flow for a task status update:

1. **API Layer** (`job_execution.go`):
   ```go
   // Receive HTTP request
   // Validate input
   // Send signal to workflow
   ```

2. **Workflow** (`daily_worker_routine.go`):
   ```go
   // Receive signal
   // Update local state
   // Execute activity to update DB
   ```

3. **Activity** (`worker_activities.go`):
   ```go
   // Update database
   // Handle any errors
   // Return result
   ```

## Error Handling and Recovery

The system implements several reliability patterns:

1. **Retry Mechanisms**:
   - Activity retries with backoff
   - Query retries for eventual consistency
   - Signal delivery guarantees

2. **State Verification**:
   - Status checks with timeouts
   - State validation before updates
   - Logging for debugging

3. **Transaction Management**:
   - Database transaction integrity
   - Workflow state consistency
   - Signal handling order preservation

## Integration Testing

The system includes comprehensive integration tests that verify:

1. **Worker Lifecycle**:
   - Check-in/check-out flow
   - Task status updates
   - Break management

2. **Task Management**:
   - Task creation and assignment
   - Status updates and verification
   - Blockage handling

3. **State Consistency**:
   - Workflow state queries
   - Database state verification
   - Signal handling verification

## API Endpoints Detail

### Worker Management

#### Check-in
```http
POST /check-in
{
    "workerId": "string",
    "checkInTime": "timestamp",
    "location": "string"
}
```

#### Check-out
```http
POST /check-out
{
    "workerId": "string",
    "checkOutTime": "timestamp",
    "location": "string",
    "notes": "string"
}
```

### Task Management

#### Update Task Progress
```http
POST /task-progress
{
    "taskId": "string",
    "newStatus": "string",
    "updateTime": "timestamp",
    "updatedBy": "string",
    "notes": "string"
}
```

## Best Practices

1. **State Management**:
   - Keep workflow state minimal
   - Use queries for real-time status
   - Implement proper signal handling

2. **Error Handling**:
   - Implement proper retries
   - Log relevant information
   - Handle edge cases

3. **Testing**:
   - Write comprehensive integration tests
   - Test eventual consistency
   - Verify state transitions