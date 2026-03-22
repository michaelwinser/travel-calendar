package main

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/michaelwinser/appbase"
	"github.com/michaelwinser/appbase/auth"
	"github.com/michaelwinser/travel-calendar/api"
	harness "github.com/michaelwinser/appbase/testing"
)

var testSessions *auth.SessionStore

func setupTestApp(t *testing.T) http.Handler {
	t.Helper()
	os.Setenv("STORE_TYPE", "sqlite")
	os.Setenv("SQLITE_DB_PATH", ":memory:")
	t.Cleanup(func() {
		os.Unsetenv("SQLITE_DB_PATH")
	})

	a, err := appbase.New(appbase.Config{Name: "travel-calendar", Quiet: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })

	store, err := NewActivityStore(a.DB())
	if err != nil {
		t.Fatal(err)
	}

	activityServer := &ActivityServer{store: store}
	api.HandlerFromMux(activityServer, a.Server().Router())

	testSessions = a.Sessions()
	return a.Server().Router()
}

func TestUseCases(t *testing.T) {
	h := harness.New(t, setupTestApp)

	login := func(c *harness.Client) {
		session, err := testSessions.Create("test@example.com", "test@example.com", 1*time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		c.SetCookie(auth.CookieName, session.ID)
	}

	// --- Authentication ---

	h.Run("UC-0001", "Unauthenticated request returns 401", func(c *harness.Client) {
		resp := c.GET("/api/activities")
		c.AssertStatus(resp, 401)
	})

	// --- CRUD basics ---

	h.Run("UC-0002", "List activities returns empty array", func(c *harness.Client) {
		login(c)
		resp := c.GET("/api/activities")
		c.AssertStatus(resp, 200)
		c.AssertJSONArray(resp, 0)
	})

	h.Run("UC-0003", "Create a single-day activity", func(c *harness.Client) {
		login(c)
		resp := c.POST("/api/activities", `{
			"title": "Dentist",
			"type": "commitment",
			"startDate": "2026-04-15",
			"location": "Home"
		}`)
		c.AssertStatus(resp, 201)
		c.AssertJSONHas(resp, "title", "Dentist")
		c.AssertJSONHas(resp, "type", "commitment")
		c.AssertJSONHas(resp, "startDate", "2026-04-15")
		c.AssertJSONHas(resp, "endDate", "2026-04-15")
		c.AssertJSONHas(resp, "location", "Home")
		c.AssertJSONHas(resp, "source", "manual")
	})

	h.Run("UC-0004", "Create activity with empty title fails", func(c *harness.Client) {
		login(c)
		resp := c.POST("/api/activities", `{
			"title": "",
			"type": "stay",
			"startDate": "2026-04-15"
		}`)
		c.AssertStatus(resp, 400)
	})

	h.Run("UC-0005", "Create activity with invalid type fails", func(c *harness.Client) {
		login(c)
		resp := c.POST("/api/activities", `{
			"title": "Bad Type",
			"type": "invalid",
			"startDate": "2026-04-15"
		}`)
		c.AssertStatus(resp, 400)
	})

	h.Run("UC-0006", "Create activity with end before start fails", func(c *harness.Client) {
		login(c)
		resp := c.POST("/api/activities", `{
			"title": "Backwards",
			"type": "stay",
			"startDate": "2026-04-15",
			"endDate": "2026-04-10"
		}`)
		c.AssertStatus(resp, 400)
	})

	// --- UC-1001: Multi-day activity creation ---

	h.Run("UC-1001", "Create a multi-day activity", func(c *harness.Client) {
		login(c)
		resp := c.POST("/api/activities", `{
			"title": "European Summit",
			"type": "conference",
			"startDate": "2026-10-04",
			"endDate": "2026-10-07",
			"location": "Brussels"
		}`)
		c.AssertStatus(resp, 201)
		c.AssertJSONHas(resp, "title", "European Summit")
		c.AssertJSONHas(resp, "startDate", "2026-10-04")
		c.AssertJSONHas(resp, "endDate", "2026-10-07")
		c.AssertJSONHas(resp, "location", "Brussels")

		// Verify it appears in list
		listResp := c.GET("/api/activities")
		c.AssertStatus(listResp, 200)
		items := listResp.JSONArray()
		found := false
		for _, item := range items {
			if item["title"] == "European Summit" {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("multi-day activity not found in list")
		}
	})

	// --- Get and Delete ---

	h.Run("UC-0007", "Get activity by ID", func(c *harness.Client) {
		login(c)
		createResp := c.POST("/api/activities", `{
			"title": "Fetch Me",
			"type": "stay",
			"startDate": "2026-05-01",
			"location": "Paris"
		}`)
		c.AssertStatus(createResp, 201)
		id := createResp.JSON()["id"].(string)

		getResp := c.GET("/api/activities/" + id)
		c.AssertStatus(getResp, 200)
		c.AssertJSONHas(getResp, "title", "Fetch Me")
		c.AssertJSONHas(getResp, "id", id)
	})

	h.Run("UC-0008", "Get nonexistent activity returns 404", func(c *harness.Client) {
		login(c)
		resp := c.GET("/api/activities/nonexistent-id")
		c.AssertStatus(resp, 404)
	})

	h.Run("UC-0009", "Delete an activity", func(c *harness.Client) {
		login(c)
		createResp := c.POST("/api/activities", `{
			"title": "Delete Me",
			"type": "stay",
			"startDate": "2026-06-01"
		}`)
		c.AssertStatus(createResp, 201)
		id := createResp.JSON()["id"].(string)

		delResp := c.DELETE("/api/activities/" + id)
		c.AssertStatus(delResp, 200)

		// Verify it's gone
		getResp := c.GET("/api/activities/" + id)
		c.AssertStatus(getResp, 404)
	})

	// --- Filtering ---

	h.Run("UC-0010", "List with month filter", func(c *harness.Client) {
		login(c)
		// Create activities in different months
		c.POST("/api/activities", `{
			"title": "March Thing",
			"type": "stay",
			"startDate": "2026-03-15",
			"location": "NYC"
		}`)
		c.POST("/api/activities", `{
			"title": "July Thing",
			"type": "vacation",
			"startDate": "2026-07-10",
			"endDate": "2026-07-20",
			"location": "Hawaii"
		}`)

		// Filter to July
		resp := c.GET("/api/activities?month=2026-07")
		c.AssertStatus(resp, 200)
		items := resp.JSONArray()
		for _, item := range items {
			title := item["title"].(string)
			if title == "March Thing" {
				t.Fatal("March activity should not appear in July filter")
			}
		}
		found := false
		for _, item := range items {
			if item["title"] == "July Thing" {
				found = true
			}
		}
		if !found {
			t.Fatal("July activity not found in July filter")
		}
	})

	// --- UC-1002: Conflict detection via check-date ---

	h.Run("UC-1002a", "Check date with no activities returns Home", func(c *harness.Client) {
		login(c)
		resp := c.GET("/api/activities/check/2027-01-01")
		c.AssertStatus(resp, 200)
		c.AssertJSONHas(resp, "location", "Home")
		c.AssertJSONHas(resp, "hasConflict", false)
	})

	h.Run("UC-1002b", "Check date shows spanning activity", func(c *harness.Client) {
		login(c)
		c.POST("/api/activities", `{
			"title": "London Trip",
			"type": "travel",
			"startDate": "2026-11-01",
			"endDate": "2026-11-05",
			"location": "London"
		}`)

		resp := c.GET("/api/activities/check/2026-11-03")
		c.AssertStatus(resp, 200)
		c.AssertJSONHas(resp, "location", "London")
		c.AssertJSONHas(resp, "hasConflict", false)

		data := resp.JSON()
		acts := data["activities"].([]interface{})
		if len(acts) != 1 {
			t.Fatalf("expected 1 activity, got %d", len(acts))
		}
	})

	h.Run("UC-1002c", "Check date detects location conflict", func(c *harness.Client) {
		login(c)
		// Trip to Seattle
		c.POST("/api/activities", `{
			"title": "Seattle Trip",
			"type": "travel",
			"startDate": "2026-12-01",
			"endDate": "2026-12-05",
			"location": "Seattle"
		}`)
		// Local commitment on a day within the trip
		c.POST("/api/activities", `{
			"title": "Dentist Appointment",
			"type": "commitment",
			"startDate": "2026-12-03",
			"location": "Home"
		}`)

		resp := c.GET("/api/activities/check/2026-12-03")
		c.AssertStatus(resp, 200)
		c.AssertJSONHas(resp, "hasConflict", true)

		data := resp.JSON()
		acts := data["activities"].([]interface{})
		if len(acts) != 2 {
			t.Fatalf("expected 2 activities on conflict date, got %d", len(acts))
		}
	})

	// --- Health ---

	h.Run("UC-0011", "Health endpoint returns ok", func(c *harness.Client) {
		resp := c.GET("/health")
		c.AssertStatus(resp, 200)
		c.AssertJSONHas(resp, "status", "ok")
	})
}
