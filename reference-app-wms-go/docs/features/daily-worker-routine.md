# Daily Worker Routine Management

## Overview
The Daily Worker Routine (DWR) is the core Workflow that manages a worker's 
entire day, from check-in to check-out. It orchestrates task assignments, 
break periods, and status tracking while ensuring compliance with labor 
regulations.

## Workflow Lifecycle

### 1. Check-In
When a worker starts their day:
```http
POST /check-in
{
    "workerId": "worker123",
    "checkInTime": "2024-03-21T09:00:00Z",
    "location": "Site A"
}
```

This initiates the DWR Workflow, which:
- Records the check-in time and location
- Initializes the daily schedule
- Retrieves assigned tasks
- Begins status tracking

### 2. Task Management
Throughout the day, the DWR Workflow:
- Assigns tasks based on priority and location
- Tracks task progress
- Handles task completion
- Manages task reassignment if needed

### 3. Break Tracking
The Workflow monitors break periods:
- Scheduled breaks (lunch, rest periods)
- Unscheduled breaks
- Compliance with required break timing
- Break duration tracking

### 4. Check-Out
At day's end:
```http
POST /check-out
{
    "workerId": "worker123",
    "checkOutTime": "2024-03-21T17:00:00Z"
}
```

The Workflow:
- Records check-out time
- Finalizes incomplete tasks
- Generates daily summary
- Archives the routine data 