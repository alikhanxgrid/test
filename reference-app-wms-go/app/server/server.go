package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"go.temporal.io/sdk/client"

	"reference-app-wms-go/app/analytics"
	"reference-app-wms-go/app/db"

	// "reference-app-wms-go/app/dwr/api"
	"reference-app-wms-go/app/dwr/api/v2/impl"
	apiv2 "reference-app-wms-go/app/dwr/api/v2/openapi"
	"reference-app-wms-go/app/scheduling"
)

// Server represents the HTTP server for the WMS application
type Server struct {
	router           *gin.Engine
	logger           *log.Logger
	temporalClient   client.Client
	db               *sqlx.DB
	analyticsService analytics.AnalyticsService
	schedulingAPI    *scheduling.SchedulingAPI
	analyticsAPI     *analytics.AnalyticsAPI
	// jobExecutionAPI  *api.JobExecutionAPI
	jobExecutionV2 *impl.JobExecutionAPIV2
}

// NewServer creates a new Server instance
func NewServer(sqlxDB *sqlx.DB, temporalClient client.Client) *Server {
	logger := log.New(os.Stdout, "[WMS Server] ", log.LstdFlags)
	router := gin.Default()

	dbWrapper := db.NewPostgresDB(sqlxDB)
	analyticsService := analytics.NewAnalyticsService(temporalClient, dbWrapper)
	schedulingService := scheduling.NewSchedulingService(dbWrapper)
	schedulingAPI := scheduling.NewSchedulingAPI(schedulingService, logger)
	analyticsAPI := analytics.NewAnalyticsAPI(analyticsService, logger)

	// Initialize both v1 and v2 job execution APIs
	// jobExecutionAPI := api.NewJobExecutionAPI(temporalClient)
	jobExecutionV2 := impl.NewJobExecutionAPIV2(temporalClient)

	server := &Server{
		router:           router,
		logger:           logger,
		temporalClient:   temporalClient,
		db:               sqlxDB,
		analyticsService: analyticsService,
		schedulingAPI:    schedulingAPI,
		analyticsAPI:     analyticsAPI,
		// jobExecutionAPI:  jobExecutionAPI,
		jobExecutionV2: jobExecutionV2,
	}

	server.setupRoutes()
	return server
}

// setupRoutes configures all the routes for the server
func (s *Server) setupRoutes() {
	// Keep v1 APIs at root level for backward compatibility
	s.schedulingAPI.SetupRoutes(s.router)
	s.analyticsAPI.SetupRoutes(s.router)
	// s.jobExecutionAPI.SetupRoutes(s.router)

	// Register v2 job execution handlers at root level instead of /api/v2
	apiv2.RegisterHandlers(s.router, s.jobExecutionV2)
}

// Start starts the HTTP server
func (s *Server) Start(addr string) error {
	srv := &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	// Channel to listen for errors coming from the listener.
	serverErrors := make(chan error, 1)

	// Start the service listening for requests.
	go func() {
		s.logger.Printf("Server is running on %s", addr)
		serverErrors <- srv.ListenAndServe()
	}()

	// Channel to listen for an interrupt or terminate signal from the OS.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Blocking main and waiting for shutdown.
	select {
	case err := <-serverErrors:
		return fmt.Errorf("error starting server: %w", err)

	case <-shutdown:
		s.logger.Println("Starting shutdown...")

		// Give outstanding requests a deadline for completion.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Asking listener to shut down and shed load.
		if err := srv.Shutdown(ctx); err != nil {
			// Error from closing listeners, or context timeout:
			s.logger.Printf("Graceful shutdown did not complete in %v: %v", 5*time.Second, err)
			if err := srv.Close(); err != nil {
				return fmt.Errorf("could not stop server gracefully: %w", err)
			}
		}
	}

	return nil
}

// Stop performs cleanup and stops the server
func (s *Server) Stop() error {
	s.temporalClient.Close()
	if err := s.db.Close(); err != nil {
		s.logger.Printf("Error closing database connection: %v", err)
	}
	return nil
}
