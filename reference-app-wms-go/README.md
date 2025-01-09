# Temporal Reference Application: Workflow Management System (Go)

The Workflow Management System (WMS) is a reference application that demonstrates 
how to implement a worker management and task tracking system using Temporal Workflows. 
This application can be run locally on your development machine or deployed to 
production environments. The required Temporal Service can be run locally or be 
provided by Temporal Cloud.

## Features

- **Worker Session Management**: Track worker check-ins and check-outs with location data
- **Task Management**: Assign, track, and update task status throughout the day
- **Break Management**: Monitor worker breaks and total break duration
- **Progress Monitoring**: Real-time monitoring of task progress and blockages
- **Task Dependencies**: Handle task dependencies and blockages with proper state management
- **Eventual Consistency**: Robust handling of workflow state updates with retry mechanisms

## Quickstart
We recommend starting with the [documentation](docs/README.md), which explains 
the features and design of the application. It also provides detailed instructions 
for deployment and operation in various environments.

For a quick local setup, follow these steps from the root directory of your clone:

### Required Software
You will need:
- [Go](https://go.dev/) 1.21 or later
- [Temporal CLI](https://docs.temporal.io/cli#install)
- [PostgreSQL](https://www.postgresql.org/) 13 or later

### Start the Temporal Service
```bash
temporal server start-dev
```

The Temporal Service manages the state of all Workflow executions, including 
worker check-ins, task assignments, and daily routines.

### Initialize the Database
```bash
psql -U your_username -d postgres -f app/db/schema.sql
```

This creates the necessary database tables for job sites, schedules, tasks, and worker records.

### Start the Workers
```bash
go run cmd/wms/worker/main.go
```

This starts the Temporal Workers that execute the Workflow and Activity functions 
managing daily worker routines and task tracking.

### Start the API Server
```bash
go run cmd/wms/main.go
```

The API server provides REST endpoints for:
- Worker check-in/check-out
- Task progress updates
- Break management
- Task blockage reporting
- Worker status queries

### Run Integration tests
```bash
go test -v ./test/integration
```

## API Endpoints

### Worker Management
- `POST /check-in`: Worker check-in with location
- `POST /check-out`: Worker check-out with completion notes
- `POST /break`: Start or end worker breaks
- `GET /workers/:workerID/tasks`: Get worker's current tasks

### Task Management
- `POST /task-progress`: Update task status and progress
- `GET /workers/:workerID/tasks/:taskID/blockage`: Get task blockage status

### Scheduling
- `POST /scheduling/job-sites`: Create new job sites
- `POST /scheduling/job-sites/:id/schedules`: Create work schedules
- `POST /scheduling/job-sites/:id/tasks`: Create new tasks

## Repository Structure

| Directory                                             | Description                                                       |
| ----------------------------------------------------- | ----------------------------------------------------------------- |
| <code><a href="app/">app/</a></code>                  | Application code including Workflows and Activities                |
| <code><a href="app/dwr/">app/dwr/</a></code>          | Daily Worker Routine implementation                               |
| <code><a href="app/monitoring/">app/monitoring/</a></code> | Progress monitoring and status tracking                           |
| <code><a href="app/scheduling/">app/scheduling/</a></code> | Job site and schedule management                                 |
| <code><a href="cmd/">cmd/</a></code>                  | Command-line tools and entry points                               |
| <code><a href="test/">test/</a></code>                | Integration and unit tests                                        |
| <code><a href="docs/">docs/</a></code>                | Documentation                                                     |

See the [technical description](docs/technical-description.md) for detailed implementation information. 
