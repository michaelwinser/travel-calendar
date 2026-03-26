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
	"encoding/csv"
	"encoding/json"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	icalparser "github.com/michaelwinser/travel-calendar/internal/ical"

	"github.com/spf13/cobra"

	"github.com/michaelwinser/appbase"
	appcli "github.com/michaelwinser/appbase/cli"
	travelapp "github.com/michaelwinser/travel-calendar/internal/app"
	"github.com/michaelwinser/travel-calendar/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

//go:embed frontend/dist/*
var frontendDist embed.FS

const appName = "travel-calendar"
const cliName = "travel"

var app *appbase.App
var activityServer *travelapp.ActivityServer

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
	activities, err := travelapp.NewActivityStore(app.DB())
	if err != nil {
		return err
	}
	trips, err := travelapp.NewTripStore(app.DB())
	if err != nil {
		return err
	}
	parseHistory, err := travelapp.NewParseHistoryStore(app.DB())
	if err != nil {
		return err
	}
	shareLinks, err := travelapp.NewShareLinkStore(app.DB())
	if err != nil {
		return err
	}
	shares, err := travelapp.NewShareStore(app.DB())
	if err != nil {
		return err
	}
	publicProfiles, err := travelapp.NewPublicProfileStore(app.DB())
	if err != nil {
		return err
	}
	places, err := travelapp.NewPlaceStore(app.DB())
	if err != nil {
		return err
	}

	// Register API routes
	activityServer = travelapp.NewActivityServer(activities, trips, parseHistory, shareLinks, shares, publicProfiles, places)
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

		// Shared calendar JSON endpoint (unauthenticated — outside /api/ prefix)
		// The SPA serves at /shared/{token} and fetches data from /shared/{token}.json
		r.Get("/shared/{token}.json", activityServer.HandleSharedCalendar)

		// Public dashboard (unauthenticated — outside /api/ prefix)
		r.Get("/public/{handle}.json", activityServer.HandlePublicDashboard)

		// Public dashboard frontend (separate entry point, no login required)
		r.Get("/public/*", func(w http.ResponseWriter, r *http.Request) {
			r.URL.Path = "/public.html"
			fileServer.ServeHTTP(w, r)
		})

		// Serve static assets (JS, CSS) directly
		r.Handle("/assets/*", fileServer)

		// Shared calendar view: separate entry point, no login required
		r.Get("/shared/*", func(w http.ResponseWriter, r *http.Request) {
			r.URL.Path = "/shared.html"
			fileServer.ServeHTTP(w, r)
		})

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
	addCmd.Flags().String("type", "stay", fmt.Sprintf("Activity type (%s)", strings.Join(travelapp.ValidTypes, ", ")))
	addCmd.Flags().String("notes", "", "Optional notes")
	addCmd.MarkFlagRequired("from")
	cli.AddCommand(addCmd)

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List activities (via API)",
		RunE:  listActivities,
	}
	listCmd.Flags().String("month", "", "Filter by month (e.g. 2026-04)")
	listCmd.Flags().String("user", "", "View another user's shared calendar (email)")
	listCmd.Flags().Bool("conflicts", false, "Show conflict indicators")
	listCmd.Flags().Bool("json", false, "Output as JSON")
	cli.AddCommand(listCmd)

	checkCmd := &cobra.Command{
		Use:   "check [date]",
		Short: "Check what's on a specific date (via API)",
		Args:  cobra.ExactArgs(1),
		RunE:  checkDate,
	}
	checkCmd.Flags().String("user", "", "Check another user's shared calendar (email)")
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

	// --- Trip commands ---

	tripCmd := &cobra.Command{
		Use:   "trip",
		Short: "Manage trips",
	}

	tripListCmd := &cobra.Command{
		Use:   "list",
		Short: "List trips",
		RunE:  listTripsCmd,
	}
	tripCmd.AddCommand(tripListCmd)
	cli.AddCommand(tripCmd)

	// --- Share link commands ---

	shareLinkCmd := &cobra.Command{
		Use:   "share-link",
		Short: "Manage share links",
	}

	shareLinkCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new share link",
		RunE:  createShareLink,
	}
	shareLinkCreateCmd.Flags().String("label", "", "Friendly name for this link")
	shareLinkCreateCmd.Flags().String("expires", "", "Expiry date-time (RFC3339, e.g. 2026-06-01T00:00:00Z)")
	shareLinkCreateCmd.Flags().String("from", "", "Only share activities from this date (YYYY-MM-DD)")
	shareLinkCreateCmd.Flags().String("to", "", "Only share activities to this date (YYYY-MM-DD)")
	shareLinkCreateCmd.Flags().String("trip-ids", "", "Comma-separated trip IDs to include")
	shareLinkCreateCmd.Flags().Bool("show-titles", false, "Include activity titles in shared view")
	shareLinkCmd.AddCommand(shareLinkCreateCmd)

	shareLinkListCmd := &cobra.Command{
		Use:   "list",
		Short: "List share links",
		RunE:  listShareLinks,
	}
	shareLinkCmd.AddCommand(shareLinkListCmd)

	shareLinkDeleteCmd := &cobra.Command{
		Use:   "delete [id-prefix]",
		Short: "Revoke a share link",
		Args:  cobra.ExactArgs(1),
		RunE:  deleteShareLink,
	}
	shareLinkCmd.AddCommand(shareLinkDeleteCmd)

	cli.AddCommand(shareLinkCmd)

	// --- User-to-user share commands ---

	shareCmd := &cobra.Command{
		Use:   "share",
		Short: "Manage user-to-user sharing",
	}

	shareAddCmd := &cobra.Command{
		Use:   "add [email]",
		Short: "Share your calendar with a user",
		Args:  cobra.ExactArgs(1),
		RunE:  shareAdd,
	}
	shareAddCmd.Flags().Bool("show-titles", false, "Include activity titles in shared view")
	shareCmd.AddCommand(shareAddCmd)

	shareListCmd := &cobra.Command{
		Use:   "list",
		Short: "List shares you've created",
		RunE:  shareList,
	}
	shareCmd.AddCommand(shareListCmd)

	shareRemoveCmd := &cobra.Command{
		Use:   "remove [id-prefix]",
		Short: "Revoke a share",
		Args:  cobra.ExactArgs(1),
		RunE:  shareRemove,
	}
	shareCmd.AddCommand(shareRemoveCmd)

	shareListSharedCmd := &cobra.Command{
		Use:   "list-shared",
		Short: "List calendars shared with you",
		RunE:  shareListShared,
	}
	shareCmd.AddCommand(shareListSharedCmd)

	cli.AddCommand(shareCmd)

	// --- Public profile commands ---

	publicCmd := &cobra.Command{
		Use:   "public",
		Short: "Manage your public dashboard",
	}

	publicEnableCmd := &cobra.Command{
		Use:   "enable",
		Short: "Enable your public dashboard",
		RunE:  publicEnable,
	}
	publicEnableCmd.Flags().String("handle", "", "URL slug (required, e.g. michael)")
	publicEnableCmd.MarkFlagRequired("handle")
	publicCmd.AddCommand(publicEnableCmd)

	publicDisableCmd := &cobra.Command{
		Use:   "disable",
		Short: "Disable your public dashboard",
		RunE:  publicDisable,
	}
	publicCmd.AddCommand(publicDisableCmd)

	publicStatusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show public dashboard status",
		RunE:  publicStatus,
	}
	publicCmd.AddCommand(publicStatusCmd)

	cli.AddCommand(publicCmd)

	// --- Places commands ---

	placesCmd := &cobra.Command{
		Use:   "places",
		Short: "List your places",
		RunE:  listPlacesCmd,
	}

	placesShowCmd := &cobra.Command{
		Use:   "show [name-or-id]",
		Short: "Show place details",
		Args:  cobra.ExactArgs(1),
		RunE:  showPlaceCmd,
	}
	placesCmd.AddCommand(placesShowCmd)


	cli.AddCommand(placesCmd)

	// --- Named locations ---

	namedCmd := &cobra.Command{
		Use:   "named",
		Short: "Manage named locations",
	}

	namedListCmd := &cobra.Command{
		Use:   "list",
		Short: "List named locations and unresolved activity locations",
		RunE:  namedList,
	}
	namedCmd.AddCommand(namedListCmd)

	namedSetCmd := &cobra.Command{
		Use:   "set [name] [location]",
		Short: "Create or update a named location",
		Args:  cobra.ExactArgs(2),
		RunE:  namedSet,
	}
	namedCmd.AddCommand(namedSetCmd)

	namedClearCmd := &cobra.Command{
		Use:   "clear [name]",
		Short: "Remove a named location",
		Args:  cobra.ExactArgs(1),
		RunE:  namedClear,
	}
	namedCmd.AddCommand(namedClearCmd)

	namedBackfillCmd := &cobra.Command{
		Use:   "backfill",
		Short: "Auto-name unresolved activity locations via gazetteer",
		RunE:  placesBackfill,
	}
	namedBackfillCmd.Flags().Bool("dry-run", false, "Show what would be linked without making changes")
	namedCmd.AddCommand(namedBackfillCmd)

	cli.AddCommand(namedCmd)

	// --- Import/Export ---

	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export all data (activities, trips, places)",
		RunE:  exportData,
	}
	exportCmd.Flags().String("format", "json", "Export format: json or csv")
	exportCmd.Flags().StringP("output", "o", "", "Output file (default: stdout)")
	cli.AddCommand(exportCmd)

	importCmd := &cobra.Command{
		Use:   "import [file]",
		Short: "Import data from a file",
		Args:  cobra.ExactArgs(1),
		RunE:  importData,
	}
	importCmd.Flags().Bool("dry-run", false, "Preview what would be imported")
	importCmd.Flags().String("mode", "merge", "Import mode: merge (skip duplicates) or replace (wipe and restore)")
	cli.AddCommand(importCmd)

	importCalCmd := &cobra.Command{
		Use:   "import-cal [url]",
		Short: "Import events from a public iCal/ICS calendar URL",
		Args:  cobra.ExactArgs(1),
		RunE:  importCalendar,
	}
	importCalCmd.Flags().Bool("dry-run", false, "Preview events without importing")
	importCalCmd.Flags().Bool("all-day-only", false, "Only import all-day events (skip timed events)")
	importCalCmd.Flags().String("type", "stay", fmt.Sprintf("Activity type for imported events (%s)", strings.Join(travelapp.ValidTypes, ", ")))
	importCalCmd.Flags().String("trip", "", "Assign imported activities to this trip")
	cli.AddCommand(importCalCmd)

	infoCmd := &cobra.Command{
		Use:   "info",
		Short: "Show current configuration and connection info",
		RunE:  showInfo,
	}
	cli.AddCommand(infoCmd)

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
		// Resolve location to a place
		placeID := resolveLocationToPlace(client, loc)
		if placeID != "" {
			req.PlaceId = &placeID
		}
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
	user, _ := cmd.Flags().GetString("user")

	// --user flag: query another user's shared calendar
	if user != "" {
		return listSharedActivities(client, user, month)
	}

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

	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(items)
	}

	if len(items) == 0 {
		fmt.Println("No activities. Add one with: travel add \"Trip\" --from 2026-04-01 --loc Paris")
		return nil
	}

	showConflicts, _ := cmd.Flags().GetBool("conflicts")

	// Build conflict map: activity ID → has conflict
	conflictIDs := map[string]bool{}
	if showConflicts {
		conflictIDs = checkConflictsForActivities(client, items)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if showConflicts {
		fmt.Fprintf(w, "ID\tDATES\tTYPE\tLOCATION\tTITLE\t\n")
	} else {
		fmt.Fprintf(w, "ID\tDATES\tTYPE\tLOCATION\tTITLE\n")
	}
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
		marker := ""
		if showConflicts && conflictIDs[a.Id] {
			marker = " !"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s%s\n", a.Id[:8], dates, a.Type, loc, a.Title, marker)
	}
	w.Flush()
	return nil
}

