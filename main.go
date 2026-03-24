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
	"bufio"
	"context"
	"embed"
	"fmt"
	"io/fs"
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

//go:embed frontend/dist/*
var frontendDist embed.FS

const appName = "travel-calendar"
const cliName = "travel"

var (
	app          *appbase.App
	activities   *ActivityStore
	trips        *TripStore
	parseHistory *ParseHistoryStore
)

func setup() error {
	var err error
	cfg := appbase.Config{
		Name:      appName,
		Quiet:     !appcli.IsServeCommand,
		LocalMode: appcli.IsLocalMode,
	}
	if appcli.LocalDataPath != "" {
		cfg.DB.SQLitePath = appcli.LocalDataPath + "/app.db"
	}
	app, err = appbase.New(cfg)
	if err != nil {
		return err
	}
	activities, err = NewActivityStore(app.DB())
	if err != nil {
		return err
	}
	trips, err = NewTripStore(app.DB())
	if err != nil {
		return err
	}
	parseHistory, err = NewParseHistoryStore(app.DB())
	if err != nil {
		return err
	}

	// Register API routes
	activityServer := &ActivityServer{store: activities, trips: trips, parseHistory: parseHistory}
	api.HandlerFromMux(activityServer, app.Server().Router())

	return nil
}

func main() {
	cli := appcli.New("travel", "Travel calendar — plan trips, detect conflicts, stay sane", setup)

	cli.SetServeFunc(func() error {
		r := app.Router()

		// Serve the Svelte SPA for authenticated users, login page otherwise.
		distFS, err := fs.Sub(frontendDist, "frontend/dist")
		if err != nil {
			return fmt.Errorf("embedding frontend: %w", err)
		}
		fileServer := http.FileServer(http.FS(distFS))

		// Serve static assets (JS, CSS) directly
		r.Handle("/assets/*", fileServer)

		// Root: login page if unauthenticated, SPA if authenticated
		r.Get("/*", app.LoginPage(func(w http.ResponseWriter, r *http.Request) {
			// Serve index.html for all routes (SPA)
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
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

	updateCmd := &cobra.Command{
		Use:   "update [id-prefix]",
		Short: "Update an activity (via API)",
		Args:  cobra.ExactArgs(1),
		RunE:  updateActivity,
	}
	updateCmd.Flags().String("title", "", "New title")
	updateCmd.Flags().String("from", "", "New start date (YYYY-MM-DD)")
	updateCmd.Flags().String("to", "", "New end date (YYYY-MM-DD)")
	updateCmd.Flags().String("loc", "", "New location")
	updateCmd.Flags().String("type", "", "New activity type")
	updateCmd.Flags().String("notes", "", "New notes")
	cli.AddCommand(updateCmd)

	quickCmd := &cobra.Command{
		Use:   "quick [text]",
		Short: "Parse freeform text into an activity, then create/edit/abort",
		Args:  cobra.MinimumNArgs(1),
		RunE:  quickAdd,
	}
	quickCmd.Flags().BoolP("yes", "y", false, "Skip confirmation and create immediately")
	cli.AddCommand(quickCmd)

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

func apiClient(cmd *cobra.Command) (client *api.ClientWithResponses, cleanup func(), err error) {
	if err := setup(); err != nil {
		return nil, nil, err
	}

	httpClient, baseURL, stop, err := appcli.ClientForCommand(cmd, cliName, app.Handler())
	if err != nil {
		return nil, nil, err
	}

	c, err := api.NewClientWithResponses(baseURL, api.WithHTTPClient(httpClient))
	if err != nil {
		stop()
		return nil, nil, err
	}
	return c, stop, nil
}

func addActivity(cmd *cobra.Command, args []string) error {
	client, cleanup, err := apiClient(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

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

func quickAdd(cmd *cobra.Command, args []string) error {
	client, cleanup, err := apiClient(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

	text := strings.Join(args, " ")
	yes, _ := cmd.Flags().GetBool("yes")

	// Parse via API
	parseResp, err := client.ParseActivityWithResponse(context.Background(), api.ParseRequest{Text: text})
	if err != nil {
		return err
	}
	if parseResp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", parseResp.StatusCode(), string(parseResp.Body))
	}

	result := parseResp.JSON200
	parsed := result.Activity
	historyID := result.Id

	// Display proposed activity
	title := deref(parsed.Title, "(none)")
	actType := "stay"
	if parsed.Type != nil {
		actType = string(*parsed.Type)
	}
	startDate := "(none)"
	if parsed.StartDate != nil {
		startDate = parsed.StartDate.Format("2006-01-02")
	}
	endDate := startDate
	if parsed.EndDate != nil {
		endDate = parsed.EndDate.Format("2006-01-02")
	}
	location := deref(parsed.Location, "(none)")

	fmt.Println()
	fmt.Printf("  Title:     %s\n", title)
	fmt.Printf("  Type:      %s\n", actType)
	if startDate == endDate {
		fmt.Printf("  Date:      %s\n", startDate)
	} else {
		fmt.Printf("  Dates:     %s → %s\n", startDate, endDate)
	}
	fmt.Printf("  Location:  %s\n", location)

	if result.Unparsed != "" {
		fmt.Printf("  Unparsed:  %s\n", result.Unparsed)
	}
	fmt.Println()

	if !yes {
		action := prompt("[C]reate  [E]dit  [A]bort? ")
		switch strings.ToLower(strings.TrimSpace(action)) {
		case "c", "":
			// fall through to create
		case "e":
			title, actType, startDate, endDate, location = editFields(title, actType, startDate, endDate, location)
		case "a":
			fmt.Println("Aborted.")
			return nil
		default:
			fmt.Println("Aborted.")
			return nil
		}
	}

	// Validate we have enough to create
	if title == "(none)" || title == "" {
		return fmt.Errorf("title is required")
	}
	if startDate == "(none)" || startDate == "" {
		return fmt.Errorf("start date is required")
	}

	// Build create request
	sd, _ := time.Parse("2006-01-02", startDate)
	req := api.CreateActivityRequest{
		Title:     title,
		Type:      api.CreateActivityRequestType(actType),
		StartDate: openapi_types.Date{Time: sd},
	}
	if endDate != "" && endDate != startDate {
		ed, _ := time.Parse("2006-01-02", endDate)
		req.EndDate = &openapi_types.Date{Time: ed}
	}
	if location != "(none)" && location != "" {
		req.Location = &location
	}
	if historyID != "" {
		req.ParseHistoryId = &historyID
	}

	createResp, err := client.CreateActivityWithResponse(context.Background(), req)
	if err != nil {
		return err
	}
	if createResp.StatusCode() != http.StatusCreated {
		return fmt.Errorf("server returned %d: %s", createResp.StatusCode(), string(createResp.Body))
	}

	a := createResp.JSON201
	loc := ""
	if a.Location != nil {
		loc = *a.Location
	}
	fmt.Printf("Created: %s (%s)\n", a.Title, a.Id[:8])
	fmt.Printf("  %s → %s  [%s]  %s\n", a.StartDate.Format("2006-01-02"), a.EndDate.Format("2006-01-02"), a.Type, loc)
	return nil
}

func editFields(title, actType, startDate, endDate, location string) (string, string, string, string, string) {
	reader := bufio.NewReader(os.Stdin)

	title = promptDefault(reader, "  Title", title)
	actType = promptDefault(reader, "  Type", actType)
	startDate = promptDefault(reader, "  Start date", startDate)
	endDate = promptDefault(reader, "  End date", endDate)
	location = promptDefault(reader, "  Location", location)
	fmt.Println()

	return title, actType, startDate, endDate, location
}

func promptDefault(reader *bufio.Reader, label, defaultVal string) string {
	fmt.Printf("%s [%s]: ", label, defaultVal)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal
	}
	return line
}

func prompt(msg string) string {
	fmt.Print(msg)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func deref(s *string, fallback string) string {
	if s == nil || *s == "" {
		return fallback
	}
	return *s
}

func listActivities(cmd *cobra.Command, args []string) error {
	client, cleanup, err := apiClient(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

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
	client, cleanup, err := apiClient(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

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

func resolveByPrefix(client *api.ClientWithResponses, prefix string) (*api.Activity, error) {
	resp, err := client.ListActivitiesWithResponse(context.Background(), &api.ListActivitiesParams{})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode(), string(resp.Body))
	}

	var matches []api.Activity
	for _, a := range *resp.JSON200 {
		if strings.HasPrefix(a.Id, prefix) {
			matches = append(matches, a)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no activity found matching %q", prefix)
	}
	if len(matches) > 1 {
		fmt.Fprintf(os.Stderr, "Ambiguous prefix %q matches %d activities:\n", prefix, len(matches))
		for _, a := range matches {
			fmt.Fprintf(os.Stderr, "  %s  %s\n", a.Id[:8], a.Title)
		}
		return nil, fmt.Errorf("provide a longer prefix")
	}
	return &matches[0], nil
}

func updateActivity(cmd *cobra.Command, args []string) error {
	client, cleanup, err := apiClient(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

	a, err := resolveByPrefix(client, args[0])
	if err != nil {
		return err
	}

	req := api.UpdateActivityRequest{}
	if v, _ := cmd.Flags().GetString("title"); v != "" {
		req.Title = &v
	}
	if v, _ := cmd.Flags().GetString("type"); v != "" {
		t := api.UpdateActivityRequestType(v)
		req.Type = &t
	}
	if v, _ := cmd.Flags().GetString("from"); v != "" {
		d, perr := time.Parse("2006-01-02", v)
		if perr != nil {
			return fmt.Errorf("invalid start date %q (expected YYYY-MM-DD)", v)
		}
		req.StartDate = &openapi_types.Date{Time: d}
	}
	if v, _ := cmd.Flags().GetString("to"); v != "" {
		d, perr := time.Parse("2006-01-02", v)
		if perr != nil {
			return fmt.Errorf("invalid end date %q (expected YYYY-MM-DD)", v)
		}
		req.EndDate = &openapi_types.Date{Time: d}
	}
	if v, _ := cmd.Flags().GetString("loc"); v != "" {
		req.Location = &v
	}
	if v, _ := cmd.Flags().GetString("notes"); v != "" {
		req.Notes = &v
	}

	resp, err := client.UpdateActivityWithResponse(context.Background(), a.Id, req)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode(), string(resp.Body))
	}

	updated := resp.JSON200
	loc := ""
	if updated.Location != nil {
		loc = *updated.Location
	}
	fmt.Printf("Updated: %s (%s)\n", updated.Title, updated.Id[:8])
	fmt.Printf("  %s → %s  [%s]  %s\n", updated.StartDate.Format("2006-01-02"), updated.EndDate.Format("2006-01-02"), updated.Type, loc)
	return nil
}

func deleteActivity(cmd *cobra.Command, args []string) error {
	client, cleanup, err := apiClient(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

	a, err := resolveByPrefix(client, args[0])
	if err != nil {
		return err
	}

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
