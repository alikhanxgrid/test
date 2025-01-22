package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.temporal.io/sdk/client"

	"reference-app-wms-go/app/db"
	"reference-app-wms-go/app/server"
)

func main() {
	// Get database configuration from command-line flags
	dbConfig := db.NewDBConfigFromFlags()

	// Connect to database
	dbConn, err := sqlx.Connect("postgres", dbConfig.ConnectionString())
	if err != nil {
		log.Fatalln("Failed to connect to database:", err)
	}
	defer dbConn.Close()

	hostPort := os.Getenv("TEMPORAL_GRPC_ENDPOINT")
	if hostPort == "" {
		// Fallback if the environment variable is unset or empty
		hostPort = "localhost:7233"
	}

	// Create temporal client
	temporalClient, err := client.Dial(client.Options{HostPort: hostPort})
	if err != nil {
		log.Fatalln("Failed to create temporal client:", err)
	}
	defer temporalClient.Close()

	// Create and start server
	srv := server.NewServer(dbConn, temporalClient)

	// Start server in a goroutine
	go func() {
		if err := srv.Start(":8081"); err != nil {
			log.Fatalln("Failed to start server:", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	if err := srv.Stop(); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}