// checkConflictsForActivities calls checkDate for each unique date across
// the activities and returns a set of activity IDs that participate in conflicts.
func checkConflictsForActivities(client *api.ClientWithResponses, items []api.Activity) map[string]bool {
	// Collect unique dates from all activities
	dates := map[string]bool{}
	for _, a := range items {
		start := a.StartDate.Format("2006-01-02")
		end := a.EndDate.Format("2006-01-02")
		// Add start, end, and a sample of days in between for multi-day activities
		d := a.StartDate.Time
		for !d.After(a.EndDate.Time) {
			dates[d.Format("2006-01-02")] = true
			d = d.AddDate(0, 0, 1)
		}
		_ = start
		_ = end
	}

	// Check each date for conflicts
	conflictDates := map[string]bool{}
	for dateStr := range dates {
		d, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		checkResp, err := client.CheckDateWithResponse(context.Background(), openapi_types.Date{Time: d})
		if err != nil || checkResp.StatusCode() != http.StatusOK || checkResp.JSON200 == nil {
			continue
		}
		if checkResp.JSON200.HasConflict {
			conflictDates[dateStr] = true
		}
	}

	// Map back to activity IDs
	result := map[string]bool{}
	for _, a := range items {
		d := a.StartDate.Time
		for !d.After(a.EndDate.Time) {
			if conflictDates[d.Format("2006-01-02")] {
				result[a.Id] = true
				break
			}
			d = d.AddDate(0, 0, 1)
		}
	}
	return result
}

