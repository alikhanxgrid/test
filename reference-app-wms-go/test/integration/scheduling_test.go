package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSchedulingAPI(t *testing.T) {
	// 1. Create a job site
	t.Log("Creating job site...")
	createJobSiteReq := map[string]string{
		"name":     "Test Construction Site",
		"location": "123 Test St, Test City",
	}

	resp, err := makeRequest("POST", "/scheduling/job-sites", createJobSiteReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var jobSite jobSiteResponse
	err = json.NewDecoder(resp.Body).Decode(&jobSite)
	require.NoError(t, err)
	require.NotEmpty(t, jobSite.ID)
	resp.Body.Close()

	// 2. Get the created job site
	t.Log("Getting job site...")
	resp, err = makeRequest("GET", fmt.Sprintf("/scheduling/job-sites/%s", jobSite.ID), nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var fetchedJobSite jobSiteResponse
	err = json.NewDecoder(resp.Body).Decode(&fetchedJobSite)
	require.NoError(t, err)
	require.Equal(t, jobSite.ID, fetchedJobSite.ID)
	resp.Body.Close()

	// 3. Create a schedule for the job site
	t.Log("Creating schedule...")
	startDate := time.Now().AddDate(0, 0, 1) // Tomorrow
	endDate := startDate.AddDate(0, 0, 7)    // Week-long schedule

	createScheduleReq := map[string]string{
		"startDate": startDate.Format(time.RFC3339),
		"endDate":   endDate.Format(time.RFC3339),
	}

	resp, err = makeRequest("POST", fmt.Sprintf("/scheduling/job-sites/%s/schedules", jobSite.ID), createScheduleReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var schedule scheduleResponse
	err = json.NewDecoder(resp.Body).Decode(&schedule)
	require.NoError(t, err)
	require.NotEmpty(t, schedule.ID)
	require.Equal(t, jobSite.ID, schedule.JobSiteID)
	resp.Body.Close()

	// 4. Get schedules for the job site
	t.Log("Getting schedules...")
	resp, err = makeRequest("GET", fmt.Sprintf("/scheduling/job-sites/%s/schedules", jobSite.ID), nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var schedules []scheduleResponse
	err = json.NewDecoder(resp.Body).Decode(&schedules)
	require.NoError(t, err)
	require.NotEmpty(t, schedules)
	require.Equal(t, schedule.ID, schedules[0].ID)
	resp.Body.Close()

	// 5. Test schedule overlap validation
	t.Log("Testing schedule overlap validation...")
	overlapScheduleReq := map[string]string{
		"startDate": startDate.AddDate(0, 0, 2).Format(time.RFC3339), // Overlapping dates
		"endDate":   endDate.AddDate(0, 0, 2).Format(time.RFC3339),
	}

	resp, err = makeRequest("POST", fmt.Sprintf("/scheduling/job-sites/%s/schedules/validate", jobSite.ID), overlapScheduleReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusConflict, resp.StatusCode)
	resp.Body.Close()

	// 6. Get active schedules
	t.Log("Getting active schedules...")
	resp, err = makeRequest("GET", fmt.Sprintf("/scheduling/job-sites/%s/schedules/active", jobSite.ID), nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var activeSchedules []scheduleResponse
	err = json.NewDecoder(resp.Body).Decode(&activeSchedules)
	require.NoError(t, err)
	resp.Body.Close()

	// 7. Get upcoming schedules
	t.Log("Getting upcoming schedules...")
	resp, err = makeRequest("GET", fmt.Sprintf("/scheduling/job-sites/%s/schedules/upcoming", jobSite.ID), nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var upcomingSchedules []scheduleResponse
	err = json.NewDecoder(resp.Body).Decode(&upcomingSchedules)
	require.NoError(t, err)
	require.NotEmpty(t, upcomingSchedules)
	resp.Body.Close()

	t.Log("Scheduling API test completed successfully")
}
