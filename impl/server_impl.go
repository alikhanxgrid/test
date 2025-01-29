package impl

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"handler/openapi" // Adjust to your actual module name
)

type ServerImpl struct {
	logger *log.Logger
}

func NewServerImpl() *ServerImpl {
	return &ServerImpl{
		logger: log.New(os.Stdout, "[ALERT-LOGGER] ", log.LstdFlags),
	}
}

// PostAlerts receives the AlertmanagerWebhook payload and logs it.
func (s *ServerImpl) PostAlerts(c *gin.Context) {
	var payload openapi.AlertmanagerWebhook
	if err := c.ShouldBindJSON(&payload); err != nil {
		s.logger.Printf("Error parsing JSON: %v\n", err)
		c.JSON(http.StatusBadRequest, openapi.Error{Error: err.Error()})
		return
	}

	s.logger.Println("Received Alertmanager Webhook:")
	s.logger.Printf("Version: %s, Status: %s, Receiver: %s\n",
		*payload.Version, *payload.Status, payload.Receiver)

	if payload.Alerts != nil {
		for i, alert := range *payload.Alerts {
			s.logger.Printf("Alert %d: status=%s fingerprint=%s\n",
				i, *alert.Status, alert.Fingerprint)
			if alert.Labels != nil {
				s.logger.Printf("  Labels: %v\n", *alert.Labels)
			}
			if alert.Annotations != nil {
				s.logger.Printf("  Annotations: %v\n", *alert.Annotations)
			}
		}
	}

	// Return a simple 200
	c.JSON(http.StatusOK, openapi.Response{Message: "Alerts logged successfully"})
}