func listSharedActivities(client *api.ClientWithResponses, email, month string) error {
	params := &api.GetSharedActivitiesParams{}
	if month != "" {
		params.Month = &month
	}

	resp, err := client.GetSharedActivitiesWithResponse(context.Background(), email, params)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode(), string(resp.Body))
	}

	cal := resp.JSON200
	fmt.Printf("Shared by: %s\n\n", email)

	if len(cal.Activities) == 0 {
		fmt.Println("No activities.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "DATES\tTYPE\tLOCATION\tTITLE\tTRIP\n")
	for _, a := range cal.Activities {
		dates := a.StartDate.Format("2006-01-02")
		endStr := a.EndDate.Format("2006-01-02")
		if endStr != dates {
			dates = dates + " -> " + endStr
		}
		loc := ""
		if a.Location != nil {
			loc = *a.Location
		}
		title := ""
		if a.Title != nil {
			title = *a.Title
		}
		trip := ""
		if a.TripName != nil {
			trip = *a.TripName
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", dates, a.Type, loc, title, trip)
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

	user, _ := cmd.Flags().GetString("user")

	// --user flag: check another user's shared calendar for this date
	if user != "" {
		from := openapi_types.Date{Time: d}
		to := openapi_types.Date{Time: d}
		params := &api.GetSharedActivitiesParams{From: &from, To: &to}
		resp, err := client.GetSharedActivitiesWithResponse(context.Background(), user, params)
		if err != nil {
			return err
		}
		if resp.StatusCode() != http.StatusOK {
			return fmt.Errorf("server returned %d: %s", resp.StatusCode(), string(resp.Body))
		}
		cal := resp.JSON200
		fmt.Printf("%s (%s's calendar):\n", dateStr, user)
		for _, a := range cal.Activities {
			loc := ""
			if a.Location != nil {
				loc = "  @ " + *a.Location
			}
			title := string(a.Type)
			if a.Title != nil {
				title = *a.Title
			}
			fmt.Printf("  - %s [%s]%s\n", title, a.Type, loc)
		}
		if len(cal.Activities) == 0 {
			fmt.Println("  No activities")
		}
		return nil
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

// --- Trip CLI commands ---

func listTripsCmd(cmd *cobra.Command, args []string) error {
	client, cleanup, err := apiClient(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

	resp, err := client.ListTripsWithResponse(context.Background())
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode(), string(resp.Body))
	}

	trips := *resp.JSON200
	if len(trips) == 0 {
		fmt.Println("No trips.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tNAME\tDATES\tLOCATIONS\tACTIVITIES\n")
	for _, t := range trips {
		dates := t.StartDate.Format("2006-01-02") + " -> " + t.EndDate.Format("2006-01-02")
		locs := ""
		if t.Locations != nil {
			locs = strings.Join(*t.Locations, ", ")
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\n", t.Id[:8], t.Name, dates, locs, t.ActivityCount)
	}
	w.Flush()
	return nil
}

// --- Share link CLI commands ---

func createShareLink(cmd *cobra.Command, args []string) error {
	client, cleanup, err := apiClient(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

	req := api.CreateShareLinkRequest{}
	if v, _ := cmd.Flags().GetString("label"); v != "" {
		req.Label = &v
	}
	if v, _ := cmd.Flags().GetString("expires"); v != "" {
		t, perr := time.Parse(time.RFC3339, v)
		if perr != nil {
			return fmt.Errorf("invalid expiry %q (expected RFC3339, e.g. 2026-06-01T00:00:00Z)", v)
		}
		req.ExpiresAt = &t
	}
	if v, _ := cmd.Flags().GetString("from"); v != "" {
		d, perr := time.Parse("2006-01-02", v)
		if perr != nil {
			return fmt.Errorf("invalid from date %q (expected YYYY-MM-DD)", v)
		}
		req.FromDate = &openapi_types.Date{Time: d}
	}
	if v, _ := cmd.Flags().GetString("to"); v != "" {
		d, perr := time.Parse("2006-01-02", v)
		if perr != nil {
			return fmt.Errorf("invalid to date %q (expected YYYY-MM-DD)", v)
		}
		req.ToDate = &openapi_types.Date{Time: d}
	}
	if v, _ := cmd.Flags().GetString("trip-ids"); v != "" {
		req.TripIds = &v
	}
	if v, _ := cmd.Flags().GetBool("show-titles"); v {
		req.ShowTitle = &v
	}

	resp, err := client.CreateShareLinkWithResponse(context.Background(), req)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusCreated {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode(), string(resp.Body))
	}

	link := resp.JSON201
	fmt.Printf("Created share link: %s (%s)\n", link.Label, link.Id[:8])
	fmt.Printf("  Token: %s\n", link.Token)
	fmt.Printf("  URL:   /shared/%s\n", link.Token)
	if link.ExpiresAt != nil {
		fmt.Printf("  Expires: %s\n", link.ExpiresAt.Format(time.RFC3339))
	}
	if link.FromDate != nil {
		fmt.Printf("  From: %s\n", link.FromDate.Format("2006-01-02"))
	}
	if link.ToDate != nil {
		fmt.Printf("  To: %s\n", link.ToDate.Format("2006-01-02"))
	}
	if link.ShowTitle {
		fmt.Println("  Titles: visible")
	}
	return nil
}

func listShareLinks(cmd *cobra.Command, args []string) error {
	client, cleanup, err := apiClient(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

	resp, err := client.ListShareLinksWithResponse(context.Background())
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode(), string(resp.Body))
	}

	links := *resp.JSON200
	if len(links) == 0 {
		fmt.Println("No share links. Create one with: travel share-link create --label \"For team\"")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tLABEL\tTOKEN\tEXPIRES\tTITLES\n")
	for _, l := range links {
		expires := "never"
		if l.ExpiresAt != nil {
			expires = l.ExpiresAt.Format("2006-01-02")
		}
		titles := "hidden"
		if l.ShowTitle {
			titles = "visible"
		}
		tokenPreview := l.Token
		if len(tokenPreview) > 12 {
			tokenPreview = tokenPreview[:12] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", l.Id[:8], l.Label, tokenPreview, expires, titles)
	}
	w.Flush()
	return nil
}

func deleteShareLink(cmd *cobra.Command, args []string) error {
	client, cleanup, err := apiClient(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

	// Resolve by prefix
	listResp, err := client.ListShareLinksWithResponse(context.Background())
	if err != nil {
		return err
	}
	if listResp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", listResp.StatusCode(), string(listResp.Body))
	}

	prefix := args[0]
	var matches []api.ShareLink
	for _, l := range *listResp.JSON200 {
		if strings.HasPrefix(l.Id, prefix) {
			matches = append(matches, l)
		}
	}
	if len(matches) == 0 {
		return fmt.Errorf("no share link found matching %q", prefix)
	}
	if len(matches) > 1 {
		fmt.Fprintf(os.Stderr, "Ambiguous prefix %q matches %d links:\n", prefix, len(matches))
		for _, l := range matches {
			fmt.Fprintf(os.Stderr, "  %s  %s\n", l.Id[:8], l.Label)
		}
		return fmt.Errorf("provide a longer prefix")
	}

	delResp, err := client.DeleteShareLinkWithResponse(context.Background(), matches[0].Id)
	if err != nil {
		return err
	}
	if delResp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", delResp.StatusCode(), string(delResp.Body))
	}

	fmt.Printf("Deleted share link: %s (%s)\n", matches[0].Label, matches[0].Id[:8])
	return nil
}

// --- User-to-user share CLI commands ---

func shareAdd(cmd *cobra.Command, args []string) error {
	client, cleanup, err := apiClient(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

	email := args[0]
	showTitles, _ := cmd.Flags().GetBool("show-titles")

	req := api.CreateShareRequest{Email: email}
	if showTitles {
		req.ShowTitle = &showTitles
	}

	resp, err := client.CreateShareWithResponse(context.Background(), req)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusCreated {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode(), string(resp.Body))
	}

	sh := resp.JSON201
	fmt.Printf("Shared calendar with %s (%s)\n", sh.SharedWith, sh.Id[:8])
	if sh.ShowTitle {
		fmt.Println("  Titles: visible")
	}
	return nil
}

func shareList(cmd *cobra.Command, args []string) error {
	client, cleanup, err := apiClient(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

	resp, err := client.ListSharesWithResponse(context.Background())
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode(), string(resp.Body))
	}

	shares := *resp.JSON200
	if len(shares) == 0 {
		fmt.Println("No shares. Share with: travel share add user@example.com")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tSHARED WITH\tTITLES\n")
	for _, sh := range shares {
		titles := "hidden"
		if sh.ShowTitle {
			titles = "visible"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", sh.Id[:8], sh.SharedWith, titles)
	}
	w.Flush()
	return nil
}

func shareRemove(cmd *cobra.Command, args []string) error {
	client, cleanup, err := apiClient(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

	listResp, err := client.ListSharesWithResponse(context.Background())
	if err != nil {
		return err
	}
	if listResp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", listResp.StatusCode(), string(listResp.Body))
	}

	prefix := args[0]
	var matches []api.Share
	for _, sh := range *listResp.JSON200 {
		if strings.HasPrefix(sh.Id, prefix) {
			matches = append(matches, sh)
		}
	}
	if len(matches) == 0 {
		return fmt.Errorf("no share found matching %q", prefix)
	}
	if len(matches) > 1 {
		fmt.Fprintf(os.Stderr, "Ambiguous prefix %q matches %d shares:\n", prefix, len(matches))
		for _, sh := range matches {
			fmt.Fprintf(os.Stderr, "  %s  %s\n", sh.Id[:8], sh.SharedWith)
		}
		return fmt.Errorf("provide a longer prefix")
	}

	delResp, err := client.DeleteShareWithResponse(context.Background(), matches[0].Id)
	if err != nil {
		return err
	}
	if delResp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", delResp.StatusCode(), string(delResp.Body))
	}

	fmt.Printf("Revoked share with %s (%s)\n", matches[0].SharedWith, matches[0].Id[:8])
	return nil
}

func shareListShared(cmd *cobra.Command, args []string) error {
	client, cleanup, err := apiClient(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

	resp, err := client.ListSharedWithMeWithResponse(context.Background())
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode(), string(resp.Body))
	}

	entries := *resp.JSON200
	if len(entries) == 0 {
		fmt.Println("No calendars shared with you.")
		return nil
	}

	fmt.Println("Calendars shared with you:")
	for _, e := range entries {
		fmt.Printf("  %s\n", e.OwnerEmail)
	}
	fmt.Println("\nView with: travel list --user <email>")
	return nil
}

// --- Public profile CLI commands ---

func publicEnable(cmd *cobra.Command, args []string) error {
	client, cleanup, err := apiClient(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

	handle, _ := cmd.Flags().GetString("handle")

	resp, err := client.UpdatePublicProfileWithResponse(context.Background(), api.UpdatePublicProfileRequest{
		Handle:  handle,
		Enabled: true,
	})
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode(), string(resp.Body))
	}

	fmt.Printf("Public dashboard enabled at /public/%s\n", resp.JSON200.Handle)
	return nil
}

func publicDisable(cmd *cobra.Command, args []string) error {
	client, cleanup, err := apiClient(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

	// Get current profile first to preserve handle
	getResp, err := client.GetPublicProfileWithResponse(context.Background())
	if err != nil {
		return err
	}
	if getResp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", getResp.StatusCode(), string(getResp.Body))
	}

	handle := getResp.JSON200.Handle
	if handle == "" {
		fmt.Println("No public profile configured.")
		return nil
	}

	resp, err := client.UpdatePublicProfileWithResponse(context.Background(), api.UpdatePublicProfileRequest{
		Handle:  handle,
		Enabled: false,
	})
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode(), string(resp.Body))
	}

	fmt.Println("Public dashboard disabled.")
	return nil
}

func publicStatus(cmd *cobra.Command, args []string) error {
	client, cleanup, err := apiClient(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

	resp, err := client.GetPublicProfileWithResponse(context.Background())
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode(), string(resp.Body))
	}

	p := resp.JSON200
	if p.Handle == "" {
		fmt.Println("Public dashboard: not configured")
		fmt.Println("Enable with: travel public enable --handle <your-handle>")
		return nil
	}

	status := "disabled"
	if p.Enabled {
		status = "enabled"
	}
	fmt.Printf("Public dashboard: %s\n", status)
	fmt.Printf("  Handle: %s\n", p.Handle)
	fmt.Printf("  URL:    /public/%s\n", p.Handle)
	return nil
}

// --- Places CLI commands ---

func listPlacesCmd(cmd *cobra.Command, args []string) error {
	client, cleanup, err := apiClient(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

	resp, err := client.ListPlacesWithResponse(context.Background())
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode(), string(resp.Body))
	}

	places := *resp.JSON200
	if len(places) == 0 {
		fmt.Println("No places. They are created automatically when you use --loc.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tNAME\tCITY\tCOUNTRY\tTIMEZONE\tKIND\n")
	for _, p := range places {
		city := ""
		if p.City != nil {
			city = *p.City
		}
		country := ""
		if p.Country != nil {
			country = *p.Country
		}
		tz := ""
		if p.Timezone != nil {
			tz = *p.Timezone
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", p.Id[:8], p.Name, city, country, tz, p.Kind)
	}
	w.Flush()
	return nil
}

func showPlaceCmd(cmd *cobra.Command, args []string) error {
	client, cleanup, err := apiClient(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

	// List all and match by name or id prefix
	listResp, err := client.ListPlacesWithResponse(context.Background())
	if err != nil {
		return err
	}
	if listResp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", listResp.StatusCode(), string(listResp.Body))
	}

	query := strings.ToLower(args[0])
	var match *api.Place
	for _, p := range *listResp.JSON200 {
		if strings.HasPrefix(p.Id, args[0]) || strings.ToLower(p.Name) == query {
			match = &p
			break
		}
	}
	if match == nil {
		return fmt.Errorf("no place found matching %q", args[0])
	}

	fmt.Printf("Name:      %s\n", match.Name)
	fmt.Printf("ID:        %s\n", match.Id)
	fmt.Printf("Kind:      %s\n", match.Kind)
	if match.Aliases != nil && len(*match.Aliases) > 0 {
		fmt.Printf("Aliases:   %s\n", strings.Join(*match.Aliases, ", "))
	}
	if match.City != nil {
		fmt.Printf("City:      %s\n", *match.City)
	}
	if match.Country != nil {
		fmt.Printf("Country:   %s\n", *match.Country)
	}
	if match.Latitude != nil && match.Longitude != nil {
		fmt.Printf("Location:  %.4f, %.4f\n", *match.Latitude, *match.Longitude)
	}
	if match.Timezone != nil {
		fmt.Printf("Timezone:  %s\n", *match.Timezone)
	}
	return nil
}

// resolveLocationToPlace calls the resolve API and either returns an existing place ID
// or creates a new place from the best gazetteer match. Returns "" if resolution fails.
func resolveLocationToPlace(client *api.ClientWithResponses, loc string) string {
	resolveResp, err := client.ResolvePlacesWithResponse(context.Background(), api.PlaceResolveRequest{Text: loc})
	if err != nil || resolveResp.StatusCode() != http.StatusOK || resolveResp.JSON200 == nil {
		return ""
	}

	result := resolveResp.JSON200

	// If there's an exact match in user's places, use it
	if result.Exact != nil {
		fmt.Printf("  Place: %s (existing)\n", result.Exact.Name)
		return result.Exact.Id
	}

	// Look for the best suggestion to create a place from
	if len(result.Suggestions) == 0 {
		return ""
	}

	best := result.Suggestions[0]

	// If it's a user place suggestion, use it directly
	if best.Source == api.User && best.Place != nil {
		fmt.Printf("  Place: %s (matched)\n", best.Place.Name)
		return best.Place.Id
	}

	// Create a new place from the gazetteer suggestion
	req := api.CreatePlaceRequest{
		Name: loc, // Preserve user's original input as the place name
		City: cityWithAdmin1(best),
	}
	if best.Country != nil {
		req.Country = best.Country
	}
	if best.Latitude != nil {
		req.Latitude = best.Latitude
	}
	if best.Longitude != nil {
		req.Longitude = best.Longitude
	}
	if best.Timezone != nil {
		req.Timezone = best.Timezone
	}
	kind := api.CreatePlaceRequestKindCity
	req.Kind = &kind

	createResp, err := client.CreatePlaceWithResponse(context.Background(), req)
	if err != nil || createResp.StatusCode() != http.StatusCreated || createResp.JSON201 == nil {
		return ""
	}

	p := createResp.JSON201
	extra := ""
	if p.Country != nil {
		extra = " (" + *p.Country
		if p.Timezone != nil {
			extra += ", " + *p.Timezone
		}
		extra += ")"
	}
	fmt.Printf("  Place: %s%s (new)\n", p.Name, extra)
	return p.Id
}

// --- Places backfill ---

func placesBackfill(cmd *cobra.Command, args []string) error {
	client, cleanup, err := apiClient(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Fetch all activities
	resp, err := client.ListActivitiesWithResponse(context.Background(), &api.ListActivitiesParams{})
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode(), string(resp.Body))
	}

	// Find activities without placeId, group by location string
	type group struct {
		location   string
		activities []api.Activity
	}
	groups := map[string]*group{}
	var unlinkedCount int

	for _, a := range *resp.JSON200 {
		if a.PlaceId != nil && *a.PlaceId != "" {
			continue // already linked
		}
		loc := ""
		if a.Location != nil {
			loc = *a.Location
		}
		if loc == "" {
			continue
		}
		unlinkedCount++
		if g, ok := groups[loc]; ok {
			g.activities = append(g.activities, a)
		} else {
			groups[loc] = &group{location: loc, activities: []api.Activity{a}}
		}
	}

	if unlinkedCount == 0 {
		fmt.Println("All activities are already linked to places.")
		return nil
	}

	fmt.Printf("Found %d unlinked activities across %d unique locations.\n\n", unlinkedCount, len(groups))

	linked := 0
	skipped := 0
	placesCreated := 0

	for loc, g := range groups {
		// Resolve the location string
		resolveResp, err := client.ResolvePlacesWithResponse(context.Background(), api.PlaceResolveRequest{Text: loc})
		if err != nil || resolveResp.StatusCode() != http.StatusOK || resolveResp.JSON200 == nil {
			fmt.Printf("  %-30s  → resolve failed, skipping\n", loc)
			skipped += len(g.activities)
			continue
		}

		result := resolveResp.JSON200
		var placeID string
		var placeName string

		if result.Exact != nil {
			// Exact match in user's places
			placeID = result.Exact.Id
			placeName = result.Exact.Name + " (existing)"
		} else if len(result.Suggestions) > 0 {
			best := result.Suggestions[0]

			// Only auto-link if confidence is high enough
			if best.Score < 0.7 {
				fmt.Printf("  %-30s  → low confidence (%.0f%%), skipping\n", loc, best.Score*100)
				skipped += len(g.activities)
				continue
			}

			if best.Source == api.User && best.Place != nil {
				placeID = best.Place.Id
				placeName = best.Place.Name + " (matched)"
			} else {
				// Create a new place from gazetteer
				if dryRun {
					extra := ""
					if best.Country != nil {
						extra = ", " + *best.Country
					}
					fmt.Printf("  %-30s  → would create \"%s%s\" and link %d activities\n", loc, best.Name, extra, len(g.activities))
					linked += len(g.activities)
					continue
				}

				req := api.CreatePlaceRequest{
					Name: loc,
					City: &best.Name,
				}
				if best.Country != nil {
					req.Country = best.Country
				}
				if best.Latitude != nil {
					req.Latitude = best.Latitude
				}
				if best.Longitude != nil {
					req.Longitude = best.Longitude
				}
				if best.Timezone != nil {
					req.Timezone = best.Timezone
				}
				kind := api.CreatePlaceRequestKindCity
				req.Kind = &kind

				createResp, err := client.CreatePlaceWithResponse(context.Background(), req)
				if err != nil || createResp.StatusCode() != http.StatusCreated || createResp.JSON201 == nil {
					errMsg := ""
					if err != nil {
						errMsg = err.Error()
					} else if createResp != nil {
						errMsg = fmt.Sprintf("status %d: %s", createResp.StatusCode(), string(createResp.Body))
					}
					fmt.Printf("  %-30s  → failed to create place (%s), skipping\n", loc, errMsg)
					skipped += len(g.activities)
					continue
				}
				placeID = createResp.JSON201.Id
				extra := ""
				if best.Country != nil {
					extra = ", " + *best.Country
				}
				placeName = best.Name + extra + " (new)"
				placesCreated++
			}
		} else {
			fmt.Printf("  %-30s  → no matches, skipping\n", loc)
			skipped += len(g.activities)
			continue
		}

		if dryRun {
			fmt.Printf("  %-30s  → would link %d activities to \"%s\"\n", loc, len(g.activities), placeName)
			linked += len(g.activities)
			continue
		}

		// Link all activities in this group to the place
		for _, a := range g.activities {
			_, err := client.UpdateActivityWithResponse(context.Background(), a.Id, api.UpdateActivityRequest{
				PlaceId: &placeID,
			})
			if err == nil {
				linked++
			} else {
				skipped++
			}
		}

		fmt.Printf("  %-30s  → %s (%d activities)\n", loc, placeName, len(g.activities))
	}

	fmt.Printf("\nDone%s. Linked: %d, Skipped: %d, Places created: %d\n",
		map[bool]string{true: " (dry run)", false: ""}[dryRun],
		linked, skipped, placesCreated)
	return nil
}

// --- Named locations CLI ---

func namedList(cmd *cobra.Command, args []string) error {
	client, cleanup, err := apiClient(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

	// Fetch places and activities
	placesResp, err := client.ListPlacesWithResponse(context.Background())
	if err != nil {
		return err
	}
	if placesResp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", placesResp.StatusCode(), string(placesResp.Body))
	}

	actsResp, err := client.ListActivitiesWithResponse(context.Background(), &api.ListActivitiesParams{})
	if err != nil {
		return err
	}
	if actsResp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", actsResp.StatusCode(), string(actsResp.Body))
	}

	places := *placesResp.JSON200
	activities := *actsResp.JSON200

	// Count activities per place
	placeActivityCount := map[string]int{}
	unresolvedLocs := map[string]int{} // location string → count, for activities with no placeId

	for _, a := range activities {
		if a.PlaceId != nil && *a.PlaceId != "" {
			placeActivityCount[*a.PlaceId]++
		} else if a.Location != nil && *a.Location != "" {
			unresolvedLocs[*a.Location]++
		}
	}

	if len(places) == 0 && len(unresolvedLocs) == 0 {
		fmt.Println("No named locations. Create one with: travel named set \"Home\" \"San Francisco\"")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "NAME\tRESOLVES TO\tACTIVITIES\n")

	for _, p := range places {
		resolvesTo := ""
		parts := []string{}
		if p.City != nil && *p.City != "" {
			parts = append(parts, *p.City)
		}
		if p.Country != nil && *p.Country != "" {
			parts = append(parts, *p.Country)
		}
		if p.Timezone != nil && *p.Timezone != "" {
			parts = append(parts, *p.Timezone)
		}
		if len(parts) > 0 {
			resolvesTo = strings.Join(parts, ", ")
		} else {
			resolvesTo = "(no geo data)"
		}

		count := placeActivityCount[p.Id]
		fmt.Fprintf(w, "%s\t%s\t%d\n", p.Name, resolvesTo, count)
	}

	// Show unresolved locations
	unresolvedCount := 0
	for loc, count := range unresolvedLocs {
		// Skip route-style locations
		if strings.Contains(loc, "→") || strings.Contains(loc, "->") {
			continue
		}
		fmt.Fprintf(w, "%s\t(unresolved)\t%d\n", loc, count)
		unresolvedCount++
	}

	w.Flush()

	if unresolvedCount > 0 {
		fmt.Printf("\n%d unresolved locations. Set them with: travel named set \"<name>\" \"<city>\"\n", unresolvedCount)
	}

	return nil
}

