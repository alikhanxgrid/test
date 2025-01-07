package integration

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"

	"reference-app-wms-go/app/server"
)

const (
	testDBName = "task_management_test_db"
	testDBHost = "localhost"
	testDBPort = "5432"
	testDBUser = "abdullahshah"
	serverPort = 8081
)

var (
	persistDB bool
	noServer  bool
)

func init() {
	flag.BoolVar(&persistDB, "persist-db", false, "If true, do not clean up test database after test completion")
	flag.BoolVar(&noServer, "no-server", false, "If true, do not start the server within the test (assumes server is running externally)")
}

// SetupTestDB creates and initializes a test database
func SetupTestDB(t *testing.T) (*sqlx.DB, func()) {
	// Connect to test database
	testConnStr := fmt.Sprintf("postgres://%s@%s:%s/%s?sslmode=disable",
		testDBUser, testDBHost, testDBPort, testDBName)
	testDB, err := sqlx.Connect("postgres", testConnStr)
	if err != nil {
		// If database doesn't exist, create it
		if strings.Contains(err.Error(), "does not exist") {
			adminConnStr := fmt.Sprintf("postgres://%s@%s:%s/postgres?sslmode=disable",
				testDBUser, testDBHost, testDBPort)
			adminDB, err := sql.Open("postgres", adminConnStr)
			require.NoError(t, err)
			defer adminDB.Close()

			_, err = adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s", testDBName))
			require.NoError(t, err)

			// Try connecting again
			testDB, err = sqlx.Connect("postgres", testConnStr)
			require.NoError(t, err)
		} else {
			require.NoError(t, err)
		}
	}

	// Clean up existing tables if they exist
	_, err = testDB.Exec(`
		DO $$ DECLARE
			r RECORD;
		BEGIN
			FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = current_schema()) LOOP
				EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.tablename) || ' CASCADE';
			END LOOP;
		END $$;
	`)
	require.NoError(t, err)

	// Read and execute schema.sql
	schemaPath := "../../app/db/schema.sql"
	schemaSQL, err := os.ReadFile(schemaPath)
	require.NoError(t, err)

	_, err = testDB.Exec(string(schemaSQL))
	require.NoError(t, err)

	// Return cleanup function
	cleanup := func() {
		if !persistDB {
			// Clean up all tables
			_, err := testDB.Exec(`
				DO $$ DECLARE
					r RECORD;
				BEGIN
					FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = current_schema()) LOOP
						EXECUTE 'TRUNCATE TABLE ' || quote_ident(r.tablename) || ' CASCADE';
					END LOOP;
				END $$;
			`)
			if err != nil {
				t.Logf("Failed to clean up tables: %v", err)
			}
		} else {
			t.Logf("Persisting test database %q for debugging", testDBName)
		}
		testDB.Close()
	}

	return testDB, cleanup
}

// StartTestServer creates and starts a test server instance
func StartTestServer(t *testing.T, dbConn *sqlx.DB) (*server.Server, func()) {
	// Create temporal client
	temporalClient, err := client.Dial(client.Options{})
	require.NoError(t, err)

	// Create and start server
	srv := server.NewServer(dbConn, temporalClient)

	// Start server in a goroutine
	go func() {
		if err := srv.Start(fmt.Sprintf(":%d", serverPort)); err != nil {
			t.Logf("Server error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Return cleanup function
	cleanup := func() {
		if err := srv.Stop(); err != nil {
			t.Logf("Error during server cleanup: %v", err)
		}
		temporalClient.Close()
	}

	return srv, cleanup
}

// Common test response types
type jobSiteResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Location  string    `json:"location"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type scheduleResponse struct {
	ID        string    `json:"id"`
	JobSiteID string    `json:"job_site_id"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type workflowResponse struct {
	WorkflowID string `json:"workflowID"`
	RunID      string `json:"runID"`
}

// RetryConfig defines the configuration for retry operations
type RetryConfig struct {
	MaxAttempts  int
	WaitInterval time.Duration
	Description  string
}

// DefaultRetryConfig provides default retry settings
var DefaultRetryConfig = RetryConfig{
	MaxAttempts:  5,
	WaitInterval: 500 * time.Millisecond,
	Description:  "operation",
}

// RetryWithConfig retries an operation with the specified configuration until it succeeds or max attempts are reached
func RetryWithConfig[T any](t *testing.T, config RetryConfig, operation func() (T, error), validator func(T) bool) (T, error) {
	var lastResult T
	var lastErr error

	for i := 0; i < config.MaxAttempts; i++ {
		result, err := operation()
		lastResult = result
		lastErr = err

		if err == nil && validator(result) {
			return result, nil
		}

		if i < config.MaxAttempts-1 {
			t.Logf("%s not successful (attempt %d/%d), waiting %v...",
				config.Description, i+1, config.MaxAttempts, config.WaitInterval)
			time.Sleep(config.WaitInterval)
		}
	}

	return lastResult, fmt.Errorf("max attempts (%d) reached: %v", config.MaxAttempts, lastErr)
}

// Retry is a convenience wrapper around RetryWithConfig using default settings
func Retry[T any](t *testing.T, description string, operation func() (T, error), validator func(T) bool) (T, error) {
	config := DefaultRetryConfig
	config.Description = description
	return RetryWithConfig(t, config, operation, validator)
}

// makeRequest is a helper function to make HTTP requests
func makeRequest(method, path string, body interface{}) (*http.Response, error) {
	var reqBody []byte
	var err error

	fullURL := fmt.Sprintf("http://localhost:%d%s", serverPort, path)
	log.Printf("Making %s request to: %s", method, fullURL)

	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		log.Printf("Request body: %s", string(reqBody))
	}

	req, err := http.NewRequest(method, fullURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	log.Printf("Response status: %s", resp.Status)
	return resp, nil
}

// DecodeResponse is a helper function to decode HTTP responses
func DecodeResponse[T any](resp *http.Response) (T, error) {
	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return result, fmt.Errorf("failed to decode response: %w", err)
	}
	return result, nil
}

// RequireEventuallyEqual is a helper function that retries until two values are equal
func RequireEventuallyEqual[T comparable](t *testing.T, description string, expected T, getValue func() (T, error)) {
	result, err := Retry(t, description, getValue, func(actual T) bool {
		return actual == expected
	})
	require.NoError(t, err)
	require.Equal(t, expected, result, "Values should eventually be equal")
}

// RequireEventually is a helper function that retries until a condition is met
func RequireEventually(t *testing.T, description string, condition func() (bool, error)) {
	_, err := Retry(t, description, condition, func(result bool) bool {
		return result
	})
	require.NoError(t, err)
}

// GenerateWorkerID generates a new UUID for worker ID
func GenerateWorkerID() string {
	return uuid.New().String()
}
