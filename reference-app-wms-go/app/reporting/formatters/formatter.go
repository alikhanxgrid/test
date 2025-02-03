package formatters

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"time"

	"reference-app-wms-go/app/reporting"
)

// Formatter defines the interface for report formatting
type Formatter interface {
	FormatDaily(report *reporting.DailyReport) ([]byte, error)
	FormatDashboard(data *reporting.DashboardData) ([]byte, error)
}

// JSONFormatter implements JSON formatting
type JSONFormatter struct{}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter() Formatter {
	return &JSONFormatter{}
}

func (f *JSONFormatter) FormatDaily(report *reporting.DailyReport) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

func (f *JSONFormatter) FormatDashboard(data *reporting.DashboardData) ([]byte, error) {
	return json.MarshalIndent(data, "", "  ")
}

// HTMLFormatter implements HTML formatting
type HTMLFormatter struct {
	dailyTemplate     *template.Template
	dashboardTemplate *template.Template
}

// NewHTMLFormatter creates a new HTML formatter
func NewHTMLFormatter() (Formatter, error) {
	// Load templates
	dailyTmpl, err := template.New("daily").Parse(dailyReportTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse daily report template: %w", err)
	}

	dashTmpl, err := template.New("dashboard").Parse(dashboardTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dashboard template: %w", err)
	}

	return &HTMLFormatter{
		dailyTemplate:     dailyTmpl,
		dashboardTemplate: dashTmpl,
	}, nil
}

func (f *HTMLFormatter) FormatDaily(report *reporting.DailyReport) ([]byte, error) {
	data := struct {
		Report    *reporting.DailyReport
		Generated time.Time
	}{
		Report:    report,
		Generated: time.Now(),
	}

	// Execute template
	return executeTemplate(f.dailyTemplate, data)
}

func (f *HTMLFormatter) FormatDashboard(data *reporting.DashboardData) ([]byte, error) {
	templateData := struct {
		Dashboard *reporting.DashboardData
		Generated time.Time
	}{
		Dashboard: data,
		Generated: time.Now(),
	}

	// Execute template
	return executeTemplate(f.dashboardTemplate, templateData)
}

// Helper function to execute templates
func executeTemplate(tmpl *template.Template, data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	err := tmpl.Execute(&buf, data)
	if err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}
	return buf.Bytes(), nil
}

// HTML templates
const dailyReportTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>Daily Site Report - {{.Report.Date.Format "2006-01-02"}}</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .section { margin: 20px 0; padding: 10px; border: 1px solid #ddd; }
        .metric { margin: 10px 0; }
        .chart { width: 100%; height: 300px; }
    </style>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
</head>
<body>
    <h1>Daily Site Report - {{.Report.SiteID}}</h1>
    <p>Date: {{.Report.Date.Format "2006-01-02"}}</p>
    
    <div class="section">
        <h2>Attendance</h2>
        <div class="metric">Total Workers: {{.Report.Attendance.TotalWorkers}}</div>
        <div class="metric">On-Time Workers: {{.Report.Attendance.OnTimeWorkers}}</div>
        <div class="metric">Late Workers: {{.Report.Attendance.LateWorkers}}</div>
        <div class="metric">Average Work Duration: {{.Report.Attendance.AverageWorkDuration}}</div>
    </div>

    <div class="section">
        <h2>Tasks</h2>
        <div class="metric">Total Tasks: {{.Report.Tasks.TotalTasks}}</div>
        <div class="metric">Completed Tasks: {{.Report.Tasks.CompletedTasks}}</div>
        <div class="metric">Blocked Tasks: {{.Report.Tasks.BlockedTasks}}</div>
        <div class="metric">Completion Rate: {{printf "%.2f%%" .Report.Tasks.CompletionRate}}</div>
    </div>

    <div class="section">
        <h2>Productivity</h2>
        <div class="metric">Tasks Per Worker: {{printf "%.2f" .Report.Productivity.TasksPerWorker}}</div>
        <div class="metric">Average Completion Time: {{.Report.Productivity.AvgCompletionTime}}</div>
        <div class="metric">Schedule Adherence: {{printf "%.2f%%" .Report.Productivity.ScheduleAdherence}}</div>
    </div>

    <div class="section">
        <h2>Site Metrics</h2>
        <div class="metric">Worker to Task Ratio: {{printf "%.2f" .Report.Site.WorkerToTaskRatio}}</div>
        <div class="metric">Utilization: {{printf "%.2f%%" .Report.Site.Utilization}}</div>
    </div>

    <footer>
        <p>Generated: {{.Generated.Format "2006-01-02 15:04:05"}}</p>
    </footer>
</body>
</html>
`

const dashboardTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>Real-Time Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .site-card { margin: 20px 0; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }
        .metric { margin: 10px 0; }
        .alert { color: red; }
    </style>
    <meta http-equiv="refresh" content="30">
</head>
<body>
    <h1>Real-Time Site Dashboard</h1>
    <p>Last Updated: {{.Dashboard.LastUpdated.Format "15:04:05"}}</p>

    {{range $siteID, $site := .Dashboard.Sites}}
    <div class="site-card">
        <h2>Site: {{$siteID}}</h2>
        <div class="metric">Active Workers: {{$site.ActiveWorkers}}</div>
        <div class="metric">Tasks In Progress: {{$site.TasksInProgress}}</div>
        <div class="metric">Blocked Tasks: {{$site.BlockedTasks}}</div>
        <div class="metric">Current Utilization: {{printf "%.2f%%" $site.CurrentUtilization}}</div>
        {{if gt $site.AlertCount 0}}
        <div class="metric alert">Active Alerts: {{$site.AlertCount}}</div>
        {{end}}
    </div>
    {{end}}

    <footer>
        <p>Generated: {{.Generated.Format "2006-01-02 15:04:05"}}</p>
    </footer>
</body>
</html>
`