func namedSet(cmd *cobra.Command, args []string) error {
	client, cleanup, err := apiClient(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

	name := args[0]
	location := args[1]

	// Resolve the location against the gazetteer
	resolveResp, err := client.ResolvePlacesWithResponse(context.Background(), api.PlaceResolveRequest{Text: location})
	if err != nil {
		return err
	}
	if resolveResp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", resolveResp.StatusCode(), string(resolveResp.Body))
	}

	result := resolveResp.JSON200

	// Check for exact match
	if result.Exact != nil {
		// Update existing place's name
		fmt.Printf("Named \"%s\" → %s (existing place)\n", name, result.Exact.Name)
		return namedLinkActivities(client, name, result.Exact.Id)
	}

	// Filter to gazetteer suggestions only (user places would be exact matches)
	var gazetteerSugs []api.PlaceSuggestion
	for _, s := range result.Suggestions {
		if s.Source == api.Gazetteer {
			gazetteerSugs = append(gazetteerSugs, s)
		}
	}

	if len(gazetteerSugs) == 0 {
		return fmt.Errorf("no matches found for %q. Try a more specific location.", location)
	}

	// Check for ambiguity: if top two results are close in score, different countries,
	// and neither is clearly dominant by population
	isAmbiguous := false
	if len(gazetteerSugs) >= 2 {
		s0, s1 := gazetteerSugs[0], gazetteerSugs[1]
		sameCountry := s0.Country != nil && s1.Country != nil && *s0.Country == *s1.Country
		scoreDiff := s0.Score - s1.Score
		if !sameCountry && scoreDiff < 0.3 {
			isAmbiguous = true
		}
	}
	if isAmbiguous {
		// Ambiguous — show candidates
		fmt.Println("Multiple matches:")
		for i, s := range gazetteerSugs {
			if i >= 5 {
				break
			}
			fmt.Printf("  %d. %s\n", i+1, formatPlaceSummary(s))
		}
		fmt.Println("Specify a more precise location.")
		return nil
	}

	// Unambiguous — create the place
	best := gazetteerSugs[0]
	req := api.CreatePlaceRequest{
		Name: name,
		City: cityWithAdmin1(best),
	}
	if best.Country != nil {
		req.Country = best.Country
	}
	if best.Latitude != nil {
		req.Latitude = best.Latitude
	}
	if best.Longitude != nil {
		req.Longitude = best.Longitude
	}
	if best.Timezone != nil {
		req.Timezone = best.Timezone
	}
	kind := api.CreatePlaceRequestKindCity
	req.Kind = &kind

	// Add the original name as an alias so "Home" resolves to this place
	aliases := []string{name}
	if strings.ToLower(name) != strings.ToLower(best.Name) {
		aliases = append(aliases, best.Name)
	}
	req.Aliases = &aliases

	createResp, err := client.CreatePlaceWithResponse(context.Background(), req)
	if err != nil {
		return err
	}
	if createResp.StatusCode() != http.StatusCreated {
		return fmt.Errorf("server returned %d: %s", createResp.StatusCode(), string(createResp.Body))
	}

	p := createResp.JSON201
	fmt.Printf("Named \"%s\" → %s\n", name, formatPlaceSummary(best))

	return namedLinkActivities(client, name, p.Id)
}

