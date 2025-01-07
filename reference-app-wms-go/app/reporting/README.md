# Reporting Package

The reporting package provides functionality to generate comprehensive analytics reports for job sites and workers. It supports both daily reports and real-time dashboards, with output in either JSON or HTML format.

## Features

- **Daily Reports**: Detailed reports for a specific job site including:
  - Attendance metrics (worker counts, check-in/out distributions)
  - Task metrics (completion rates, status distributions)
  - Break patterns
  - Site utilization
  - Worker productivity

- **Real-time Dashboards**: Live monitoring of multiple sites showing:
  - Active worker counts
  - Current task status
  - Site utilization rates
  - Blocked tasks
  - Recent breaks
  - Active alerts

- **Output Formats**:
  - JSON: Machine-readable format for API integration
  - HTML: Human-readable format with charts and styling

## Installation and Setup

### Environment Variables

The following environment variables can be set to configure the database and Temporal connection:

```bash
# Database configuration (with default values)
DB_HOST=localhost
DB_PORT=5432
DB_USER=abdullahshah
DB_NAME=wms
DB_PASSWORD=  # Required, no default

# Temporal configuration
TEMPORAL_HOST_PORT=localhost:7233  # Default Temporal server address
```

If environment variables are not set, the application will use the default values shown above.

### Building the CLI Tool

1. Navigate to the CLI directory:
```bash
cd xfield-ops/reference-app-wms-go/app/reporting/cmd/report
```

2. Build the binary:
```bash
go build -o report main.go
```

This will create an executable named `report` in the current directory.

### Alternative: Running without Building

You can also run the tool directly using `go run`:
```bash
cd xfield-ops/reference-app-wms-go/app/reporting/cmd/report
go run main.go [flags]
```

## Usage

### Command Line Interface

The package includes a CLI tool for generating reports. After building, you can run it as follows:

```bash
# From the build directory
./report -type daily -sites SITE123 -date 2024-02-14 -format html -output report.html

# Or using go run
go run main.go -type daily -sites SITE123 -date 2024-02-14 -format html -output report.html

# Generate a JSON dashboard for multiple sites
./report -type dashboard -sites SITE1,SITE2,SITE3 -format json

# Generate today's report to stdout (defaults to HTML)
./report -type daily -sites SITE123

# Generate reports for multiple sites
./report -type daily -sites SITE1,SITE2,SITE3 -date 2024-02-14 -format html -output reports/

# Generate reports for all sites
./report -type daily -sites all -date 2024-02-14 -format html -output reports/daily/

# Generate dashboard for all sites and save to a specific directory
./report -type dashboard -sites all -format json -output reports/dashboard/current.json

# Organize reports by date
./report -type daily -sites all -format html -output reports/2024/02/14/
```

### Command Line Options

- `-type`: Report type (required)
  - `daily`: Generate a daily report
  - `dashboard`: Generate a real-time dashboard

- `-sites`: Site selection (required)
  - Comma-separated list of site IDs
  - Use `all` to generate reports for all available sites
  - For multiple sites, separate output files will be created with site IDs appended to the filename

- `-date`: Report date in YYYY-MM-DD format
  - Optional for daily reports (defaults to today)
  - Not used for dashboards

- `-format`: Output format
  - `html`: HTML format with styling (default)
  - `json`: JSON format

- `-output`: Output file path
  - Optional (defaults to stdout)
  - For HTML format, should end with .html
  - For JSON format, should end with .json
  - Can include directories (e.g., `reports/daily/report.html`)
  - Directories will be created automatically if they don't exist
  - When generating reports for multiple sites:
    - Single site: Uses the exact filename specified
    - Multiple sites: Appends site ID to filename (e.g., `reports/daily/report_SITE1.html`)

### Viewing HTML Reports

After generating an HTML report, you can open it in your browser using any of these methods:

1. **Using the terminal (macOS/Linux)**:
   ```bash
   # Generate and open the report
   ./report -type daily -sites SITE123 -output report.html
   open report.html  # macOS
   xdg-open report.html  # Linux
   ```

2. **Direct browser access**:
   - Simply drag and drop the HTML file into any browser window
   - Or use File → Open in your browser and navigate to the file

3. **Using a local server** (if the HTML has relative resource paths):
   ```bash
   # Generate the report
   ./report -type daily -sites SITE123 -output report.html
   
   # Start a simple HTTP server
   python3 -m http.server 8000
   
   # Open in browser
   open http://localhost:8000/report.html  # macOS
   xdg-open http://localhost:8000/report.html  # Linux
   ```

### Programmatic Usage

```go
package main

import (
    "context"
    "time"
    "reference-app-wms-go/app/analytics"
    "reference-app-wms-go/app/reporting"
    "reference-app-wms-go/app/reporting/formatters"
)

func main() {
    // Initialize services
    analyticsService := analytics.NewAnalyticsService()
    generator := reporting.NewGenerator(analyticsService)

    // Create formatter
    htmlFormatter, _ := formatters.NewHTMLFormatter()
    // or
    jsonFormatter := formatters.NewJSONFormatter()

    // Generate daily report for a single site
    report, err := generator.GenerateDaily(
        context.Background(),
        "SITE123",
        time.Now(),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Generate reports for all sites
    reports, err := generator.GenerateAllSites(
        context.Background(),
        time.Now(),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Format reports
    for _, report := range reports {
        output, err := htmlFormatter.FormatDaily(report)
        if err != nil {
            log.Printf("Failed to format report for site %s: %v", report.SiteID, err)
            continue
        }
        // Write output to file or process as needed
    }

    // Generate dashboard for multiple sites
    dashboard, err := generator.GenerateDashboard(
        context.Background(),
        reporting.ReportFilters{
            SiteIDs: []string{"SITE1", "SITE2"},
        },
    )
    if err != nil {
        log.Fatal(err)
    }

    // Format dashboard
    output, err = htmlFormatter.FormatDashboard(dashboard)
    if err != nil {
        log.Fatal(err)
    }
}
```

## Report Contents

### Daily Report Structure

```go
type DailyReport struct {
    Date         time.Time
    SiteID       string
    Attendance   AttendanceMetrics
    Tasks        TaskMetrics
    Breaks       BreakMetrics
    Productivity ProductivityMetrics
    Site         SiteMetrics
}
```

- **Attendance Metrics**:
  - Total workers
  - On-time/late workers
  - Average work duration
  - Check-in/out distributions

- **Task Metrics**:
  - Total tasks
  - Completed tasks
  - Blocked tasks
  - Completion rate
  - Average task duration
  - Status distribution

- **Break Metrics**:
  - Total breaks
  - Average break duration
  - Break time distribution
  - Compliance rate

- **Productivity Metrics**:
  - Tasks per worker
  - Average completion time
  - Schedule adherence
  - Blockage resolution time

- **Site Metrics**:
  - Worker to task ratio
  - Peak hours
  - Utilization rate
  - Schedule coverage

### Dashboard Structure

```go
type DashboardData struct {
    Timestamp    time.Time
    Sites        []SiteDashboard
}

type SiteDashboard struct {
    SiteID       string
    ActiveWorkers int
    TaskStatus   map[string]int
    Utilization  float64
    BlockedTasks int
    RecentBreaks int
    Alerts       []string
}
```

## Output Formats

### HTML Format

The HTML output includes:
- Responsive layout
- Charts and graphs for metrics visualization
- Tables for detailed data
- Color-coded status indicators
- Navigation between sections
- Print-friendly styling

### JSON Format

The JSON output follows the same structure as the Go types, making it easy to parse and integrate with other systems. All timestamps are in RFC3339 format. 