package impl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"handler/openapi" // Adjust module/package path as needed
)

const (
	serviceNowBaseURL = ""
	serviceNowUser    = ""
	serviceNowPass    = ""
)

type ServerImpl struct {
	logger *log.Logger
}

func NewServerImpl() *ServerImpl {
	return &ServerImpl{
		logger: log.New(os.Stdout, "[ALERT-LOGGER] ", log.LstdFlags),
	}
}

// SeverityMapping defines the mapping between Alertmanager severity and ServiceNow impact/urgency
type SeverityMapping struct {
	impact  string
	urgency string
}

// Severity level mappings for ServiceNow
var severityToServiceNow = map[string]SeverityMapping{
	"critical": {impact: "1", urgency: "1"}, // High/High
	"error":    {impact: "2", urgency: "1"}, // Medium/High
	"warning":  {impact: "2", urgency: "2"}, // Medium/Medium
	"info":     {impact: "3", urgency: "3"}, // Low/Low
	// Default will be Medium/Medium if severity is not matched
}

// PostAlerts receives the AlertmanagerWebhook payload and logs it.
// For each "firing" alert, create a new incident in ServiceNow.
func (s *ServerImpl) PostAlerts(c *gin.Context) {
	var payload openapi.AlertmanagerWebhook
	if err := c.ShouldBindJSON(&payload); err != nil {
		s.logger.Printf("Error parsing JSON: %v\n", err)
		c.JSON(http.StatusBadRequest, openapi.Error{Error: err.Error()})
		return
	}

	s.logger.Println("Received Alertmanager Webhook:")
	s.logger.Printf("Version: %s, Status: %s\n", payload.Version, payload.Status)

	// Convert payload to JSON for better logging
	payloadJSON, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		s.logger.Printf("Error marshaling payload: %v\n", err)
	} else {
		s.logger.Printf("Payload: %s\n", string(payloadJSON))
	}

	// Receiver is a *string, so check for nil before dereferencing:
	receiver := "<nil>"
	if payload.Receiver != nil {
		receiver = *payload.Receiver
	}
	s.logger.Printf("Receiver: %s\n", receiver)

	// Iterate over the Alerts slice:
	// for i, alert := range payload.Alerts {
	// 	// read the alert status
	// 	alertStatus := "<nil>"
	// 	if alert.Status != nil {
	// 		alertStatus = *alert.Status
	// 	}

	// 	fingerprint := "<nil>"
	// 	if alert.Fingerprint != nil {
	// 		fingerprint = *alert.Fingerprint
	// 	}

	// 	s.logger.Printf("Alert %d: status=%s fingerprint=%s\n", i, alertStatus, fingerprint)

	// 	// Label & Annotation logging
	// 	if alert.Labels != nil {
	// 		s.logger.Printf("  Labels: %v\n", *alert.Labels)
	// 	}
	// 	if alert.Annotations != nil {
	// 		s.logger.Printf("  Annotations: %v\n", *alert.Annotations)
	// 	}

	// 	// Create a ServiceNow incident ONLY if alert is "firing"
	// 	if alertStatus == "firing" {
	// 		shortDesc := fmt.Sprintf("Alert firing (fingerprint: %s)", fingerprint)
	// 		desc := "No description available."

	// 		// If the alert has an annotation "description", we'll try to use it:
	// 		if alert.Annotations != nil {
	// 			if rawDesc, ok := (*alert.Annotations)["description"]; ok {
	// 				// Convert to string if possible
	// 				if strDesc, ok := rawDesc.(string); ok {
	// 					desc = strDesc
	// 				}
	// 			}
	// 		}

	// 		incidentNumber, err := s.createIncident(shortDesc, desc, alert)
	// 		if err != nil {
	// 			s.logger.Printf("[ERROR] failed to create incident for alert: %v", err)
	// 			// We won't fail the entire request for one incident error, so just continue
	// 			continue
	// 		}
	// 		s.logger.Printf("[INFO] Created incident '%s' for alert fingerprint '%s'", incidentNumber, fingerprint)
	// 	} else {
	// 		s.logger.Printf("[INFO] Not creating incident for alert with status %s", alertStatus)
	// 	}
	// }

	// Return a simple 200
	c.JSON(http.StatusOK, openapi.Response{Message: "Alerts processed successfully"})
}

// createIncident does an HTTP POST to ServiceNow's /incident endpoint, filling in
// the required fields. Returns the newly created incident number or an error.
func (s *ServerImpl) createIncident(shortDescription, description string, alert openapi.Alert) (string, error) {
	// Get severity from alert labels
	severity := "warning" // default severity
	if alert.Labels != nil {
		if rawSeverity, ok := (*alert.Labels)["severity"]; ok {
			if strSeverity, ok := rawSeverity.(string); ok {
				severity = strSeverity
			}
		}
	}

	// Get impact/urgency mapping based on severity
	mapping, ok := severityToServiceNow[severity]
	if !ok {
		// Default to Medium/Medium if severity not found in mapping
		mapping = severityToServiceNow["warning"]
	}

	bodyMap := map[string]interface{}{
		"short_description": shortDescription,
		"description":       description,
		"caller_id":         "guest",
		"impact":            mapping.impact,
		"urgency":           mapping.urgency,
		"assignment_group":  "Assignment Group XYZ",
	}

	bodyBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return "", fmt.Errorf("failed to marshal incident body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, serviceNowBaseURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(serviceNowUser, serviceNowPass)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var createResp struct {
		Result struct {
			Number string `json:"number"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return "", fmt.Errorf("failed to decode ServiceNow response: %w", err)
	}

	return createResp.Result.Number, nil
}