// namedLinkActivities finds activities whose location matches the name and links them.
// cityWithAdmin1 returns a city string that includes state/province if available.
// e.g. "Westport, CT" or just "Milan" if no admin1.
func cityWithAdmin1(s api.PlaceSuggestion) *string {
	city := s.Name
	if s.Admin1 != nil && *s.Admin1 != "" {
		city += ", " + *s.Admin1
	}
	return &city
}

// formatPlaceSummary produces a readable location string from a suggestion.
// e.g. "Westport, CT, US (America/New_York, pop 26k)"
func formatPlaceSummary(s api.PlaceSuggestion) string {
	parts := []string{s.Name}
	if s.Admin1 != nil && *s.Admin1 != "" {
		parts = append(parts, *s.Admin1)
	}
	if s.Country != nil && *s.Country != "" {
		parts = append(parts, *s.Country)
	}
	result := strings.Join(parts, ", ")

	var meta []string
	if s.Timezone != nil && *s.Timezone != "" {
		meta = append(meta, *s.Timezone)
	}
	if s.Population != nil && *s.Population > 0 {
		meta = append(meta, fmt.Sprintf("pop %dk", *s.Population/1000))
	}
	if len(meta) > 0 {
		result += " (" + strings.Join(meta, ", ") + ")"
	}
	return result
}

