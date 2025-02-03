# Integration Testing Guide

This directory contains integration tests for the WMS application. The tests are designed to be flexible and support different testing scenarios.

## Test Database

The tests use a dedicated test database (default: `task_management_test_db`). The test setup will:
1. Create the test database if it doesn't exist
2. Clean all existing tables
3. Initialize fresh schema for each test run
4. Clean up after tests (unless `-persist-db` flag is used)

## Running Tests

### Basic Test Run
```bash
go test -v ./test/integration
```
This will:
- Create/clean the test database
- Start an embedded server
- Run all tests
- Clean up the database
- Stop the server

### Running Individual Tests

To run a specific test, you need to include both the test file and any utility files it depends on. For example:

#### Task Management Test
```bash
go test -v ./test/integration/test_utils.go ./test/integration/task_management_test.go
```

You can combine this with any of the test flags:
```bash
# Run with database persistence
go test -v ./test/integration/test_utils.go ./test/integration/task_management_test.go -persist-db

# Run against external server
go test -v ./test/integration/test_utils.go ./test/integration/task_management_test.go -no-server

# Run with both flags
go test -v ./test/integration/test_utils.go ./test/integration/task_management_test.go -persist-db -no-server
```

You can also use the `-run` flag to run a specific test function:
```bash
go test -v ./test/integration/test_utils.go ./test/integration/task_management_test.go -run TestTaskManagementAndWorkerState
```

### Test Flags

#### Persist Database (`-persist-db`)
```bash
go test -v ./test/integration -persist-db
```
This flag prevents the test from cleaning up the database after the test run. Useful for:
- Debugging test failures
- Examining data state
- Manual verification

#### External Server (`-no-server`)
```bash
go test -v ./test/integration -no-server
```
This flag prevents the test from starting an embedded server. Use this when:
- Running against an existing server instance
- Debugging server issues
- Testing with custom server configuration

#### Combining Flags
```bash
go test -v ./test/integration -persist-db -no-server
```
This combination:
- Won't start a server (assumes external server)
- Won't clean up the database
- Perfect for debugging both server and data issues

### Running with External Server

1. Start the server with test database:
```bash
go run cmd/wms/main.go -db-name=task_management_test_db
```

2. Run tests with `-no-server` flag:
```bash
go test -v ./test/integration -no-server
```

## Test Database Configuration

Default settings:
- Host: localhost
- Port: 5432
- User: abdullahshah
- Database: task_management_test_db

These can be modified by updating the constants in `task_management_test.go`. 