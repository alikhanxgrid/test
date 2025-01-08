package main

import (
	"log"
	"os"

	"reference-app-wms-go/app/dwr/activities"
	"reference-app-wms-go/app/dwr/workflows"
	"reference-app-wms-go/app/scheduling"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

const TemporalTaskQueue = "DWR-TASK-QUEUE"

func main() {
	c, err := client.Dial(client.Options{
		HostPort: os.Getenv("TEMPORAL_GRPC_ENDPOINT"),
	})
	if err != nil {
		log.Fatalln("Unable to create client", err)
	}
	defer c.Close()

	w := worker.New(c, TemporalTaskQueue, worker.Options{})

	// Initialize scheduling API client
	schedulingAPIClient := scheduling.NewSchedulingAPIClient(getSchedulingAPIEndpoint())

	// Register workflows
	w.RegisterWorkflow(workflows.DailyWorkerRoutine)

	// Register activities
	activities := activities.NewActivities(schedulingAPIClient)
	w.RegisterActivity(activities.RecordCheckInActivity)
	w.RegisterActivity(activities.RetrieveDailyTasksActivity)
	w.RegisterActivity(activities.UpdateTaskProgressActivity)
	w.RegisterActivity(activities.RecordCheckOutActivity)

	// Start listening to the Task Queue
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("Unable to start worker", err)
	}
}

func getSchedulingAPIEndpoint() string {
	endpoint := os.Getenv("SCHEDULING_API_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:8081"
	}
	return endpoint
}