func namedLinkActivities(client *api.ClientWithResponses, name, placeID string) error {
	actsResp, err := client.ListActivitiesWithResponse(context.Background(), &api.ListActivitiesParams{})
	if err != nil {
		return nil // non-fatal
	}
	if actsResp.StatusCode() != http.StatusOK {
		return nil
	}

	lowerName := strings.ToLower(name)
	linked := 0
	for _, a := range *actsResp.JSON200 {
		if a.PlaceId != nil && *a.PlaceId != "" {
			continue // already linked
		}
		loc := ""
		if a.Location != nil {
			loc = *a.Location
		}
		if strings.ToLower(loc) == lowerName {
			_, err := client.UpdateActivityWithResponse(context.Background(), a.Id, api.UpdateActivityRequest{
				PlaceId: &placeID,
			})
			if err == nil {
				linked++
			}
		}
	}

	if linked > 0 {
		fmt.Printf("Linked %d activities.\n", linked)
	}
	return nil
}

func namedClear(cmd *cobra.Command, args []string) error {
	client, cleanup, err := apiClient(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

	name := args[0]

	// Find the place by name
	placesResp, err := client.ListPlacesWithResponse(context.Background())
	if err != nil {
		return err
	}
	if placesResp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", placesResp.StatusCode(), string(placesResp.Body))
	}

	lowerName := strings.ToLower(name)
	var match *api.Place
	for _, p := range *placesResp.JSON200 {
		if strings.ToLower(p.Name) == lowerName {
			match = &p
			break
		}
	}
	if match == nil {
		return fmt.Errorf("no named location %q found", name)
	}

	// Unlink activities referencing this place
	actsResp, err := client.ListActivitiesWithResponse(context.Background(), &api.ListActivitiesParams{})
	if err == nil && actsResp.StatusCode() == http.StatusOK {
		empty := ""
		unlinked := 0
		for _, a := range *actsResp.JSON200 {
			if a.PlaceId != nil && *a.PlaceId == match.Id {
				client.UpdateActivityWithResponse(context.Background(), a.Id, api.UpdateActivityRequest{
					PlaceId: &empty,
				})
				unlinked++
			}
		}
		if unlinked > 0 {
			fmt.Printf("Unlinked %d activities.\n", unlinked)
		}
	}

	// Delete the place
	delResp, err := client.DeletePlaceWithResponse(context.Background(), match.Id)
	if err != nil {
		return err
	}
	if delResp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", delResp.StatusCode(), string(delResp.Body))
	}

	fmt.Printf("Removed named location \"%s\".\n", name)
	return nil
}

// --- Export/Import ---

// ExportData is the versioned backup format.
type ExportData struct {
	Version    int              `json:"version"`
	ExportedAt string           `json:"exportedAt"`
	Trips      []ExportTrip     `json:"trips"`
	Places     []ExportPlace    `json:"places"`
	Activities []ExportActivity `json:"activities"`
}

type ExportTrip struct {
	Key   string `json:"key"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type ExportPlace struct {
	Key       string   `json:"key"`
	Name      string   `json:"name"`
	Aliases   []string `json:"aliases,omitempty"`
	City      string   `json:"city,omitempty"`
	Country   string   `json:"country,omitempty"`
	Latitude  float64  `json:"latitude,omitempty"`
	Longitude float64  `json:"longitude,omitempty"`
	Timezone  string   `json:"timezone,omitempty"`
	Kind      string   `json:"kind,omitempty"`
}

type ExportActivity struct {
	Key       string `json:"key"`
	Title     string `json:"title"`
	Type      string `json:"type"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Location  string `json:"location,omitempty"`
	Notes     string `json:"notes,omitempty"`
	TripName  string `json:"tripName,omitempty"`
	PlaceName string `json:"placeName,omitempty"`
}

func exportData(cmd *cobra.Command, args []string) error {
	client, cleanup, err := apiClient(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

	format, _ := cmd.Flags().GetString("format")
	output, _ := cmd.Flags().GetString("output")

	// Fetch all data
	actsResp, err := client.ListActivitiesWithResponse(context.Background(), &api.ListActivitiesParams{})
	if err != nil {
		return err
	}
	if actsResp.StatusCode() != http.StatusOK {
		return fmt.Errorf("activities: %d %s", actsResp.StatusCode(), string(actsResp.Body))
	}

	tripsResp, err := client.ListTripsWithResponse(context.Background())
	if err != nil {
		return err
	}
	if tripsResp.StatusCode() != http.StatusOK {
		return fmt.Errorf("trips: %d %s", tripsResp.StatusCode(), string(tripsResp.Body))
	}

	placesResp, err := client.ListPlacesWithResponse(context.Background())
	if err != nil {
		return err
	}
	if placesResp.StatusCode() != http.StatusOK {
		return fmt.Errorf("places: %d %s", placesResp.StatusCode(), string(placesResp.Body))
	}

	activities := *actsResp.JSON200
	apiTrips := *tripsResp.JSON200
	apiPlaces := *placesResp.JSON200

	// Build lookup maps: ID → name
	tripNames := map[string]string{}
	for _, t := range apiTrips {
		tripNames[t.Id] = t.Name
	}
	placeNames := map[string]string{}
	for _, p := range apiPlaces {
		placeNames[p.Id] = p.Name
	}

	// Build export data
	data := ExportData{
		Version:    2,
		ExportedAt: time.Now().Format(time.RFC3339),
	}

	for _, t := range apiTrips {
		key := "trip/" + travelapp.Slug(t.Name)
		data.Trips = append(data.Trips, ExportTrip{
			Key:   key,
			Name:  t.Name,
			Color: t.Color,
		})
	}

	for _, p := range apiPlaces {
		ep := ExportPlace{
			Key:     "place/" + travelapp.Slug(p.Name),
			Name:    p.Name,
			Kind:    string(p.Kind),
		}
		if p.Aliases != nil {
			ep.Aliases = *p.Aliases
		}
		if p.City != nil {
			ep.City = *p.City
		}
		if p.Country != nil {
			ep.Country = *p.Country
		}
		if p.Latitude != nil {
			ep.Latitude = *p.Latitude
		}
		if p.Longitude != nil {
			ep.Longitude = *p.Longitude
		}
		if p.Timezone != nil {
			ep.Timezone = *p.Timezone
		}
		data.Places = append(data.Places, ep)
	}

	for _, a := range activities {
		startStr := a.StartDate.Format("2006-01-02")
		ea := ExportActivity{
			Key:       string(a.Type) + "/" + startStr + "/" + travelapp.Slug(a.Title),
			Title:     a.Title,
			Type:      string(a.Type),
			StartDate: startStr,
			EndDate:   a.EndDate.Format("2006-01-02"),
		}
		if a.Location != nil {
			ea.Location = *a.Location
		}
		if a.Notes != nil {
			ea.Notes = *a.Notes
		}
		if a.TripId != nil && *a.TripId != "" {
			ea.TripName = tripNames[*a.TripId]
		}
		if a.PlaceId != nil && *a.PlaceId != "" {
			ea.PlaceName = placeNames[*a.PlaceId]
		}
		data.Activities = append(data.Activities, ea)
	}

	// Output
	var w *os.File
	if output != "" {
		w, err = os.Create(output)
		if err != nil {
			return err
		}
		defer w.Close()
	} else {
		w = os.Stdout
	}

	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if err := enc.Encode(data); err != nil {
			return err
		}
	case "csv":
		return exportCSV(w, data)
	default:
		return fmt.Errorf("unknown format %q (use json or csv)", format)
	}

	if output != "" {
		fmt.Fprintf(os.Stderr, "Exported %d activities, %d trips, %d places to %s\n",
			len(data.Activities), len(data.Trips), len(data.Places), output)
	}
	return nil
}

