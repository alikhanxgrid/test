package impl

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"reference-app-wms-go/app/db"

	apiv2 "reference-app-wms-go/app/analytics/api/v2/openapi"

	"github.com/gin-gonic/gin"
	"go.temporal.io/sdk/client"
)

// AnalyticsServiceImpl implements the AnalyticsService interface
type AnalyticsServiceAPIV2 struct {
	temporalClient client.Client
	db             db.DB
	logger         *log.Logger
}

// NewAnalyticsServiceImpl creates a new instance of the analytics service
func NewAnalyticsServiceAPIV2(temporalClient client.Client, db db.DB) *AnalyticsServiceAPIV2 {
	return &AnalyticsServiceAPIV2{
		temporalClient: temporalClient,
		db:             db,
		logger:         log.New(os.Stdout, "[AnalyticsServiceAPIV2] ", log.LstdFlags),
	}
}

// ----------------- Implement the methods from server.gen.go ----------------- //

func (s *AnalyticsServiceAPIV2) GetWorkerCurrentTasks(c *gin.Context, workflowID string) (*apiv2.WorkerTaskStatus, error) {

	if workflowID == "" {
		c.JSON(http.StatusBadRequest, apiv2.Error{Error: "workflowID is required"})
		return nil, fmt.Errorf("workflowID is required")
	}
	s.logger.Printf("Getting current tasks for workflow: %s", workflowID)

	response, err := s.temporalClient.QueryWorkflow(ctx, workflowID, "", "getCurrentTasks")
	if err != nil {
		s.logger.Printf("Failed to query workflow for current tasks: %v", err)
		return nil, err
	}

}
