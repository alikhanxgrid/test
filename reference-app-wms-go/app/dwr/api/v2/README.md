# DWR API v2 Implementation

This directory contains the v2 implementation of the Daily Worker Routine (DWR) API. The API is defined using OpenAPI 3.0 specification and uses generated code for type safety and consistency.

## Directory Structure

```
api/v2/
├── openapi/           # Generated code from OpenAPI spec
│   ├── client.gen.go  # Generated client code
│   ├── server.gen.go  # Generated server interfaces
│   └── models.gen.go  # Generated model types
    └── job_execution.yaml # OpenAPI specification files
├── impl/             # Implementation of the generated server interfaces
│   └── job_execution_server_impl.go
```

## Code Generation

The server, client, and models are generated from the OpenAPI specification using `oapi-codegen`. The following make command is available:

```bash
# Generate all code (server, client, models)
make generate-api
```

The code generation is configured using YAML config files:
- `models.config.yaml` - Configuration for model generation
- `server.config.yaml` - Configuration for server interface generation
- `client.config.yaml` - Configuration for client generation

Example config file:
```yaml
package: openapi
generate:
  models: true  # for models.config.yaml
  server: true  # for server.config.yaml
  client: true  # for client.config.yaml
output: models.gen.go  # Change based on type
```

## Implementation Details

- The API uses generated interfaces from the OpenAPI specification
- Implementation is provided in the `impl` package
- All models are strongly typed and generated from the spec
- Client code provides type-safe API calls with response handling
- Server interfaces ensure implementation matches the spec

## Usage

### Server Side

1. Implement the generated server interfaces in `impl` package
2. Register the implementation with the server:

```go
import (
    "reference-app-wms-go/app/dwr/api/v2/impl"
    "reference-app-wms-go/app/dwr/api/v2/openapi"
)

// Create implementation
jobExecutionAPI := impl.NewJobExecutionAPIV2()

// Register with server
server := openapi.NewServer(jobExecutionAPI)
```

### Client Side

```go
import "reference-app-wms-go/app/dwr/api/v2/openapi"

// Create client
client, err := openapi.NewClientWithResponses("http://localhost:8080")

// Make API calls
resp, err := client.WorkerCheckInWithResponse(context.Background(), checkInRequest)
```

## Testing

The generated client is used in integration tests:
- `test/integration/task_management_test.go`
- `test/integration/dwr_test.go`
- `test/integration/monitoring_test.go`

Create a test client using the helper function in `test_utils.go`:

```go
client := createJobExecutionClient(t)
```

## Makefile Configuration

Add these targets to your Makefile:

```makefile
.PHONY: generate-api clean

# Generate all API code
generate-api:
	@echo "Generating OpenAPI code..."
	@go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config models.config.yaml api.yaml
	@go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config server.config.yaml api.yaml
	@go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config client.config.yaml api.yaml

# Clean generated files
clean:
	@rm -f models.gen.go server.gen.go client.gen.go
```

## Migration Notes

When migrating from v1 to v2:
1. Update import paths to use the v2 package
2. Replace direct HTTP calls with client methods
3. Use generated model types instead of custom types
4. Implement new server interfaces in the impl package

## Best Practices

1. Don't modify generated code directly
2. Keep implementation separate in the impl package
3. Use the generated client for type-safe API calls
4. Maintain backwards compatibility when updating the spec
5. Run code generation after spec changes 