package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.temporal.io/sdk/client"

	"reference-app-wms-go/app/analytics"
	"reference-app-wms-go/app/db"
	"reference-app-wms-go/app/reporting"
	"reference-app-wms-go/app/reporting/formatters"
)

func main() {
	// Parse command line flags
	reportType := flag.String("type", "daily", "Report type (daily or dashboard)")
	siteIDs := flag.String("sites", "", "Comma-separated list of site IDs (use 'all' for all sites)")
	startDate := flag.String("start-date", time.Now().Format("2006-01-02"), "Start date (YYYY-MM-DD)")
	endDate := flag.String("end-date", "", "End date (YYYY-MM-DD), defaults to start-date if not specified")
	format := flag.String("format", "html", "Output format (json or html)")
	outputFile := flag.String("output", "", "Output file (defaults to stdout)")
	flag.Parse()

	// Validate flags
	if *siteIDs == "" {
		log.Fatal("sites parameter is required (use 'all' for all sites)")
	}

	reportStartDate, err := time.Parse("2006-01-02", *startDate)
	if err != nil {
		log.Fatalf("Invalid start date format: %v", err)
	}

	reportEndDate := reportStartDate
	if *endDate != "" {
		reportEndDate, err = time.Parse("2006-01-02", *endDate)
		if err != nil {
			log.Fatalf("Invalid end date format: %v", err)
		}
		if reportEndDate.Before(reportStartDate) {
			log.Fatal("End date cannot be before start date")
		}
	}

	// Initialize services
	analyticsService := initAnalyticsService()
	generator := reporting.NewGenerator(analyticsService)

	// Initialize formatter based on format
	var formatter formatters.Formatter
	switch strings.ToLower(*format) {
	case "json":
		formatter = formatters.NewJSONFormatter()
	case "html":
		var err error
		formatter, err = formatters.NewHTMLFormatter()
		if err != nil {
			log.Fatalf("Failed to create HTML formatter: %v", err)
		}
	default:
		log.Fatalf("Unsupported format: %s", *format)
	}

	ctx := context.Background()
	var reportData []byte

	switch strings.ToLower(*reportType) {
	case "daily":
		if *siteIDs == "all" {
			// Generate reports for each date in the range
			currentDate := reportStartDate
			for !currentDate.After(reportEndDate) {
				reports, err := generator.GenerateAllSites(ctx, currentDate)
				if err != nil {
					log.Fatalf("Failed to generate reports for all sites: %v", err)
				}
				for i, report := range reports {
					reportData, err = formatter.FormatDaily(report)
					if err != nil {
						log.Printf("Failed to format report for site %s: %v", report.SiteID, err)
						continue
					}
					// If output file is specified, create separate files for each site and date
					if *outputFile != "" {
						outputPath := *outputFile
						if len(reports) > 1 {
							ext := filepath.Ext(outputPath)
							base := strings.TrimSuffix(outputPath, ext)
							outputPath = fmt.Sprintf("%s_%s_%s%s", base, report.SiteID, currentDate.Format("2006-01-02"), ext)
						}
						writeOutput(reportData, outputPath)
					} else if i > 0 {
						fmt.Println("\n---\n")
						fmt.Println(string(reportData))
					} else {
						fmt.Println(string(reportData))
					}
				}
				currentDate = currentDate.AddDate(0, 0, 1)
			}
		} else {
			sites := strings.Split(*siteIDs, ",")
			// Generate reports for each site and date
			currentDate := reportStartDate
			for !currentDate.After(reportEndDate) {
				for i, siteID := range sites {
					report, err := generator.GenerateDaily(ctx, siteID, currentDate)
					if err != nil {
						log.Printf("Failed to generate daily report for site %s on %s: %v", siteID, currentDate.Format("2006-01-02"), err)
						continue
					}
					reportData, err = formatter.FormatDaily(report)
					if err != nil {
						log.Printf("Failed to format daily report for site %s: %v", siteID, err)
						continue
					}
					// If output file is specified, create separate files for each site and date
					if *outputFile != "" {
						outputPath := *outputFile
						if len(sites) > 1 || !reportStartDate.Equal(reportEndDate) {
							ext := filepath.Ext(outputPath)
							base := strings.TrimSuffix(outputPath, ext)
							outputPath = fmt.Sprintf("%s_%s_%s%s", base, siteID, currentDate.Format("2006-01-02"), ext)
						}
						writeOutput(reportData, outputPath)
					} else if i > 0 {
						fmt.Println("\n---\n")
						fmt.Println(string(reportData))
					} else {
						fmt.Println(string(reportData))
					}
				}
				currentDate = currentDate.AddDate(0, 0, 1)
			}
		}

	case "dashboard":
		var sites []string
		if *siteIDs == "all" {
			allSites, err := analyticsService.GetAllJobSites(ctx)
			if err != nil {
				log.Fatalf("Failed to get all sites: %v", err)
			}
			for _, site := range allSites {
				sites = append(sites, site.ID)
			}
		} else {
			sites = strings.Split(*siteIDs, ",")
		}

		dashboard, err := generator.GenerateDashboard(ctx, reporting.ReportFilters{
			SiteIDs: sites,
		})
		if err != nil {
			log.Fatalf("Failed to generate dashboard: %v", err)
		}
		reportData, err = formatter.FormatDashboard(dashboard)
		if err != nil {
			log.Fatalf("Failed to format dashboard: %v", err)
		}
		writeOutput(reportData, *outputFile)

	default:
		log.Fatalf("Unsupported report type: %s", *reportType)
	}
}

func initAnalyticsService() analytics.AnalyticsService {
	// Initialize Temporal client
	temporalClient, err := client.Dial(client.Options{HostPort: os.Getenv("TEMPORAL_GRPC_ENDPOINT")})
	if err != nil {
		log.Fatalf("Failed to create Temporal client: %v", err)
	}

	// Initialize database connection
	dbConfig := db.DBConfig{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
	}

	// Set default values if environment variables are not set
	if dbConfig.Host == "" {
		dbConfig.Host = "localhost"
	}
	if dbConfig.Port == "" {
		dbConfig.Port = "5432"
	}
	if dbConfig.User == "" {
		dbConfig.User = "abdullahshah"
	}
	if dbConfig.DBName == "" {
		dbConfig.DBName = "task_management_test_db"
	}

	// Connect to database
	sqlxDB, err := sqlx.Connect("postgres", dbConfig.ConnectionString())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Create database wrapper
	database := db.NewPostgresDB(sqlxDB)

	// Create analytics service
	return analytics.NewAnalyticsService(temporalClient, database)
}

func writeOutput(data []byte, outputFile string) {
	if outputFile == "" {
		fmt.Println(string(data))
		return
	}

	// Create the output directory if it doesn't exist
	outputDir := filepath.Dir(outputFile)
	if outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			log.Fatalf("Failed to create output directory: %v", err)
		}
	}

	// Write the file
	err := os.WriteFile(outputFile, data, 0644)
	if err != nil {
		log.Fatalf("Failed to write output file: %v", err)
	}
}
