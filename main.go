// travel-calendar v2: a high-velocity planning tool for frequent travelers.
//
// Built on appbase with API-first architecture. The OpenAPI spec (openapi.yaml)
// is the source of truth. Server and client code are generated.
//
// Server:
//
//	go run . serve
//
// CLI (talks to server via HTTP):
//
//	go run . login --server http://localhost:3000
//	go run . add "European Summit" --from 2026-04-01 --to 2026-04-05 --loc Brussels --type conference
//	go run . list [--month 2026-04]
//	go run . check 2026-04-03
//	go run . delete <id-prefix>
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/michaelwinser/appbase"
	appcli "github.com/michaelwinser/appbase/cli"
	"github.com/michaelwinser/travel-calendar/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

const appName = "travel-calendar"

var (
	app        *appbase.App
	activities *ActivityStore
)

func setup() error {
	var err error
	app, err = appbase.New(appbase.Config{Name: appName, Quiet: !appcli.IsServeCommand})
	if err != nil {
		return err
	}
	activities, err = NewActivityStore(app.DB())
	if err != nil {
		return err
	}
	return nil
}

func main() {
	cli := appcli.New("travel", "Travel calendar — plan trips, detect conflicts, stay sane", setup)

	cli.SetServeFunc(func() error {
		activityServer := &ActivityServer{store: activities}
		api.HandlerFromMux(activityServer, app.Server().Router())

		r := app.Router()
		r.Get("/", app.LoginPage(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<!DOCTYPE html>
<html><head><title>Travel Calendar</title></head>
<body style="font-family:system-ui;max-width:800px;margin:2rem auto;padding:0 1rem">
<h1>Travel Calendar</h1>
<p>Signed in as ` + appbase.Email(r) + `. <a href="/api/activities">/api/activities</a></p>
<form method="POST" action="/api/auth/logout"><button>Sign out</button></form>
</body></html>`))
		}))

		return app.Serve()
	})

	// --- CLI commands: all go through the HTTP API ---

	addCmd := &cobra.Command{
		Use:   "add [title]",
		Short: "Add an activity (via API)",
		Args:  cobra.MinimumNArgs(1),
		RunE:  addActivity,
	}
	addCmd.Flags().String("from", "", "Start date (YYYY-MM-DD, required)")
	addCmd.Flags().String("to", "", "End date (YYYY-MM-DD, defaults to --from)")
	addCmd.Flags().String("loc", "", "Location (e.g. Brussels, Home)")
	addCmd.Flags().String("type", "stay", fmt.Sprintf("Activity type (%s)", strings.Join(ValidTypes, ", ")))
	addCmd.Flags().String("notes", "", "Optional notes")
	addCmd.MarkFlagRequired("from")
	cli.AddCommand(addCmd)

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List activities (via API)",
		RunE:  listActivities,
	}
	listCmd.Flags().String("month", "", "Filter by month (e.g. 2026-04)")
	cli.AddCommand(listCmd)

	checkCmd := &cobra.Command{
		Use:   "check [date]",
		Short: "Check what's on a specific date (via API)",
		Args:  cobra.ExactArgs(1),
		RunE:  checkDate,
	}
	cli.AddCommand(checkCmd)

	delCmd := &cobra.Command{
		Use:   "delete [id-prefix]",
		Short: "Delete an activity by ID prefix (via API)",
		Args:  cobra.ExactArgs(1),
		RunE:  deleteActivity,
	}
	cli.AddCommand(delCmd)

	cli.Execute()
}

// --- CLI command implementations ---

func apiClient(cmd *cobra.Command) (*api.ClientWithResponses, error) {
	serverFlag, _ := cmd.Flags().GetString("server")
	serverURL := appcli.ResolveServerURL(serverFlag, appName)

	httpClient, err := appcli.AuthenticatedClient(appName)
	if err != nil {
		return nil, fmt.Errorf("not logged in — run: travel login --server %s", serverURL)
	}

	return api.NewClientWithResponses(serverURL, api.WithHTTPClient(httpClient))
}

func addActivity(cmd *cobra.Command, args []string) error {
	client, err := apiClient(cmd)
	if err != nil {
		return err
	}

	title := strings.Join(args, " ")
	from, _ := cmd.Flags().GetString("from")
	to, _ := cmd.Flags().GetString("to")
	loc, _ := cmd.Flags().GetString("loc")
	actType, _ := cmd.Flags().GetString("type")
	notes, _ := cmd.Flags().GetString("notes")

	startDate, perr := time.Parse("2006-01-02", from)
	if perr != nil {
		return fmt.Errorf("invalid start date %q (expected YYYY-MM-DD)", from)
	}

	req := api.CreateActivityRequest{
		Title:     title,
		Type:      api.CreateActivityRequestType(actType),
		StartDate: openapi_types.Date{Time: startDate},
	}
	if to != "" {
		endDate, perr := time.Parse("2006-01-02", to)
		if perr != nil {
			return fmt.Errorf("invalid end date %q (expected YYYY-MM-DD)", to)
		}
		req.EndDate = &openapi_types.Date{Time: endDate}
	}
	if loc != "" {
		req.Location = &loc
	}
	if notes != "" {
		req.Notes = &notes
	}

	resp, err := client.CreateActivityWithResponse(context.Background(), req)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusCreated {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode(), string(resp.Body))
	}

	a := resp.JSON201
	fmt.Printf("Created: %s (%s)\n", a.Title, a.Id[:8])
	endStr := a.EndDate.Format("2006-01-02")
	startStr := a.StartDate.Format("2006-01-02")
	locStr := ""
	if a.Location != nil {
		locStr = *a.Location
	}
	fmt.Printf("  %s → %s  [%s]  %s\n", startStr, endStr, a.Type, locStr)

	// Check for conflicts
	checkResp, err := client.CheckDateWithResponse(context.Background(), a.StartDate)
	if err == nil && checkResp.StatusCode() == http.StatusOK && checkResp.JSON200 != nil {
		check := checkResp.JSON200
		conflicts := 0
		for _, o := range check.Activities {
			if o.Id == a.Id {
				continue
			}
			if conflicts == 0 {
				fmt.Println("\n  Overlapping activities:")
			}
			conflicts++
			oLoc := ""
			if o.Location != nil {
				oLoc = *o.Location
			}
			fmt.Printf("    - %s (%s to %s) [%s] %s\n", o.Title, o.StartDate.Format("2006-01-02"), o.EndDate.Format("2006-01-02"), o.Type, oLoc)
		}
	}
	return nil
}

func listActivities(cmd *cobra.Command, args []string) error {
	client, err := apiClient(cmd)
	if err != nil {
		return err
	}

	month, _ := cmd.Flags().GetString("month")
	params := &api.ListActivitiesParams{}
	if month != "" {
		params.Month = &month
	}

	resp, err := client.ListActivitiesWithResponse(context.Background(), params)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode(), string(resp.Body))
	}

	items := *resp.JSON200
	if len(items) == 0 {
		fmt.Println("No activities. Add one with: travel add \"Trip\" --from 2026-04-01 --loc Paris")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tDATES\tTYPE\tLOCATION\tTITLE\n")
	for _, a := range items {
		dates := a.StartDate.Format("2006-01-02")
		endStr := a.EndDate.Format("2006-01-02")
		if endStr != dates {
			dates = dates + " -> " + endStr
		}
		loc := ""
		if a.Location != nil {
			loc = *a.Location
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", a.Id[:8], dates, a.Type, loc, a.Title)
	}
	w.Flush()
	return nil
}

func checkDate(cmd *cobra.Command, args []string) error {
	client, err := apiClient(cmd)
	if err != nil {
		return err
	}

	dateStr := args[0]
	d, perr := time.Parse("2006-01-02", dateStr)
	if perr != nil {
		return fmt.Errorf("invalid date %q (expected YYYY-MM-DD)", dateStr)
	}

	resp, err := client.CheckDateWithResponse(context.Background(), openapi_types.Date{Time: d})
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode(), string(resp.Body))
	}

	check := resp.JSON200
	fmt.Printf("%s: %s\n", check.Date.Format("2006-01-02"), check.Location)
	for _, a := range check.Activities {
		loc := ""
		if a.Location != nil {
			loc = "  @ " + *a.Location
		}
		fmt.Printf("  - %s [%s]%s\n", a.Title, a.Type, loc)
	}
	if check.HasConflict {
		fmt.Println("\n  Location conflict: activities in multiple locations on this date")
	}
	return nil
}

func deleteActivity(cmd *cobra.Command, args []string) error {
	client, err := apiClient(cmd)
	if err != nil {
		return err
	}

	prefix := args[0]

	// List all to find by prefix
	resp, err := client.ListActivitiesWithResponse(context.Background(), &api.ListActivitiesParams{})
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode(), string(resp.Body))
	}

	var matches []api.Activity
	for _, a := range *resp.JSON200 {
		if strings.HasPrefix(a.Id, prefix) {
			matches = append(matches, a)
		}
	}

	if len(matches) == 0 {
		return fmt.Errorf("no activity found matching %q", prefix)
	}
	if len(matches) > 1 {
		fmt.Fprintf(os.Stderr, "Ambiguous prefix %q matches %d activities:\n", prefix, len(matches))
		for _, a := range matches {
			fmt.Fprintf(os.Stderr, "  %s  %s\n", a.Id[:8], a.Title)
		}
		return fmt.Errorf("provide a longer prefix")
	}

	a := matches[0]
	delResp, err := client.DeleteActivityWithResponse(context.Background(), a.Id)
	if err != nil {
		return err
	}
	if delResp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", delResp.StatusCode(), string(delResp.Body))
	}

	fmt.Printf("Deleted: %s (%s to %s) [%s]\n", a.Title, a.StartDate.Format("2006-01-02"), a.EndDate.Format("2006-01-02"), a.Type)
	return nil
}
