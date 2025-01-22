# Task Management

## Overview
The Task Management system handles the creation, assignment, and tracking 
of work tasks. It ensures efficient distribution of work and maintains 
clear visibility of task progress.

## Task Lifecycle

### 1. Task Creation
Tasks can be created through the API:
```http
POST /task
{
    "description": "Install electrical panel",
    "priority": 1,
    "estimatedDuration": "2h30m",
    "location": "Site A",
    "requiredSkills": ["electrical", "installation"]
}
```

### 2. Task Assignment
Tasks are assigned to workers based on:
- Worker availability
- Required skills
- Location proximity
- Task priority
- Estimated duration

### 3. Task Progress
Workers update task status through signals:
```http
PUT /task/{taskId}/status
{
    "status": "in-progress",
    "progress": 75,
    "notes": "Completing final connections"
}
```

### 4. Task Completion
When a task is completed:
- Final status is recorded
- Actual duration is calculated
- Task history is archived
- Worker becomes available for new tasks 