func exportCSV(w *os.File, data ExportData) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	cw.Write([]string{"title", "type", "startDate", "endDate", "location", "notes", "trip"})
	for _, a := range data.Activities {
		cw.Write([]string{a.Title, a.Type, a.StartDate, a.EndDate, a.Location, a.Notes, a.TripName})
	}
	return nil
}

func importData(cmd *cobra.Command, args []string) error {
	client, cleanup, err := apiClient(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	mode, _ := cmd.Flags().GetString("mode")
	filePath := args[0]

	if mode != "merge" && mode != "replace" {
		return fmt.Errorf("invalid mode %q (use merge or replace)", mode)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	// Detect format by extension
	if strings.HasSuffix(filePath, ".csv") {
		return importCSV(client, string(data), dryRun)
	}

	// JSON import
	var export ExportData
	if err := json.Unmarshal(data, &export); err != nil {
		return fmt.Errorf("parsing JSON: %w", err)
	}

	fmt.Printf("Import (%s): %d activities, %d trips, %d places (version %d)\n",
		mode, len(export.Activities), len(export.Trips), len(export.Places), export.Version)

	if dryRun {
		fmt.Println("(dry run — no changes made)")
		return nil
	}

	// In replace mode, delete all existing data first
	if mode == "replace" {
		fmt.Println("Deleting existing data...")
		deleteAllData(client)
	}

	// Build existing key sets for merge dedup
	existingTripKeys := map[string]string{} // key → ID
	existingPlaceKeys := map[string]string{}
	existingActivityKeys := map[string]bool{}
	tripIDs := map[string]string{} // name → ID

	if mode == "merge" {
		// Load existing trips
		if resp, err := client.ListTripsWithResponse(context.Background()); err == nil && resp.JSON200 != nil {
			for _, t := range *resp.JSON200 {
				key := "trip/" + travelapp.Slug(t.Name)
				existingTripKeys[key] = t.Id
				tripIDs[t.Name] = t.Id
			}
		}
		// Load existing places
		if resp, err := client.ListPlacesWithResponse(context.Background()); err == nil && resp.JSON200 != nil {
			for _, p := range *resp.JSON200 {
				key := "place/" + travelapp.Slug(p.Name)
				existingPlaceKeys[key] = p.Id
			}
		}
		// Load existing activities and compute keys
		if resp, err := client.ListActivitiesWithResponse(context.Background(), &api.ListActivitiesParams{}); err == nil && resp.JSON200 != nil {
			for _, a := range *resp.JSON200 {
				key := string(a.Type) + "/" + a.StartDate.Format("2006-01-02") + "/" + travelapp.Slug(a.Title)
				existingActivityKeys[key] = true
			}
		}
	}

	// 1. Import trips
	tripsCreated, tripsSkipped := 0, 0
	for _, t := range export.Trips {
		key := t.Key
		if key == "" {
			key = "trip/" + travelapp.Slug(t.Name)
		}
		if mode == "merge" {
			if id, exists := existingTripKeys[key]; exists {
				tripIDs[t.Name] = id
				tripsSkipped++
				continue
			}
		}
		color := t.Color
		resp, err := client.CreateTripWithResponse(context.Background(), api.CreateTripRequest{
			Name:  t.Name,
			Color: &color,
		})
		if err == nil && resp.StatusCode() == http.StatusCreated && resp.JSON201 != nil {
			tripIDs[t.Name] = resp.JSON201.Id
			tripsCreated++
		}
	}

	// 2. Import places
	placeIDs := map[string]string{} // name → ID
	placesCreated, placesSkipped := 0, 0
	for _, p := range export.Places {
		key := p.Key
		if key == "" {
			key = "place/" + travelapp.Slug(p.Name)
		}
		if mode == "merge" {
			if id, exists := existingPlaceKeys[key]; exists {
				placeIDs[p.Name] = id
				placesSkipped++
				continue
			}
		}
		req := api.CreatePlaceRequest{Name: p.Name}
		if p.City != "" {
			req.City = &p.City
		}
		if p.Country != "" {
			req.Country = &p.Country
		}
		if p.Latitude != 0 {
			req.Latitude = &p.Latitude
		}
		if p.Longitude != 0 {
			req.Longitude = &p.Longitude
		}
		if p.Timezone != "" {
			req.Timezone = &p.Timezone
		}
		if p.Kind != "" {
			k := api.CreatePlaceRequestKind(p.Kind)
			req.Kind = &k
		}
		if len(p.Aliases) > 0 {
			req.Aliases = &p.Aliases
		}
		resp, err := client.CreatePlaceWithResponse(context.Background(), req)
		if err == nil && resp.StatusCode() == http.StatusCreated && resp.JSON201 != nil {
			placeIDs[p.Name] = resp.JSON201.Id
			placesCreated++
		}
	}

	// 3. Import activities
	activitiesCreated, activitiesSkipped := 0, 0
	for _, a := range export.Activities {
		key := a.Key
		if key == "" {
			key = a.Type + "/" + a.StartDate + "/" + travelapp.Slug(a.Title)
		}
		if mode == "merge" && existingActivityKeys[key] {
			activitiesSkipped++
			continue
		}

		startDate, err := time.Parse("2006-01-02", a.StartDate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Skipping %q: invalid date %q\n", a.Title, a.StartDate)
			continue
		}

		req := api.CreateActivityRequest{
			Title:     a.Title,
			Type:      api.CreateActivityRequestType(a.Type),
			StartDate: openapi_types.Date{Time: startDate},
		}
		if a.EndDate != "" && a.EndDate != a.StartDate {
			if ed, err := time.Parse("2006-01-02", a.EndDate); err == nil {
				req.EndDate = &openapi_types.Date{Time: ed}
			}
		}
		if a.Location != "" {
			req.Location = &a.Location
		}
		if a.Notes != "" {
			req.Notes = &a.Notes
		}
		if a.TripName != "" {
			if id, ok := tripIDs[a.TripName]; ok {
				req.TripId = &id
			}
		}
		if a.PlaceName != "" {
			if id, ok := placeIDs[a.PlaceName]; ok {
				req.PlaceId = &id
			}
		}

		resp, err := client.CreateActivityWithResponse(context.Background(), req)
		if err == nil && resp.StatusCode() == http.StatusCreated {
			activitiesCreated++
		} else {
			errMsg := ""
			if err != nil {
				errMsg = err.Error()
			} else if resp != nil {
				errMsg = string(resp.Body)
			}
			fmt.Fprintf(os.Stderr, "  Failed: %q (%s)\n", a.Title, errMsg)
		}
	}

	if mode == "merge" {
		fmt.Printf("Created: %d activities, %d trips, %d places\n", activitiesCreated, tripsCreated, placesCreated)
		fmt.Printf("Skipped: %d activities, %d trips, %d places (already exist)\n", activitiesSkipped, tripsSkipped, placesSkipped)
	} else {
		fmt.Printf("Imported: %d activities, %d trips, %d places\n", activitiesCreated, tripsCreated, placesCreated)
	}
	return nil
}

// deleteAllData removes all activities, trips, and places for a clean replace import.
func deleteAllData(client *api.ClientWithResponses) {
	// Delete activities
	if resp, err := client.ListActivitiesWithResponse(context.Background(), &api.ListActivitiesParams{}); err == nil && resp.JSON200 != nil {
		for _, a := range *resp.JSON200 {
			client.DeleteActivityWithResponse(context.Background(), a.Id)
		}
	}
	// Delete trips
	if resp, err := client.ListTripsWithResponse(context.Background()); err == nil && resp.JSON200 != nil {
		for _, t := range *resp.JSON200 {
			client.DeleteTripWithResponse(context.Background(), t.Id)
		}
	}
	// Delete places
	if resp, err := client.ListPlacesWithResponse(context.Background()); err == nil && resp.JSON200 != nil {
		for _, p := range *resp.JSON200 {
			client.DeletePlaceWithResponse(context.Background(), p.Id)
		}
	}
}

func importCSV(client *api.ClientWithResponses, data string, dryRun bool) error {
	r := csv.NewReader(strings.NewReader(data))

	// Read header
	header, err := r.Read()
	if err != nil {
		return fmt.Errorf("reading CSV header: %w", err)
	}

	// Map column names to indices
	colIdx := map[string]int{}
	for i, h := range header {
		colIdx[strings.ToLower(strings.TrimSpace(h))] = i
	}

	// Require at least title and startDate
	if _, ok := colIdx["title"]; !ok {
		if _, ok := colIdx["startdate"]; !ok {
			return fmt.Errorf("CSV must have at least 'title' and 'startDate' columns")
		}
	}

	col := func(row []string, name string) string {
		if idx, ok := colIdx[name]; ok && idx < len(row) {
			return strings.TrimSpace(row[idx])
		}
		return ""
	}

	// Read all rows
	var rows [][]string
	for {
		record, err := r.Read()
		if err != nil {
			break
		}
		rows = append(rows, record)
	}

	fmt.Printf("CSV: %d rows to import\n", len(rows))
	if dryRun {
		for i, row := range rows {
			if i >= 10 {
				fmt.Printf("  ... and %d more\n", len(rows)-10)
				break
			}
			fmt.Printf("  %s | %s | %s | %s\n", col(row, "title"), col(row, "type"), col(row, "startdate"), col(row, "location"))
		}
		fmt.Println("(dry run — no changes made)")
		return nil
	}

	// Build trip lookup for implicit creation
	tripIDs := map[string]string{}
	existingTrips, _ := client.ListTripsWithResponse(context.Background())
	if existingTrips != nil && existingTrips.JSON200 != nil {
		for _, t := range *existingTrips.JSON200 {
			tripIDs[t.Name] = t.Id
		}
	}

	created := 0
	for _, row := range rows {
		title := col(row, "title")
		actType := col(row, "type")
		startStr := col(row, "startdate")
		endStr := col(row, "enddate")
		location := col(row, "location")
		notes := col(row, "notes")
		tripName := col(row, "trip")

		if title == "" || startStr == "" {
			continue
		}
		if actType == "" {
			actType = "stay"
		}

		startDate, err := time.Parse("2006-01-02", startStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Skipping %q: invalid date %q\n", title, startStr)
			continue
		}

		req := api.CreateActivityRequest{
			Title:     title,
			Type:      api.CreateActivityRequestType(actType),
			StartDate: openapi_types.Date{Time: startDate},
		}

		if endStr != "" && endStr != startStr {
			if ed, err := time.Parse("2006-01-02", endStr); err == nil {
				req.EndDate = &openapi_types.Date{Time: ed}
			}
		}
		if location != "" {
			req.Location = &location
		}
		if notes != "" {
			req.Notes = &notes
		}

		// Implicit trip creation
		if tripName != "" {
			if id, ok := tripIDs[tripName]; ok {
				req.TripId = &id
			} else {
				tripResp, err := client.CreateTripWithResponse(context.Background(), api.CreateTripRequest{Name: tripName})
				if err == nil && tripResp.StatusCode() == http.StatusCreated && tripResp.JSON201 != nil {
					tripIDs[tripName] = tripResp.JSON201.Id
					req.TripId = &tripResp.JSON201.Id
				}
			}
		}

		// Resolve location to place
		if location != "" {
			placeID := resolveLocationToPlace(client, location)
			if placeID != "" {
				req.PlaceId = &placeID
			}
		}

		resp, err := client.CreateActivityWithResponse(context.Background(), req)
		if err == nil && resp.StatusCode() == http.StatusCreated {
			created++
		} else {
			fmt.Fprintf(os.Stderr, "  Failed: %q\n", title)
		}
	}

	fmt.Printf("Imported %d activities.\n", created)
	return nil
}

// --- Info ---

func showInfo(cmd *cobra.Command, args []string) error {
	serverFlag, _ := cmd.Root().PersistentFlags().GetString("server")

	if appcli.IsLocalMode {
		fmt.Println("Mode:     local")
		fmt.Printf("Data:     %s/app.db\n", appcli.LocalDataPath)
	} else {
		serverURL := appcli.ResolveServerURL(serverFlag, cliName)
		fmt.Println("Mode:     remote")
		fmt.Printf("Server:   %s\n", serverURL)
	}

	fmt.Printf("App:      %s\n", appName)
	return nil
}

// --- Calendar import ---

func importCalendar(cmd *cobra.Command, args []string) error {
	client, cleanup, err := apiClient(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

	url := args[0]
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	allDayOnly, _ := cmd.Flags().GetBool("all-day-only")
	actType, _ := cmd.Flags().GetString("type")
	tripName, _ := cmd.Flags().GetString("trip")

	// Fetch the iCal feed
	fmt.Printf("Fetching %s ...\n", url)
	httpResp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("fetching calendar: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != 200 {
		return fmt.Errorf("server returned %d", httpResp.StatusCode)
	}

	// Parse iCal
	events, err := icalparser.Parse(httpResp.Body)
	if err != nil {
		return fmt.Errorf("parsing iCal: %w", err)
	}

	// Filter
	if allDayOnly {
		var filtered []icalparser.Event
		for _, e := range events {
			if e.AllDay {
				filtered = append(filtered, e)
			}
		}
		events = filtered
	}

	if len(events) == 0 {
		fmt.Println("No events found in calendar.")
		return nil
	}

	fmt.Printf("Found %d events.\n\n", len(events))

	// Preview
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "DATES\tTITLE\tLOCATION\n")
	for _, e := range events {
		startStr := e.Start.Format("2006-01-02")
		endStr := e.End.Format("2006-01-02")
		// For all-day events, DTEND is exclusive — subtract a day for display
		if e.AllDay && !e.End.IsZero() && e.End.After(e.Start) {
			endStr = e.End.AddDate(0, 0, -1).Format("2006-01-02")
		}
		dates := startStr
		if endStr != startStr {
			dates = startStr + " -> " + endStr
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", dates, e.Summary, e.Location)
	}
	w.Flush()

	if dryRun {
		fmt.Println("\n(dry run — no changes made)")
		return nil
	}

	fmt.Println()

	// Build existing activity keys for merge dedup
	existingKeys := map[string]bool{}
	if resp, err := client.ListActivitiesWithResponse(context.Background(), &api.ListActivitiesParams{}); err == nil && resp.JSON200 != nil {
		for _, a := range *resp.JSON200 {
			key := string(a.Type) + "/" + a.StartDate.Format("2006-01-02") + "/" + travelapp.Slug(a.Title)
			existingKeys[key] = true
		}
	}

	// Resolve trip if specified
	var tripID string
	if tripName != "" {
		if resp, err := client.ListTripsWithResponse(context.Background()); err == nil && resp.JSON200 != nil {
			for _, t := range *resp.JSON200 {
				if strings.EqualFold(t.Name, tripName) {
					tripID = t.Id
					break
				}
			}
		}
		if tripID == "" {
			// Create the trip
			resp, err := client.CreateTripWithResponse(context.Background(), api.CreateTripRequest{Name: tripName})
			if err == nil && resp.StatusCode() == http.StatusCreated && resp.JSON201 != nil {
				tripID = resp.JSON201.Id
			}
		}
	}

	// Import events
	created, skipped := 0, 0
	for _, e := range events {
		startStr := e.Start.Format("2006-01-02")
		endStr := e.End.Format("2006-01-02")
		// Adjust all-day DTEND (exclusive → inclusive)
		if e.AllDay && !e.End.IsZero() && e.End.After(e.Start) {
			endStr = e.End.AddDate(0, 0, -1).Format("2006-01-02")
		}

		key := actType + "/" + startStr + "/" + travelapp.Slug(e.Summary)
		if existingKeys[key] {
			skipped++
			continue
		}

		req := api.CreateActivityRequest{
			Title:     e.Summary,
			Type:      api.CreateActivityRequestType(actType),
			StartDate: openapi_types.Date{Time: e.Start},
		}
		if endStr != startStr {
			endTime, _ := time.Parse("2006-01-02", endStr)
			req.EndDate = &openapi_types.Date{Time: endTime}
		}
		if e.Location != "" {
			req.Location = &e.Location
			// Try to resolve location to a place
			placeID := resolveLocationToPlace(client, e.Location)
			if placeID != "" {
				req.PlaceId = &placeID
			}
		}
		if tripID != "" {
			req.TripId = &tripID
		}

		resp, err := client.CreateActivityWithResponse(context.Background(), req)
		if err == nil && resp.StatusCode() == http.StatusCreated {
			created++
			existingKeys[key] = true // prevent dupes within the same feed
		} else {
			fmt.Fprintf(os.Stderr, "  Failed: %q\n", e.Summary)
		}
	}

	fmt.Printf("Imported %d events, skipped %d (already exist).\n", created, skipped)
	return nil
}
