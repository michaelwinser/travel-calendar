// travel-calendar v2: a high-velocity planning tool for frequent travelers.
//
// Built on appbase. Run as server or use CLI commands.
//
// Server:
//
//	go run . serve
//
// CLI:
//
//	go run . add "European Summit" --from 2026-04-01 --to 2026-04-05 --loc Brussels --type conference
//	go run . list
//	go run . check 2026-04-03
//	go run . delete <id>
package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/michaelwinser/appbase"
	appcli "github.com/michaelwinser/appbase/cli"
)

var (
	app        *appbase.App
	activities *ActivityStore
)

const cliUser = "cli-user"

func setup() error {
	var err error
	app, err = appbase.New(appbase.Config{Name: "travel-calendar"})
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
		// Web UI will be added later. For now, just the API.
		r := app.Router()
		_ = r // routes will be registered here
		return app.Serve()
	})

	// --- add ---
	addCmd := cli.Command("add", "Add an activity", addActivity)
	addCmd.Args = cobra.MinimumNArgs(1)
	addCmd.Flags().String("from", "", "Start date (YYYY-MM-DD, required)")
	addCmd.Flags().String("to", "", "End date (YYYY-MM-DD, defaults to --from)")
	addCmd.Flags().String("loc", "", "Location (e.g. Brussels, Home)")
	addCmd.Flags().String("type", TypeStay, fmt.Sprintf("Activity type (%s)", strings.Join(ValidTypes, ", ")))
	addCmd.Flags().String("notes", "", "Optional notes")
	addCmd.MarkFlagRequired("from")
	cli.AddCommand(addCmd)

	// --- list ---
	listCmd := cli.Command("list", "List activities", listActivities)
	listCmd.Flags().String("month", "", "Filter by month (e.g. 2026-04)")
	cli.AddCommand(listCmd)

	// --- check ---
	checkCmd := cli.Command("check", "Check what's on a specific date", checkDate)
	checkCmd.Args = cobra.ExactArgs(1)
	cli.AddCommand(checkCmd)

	// --- delete ---
	delCmd := cli.Command("delete", "Delete an activity by ID (prefix match)", deleteActivity)
	delCmd.Args = cobra.ExactArgs(1)
	cli.AddCommand(delCmd)

	cli.Execute()
}

func addActivity(cmd *cobra.Command, args []string) error {
	title := strings.Join(args, " ")
	from, _ := cmd.Flags().GetString("from")
	to, _ := cmd.Flags().GetString("to")
	if to == "" {
		to = from
	}
	loc, _ := cmd.Flags().GetString("loc")
	actType, _ := cmd.Flags().GetString("type")
	notes, _ := cmd.Flags().GetString("notes")

	a, err := activities.Create(cliUser, title, actType, from, to, loc, notes)
	if err != nil {
		return err
	}
	fmt.Printf("Created: %s (%s)\n", a.Title, a.ID[:8])
	fmt.Printf("  %s → %s  [%s]  %s\n", a.StartDate, a.EndDate, a.Type, a.Location)

	// Check for conflicts on these dates
	overlapping, err := activities.ListRange(cliUser, from, to)
	if err != nil {
		return nil // non-fatal
	}
	conflicts := 0
	for _, o := range overlapping {
		if o.ID == a.ID {
			continue
		}
		if conflicts == 0 {
			fmt.Println("\n  ⚠ Overlapping activities:")
		}
		conflicts++
		fmt.Printf("    • %s (%s → %s) [%s]\n", o.Title, o.StartDate, o.EndDate, o.Type)
	}
	return nil
}

func listActivities(cmd *cobra.Command, args []string) error {
	month, _ := cmd.Flags().GetString("month")

	var items []Activity
	var err error

	if month != "" {
		// Parse month as YYYY-MM
		t, perr := time.Parse("2006-01", month)
		if perr != nil {
			return fmt.Errorf("invalid month %q (expected YYYY-MM)", month)
		}
		from := t.Format("2006-01-02")
		to := t.AddDate(0, 1, -1).Format("2006-01-02")
		items, err = activities.ListRange(cliUser, from, to)
	} else {
		items, err = activities.List(cliUser)
	}
	if err != nil {
		return err
	}

	if len(items) == 0 {
		fmt.Println("No activities. Add one with: travel add \"Trip\" --from 2026-04-01 --loc Paris")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tDATES\tTYPE\tLOCATION\tTITLE\n")
	for _, a := range items {
		dates := a.StartDate
		if a.EndDate != a.StartDate {
			dates = a.StartDate + " → " + a.EndDate
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", a.ID[:8], dates, a.Type, a.Location, a.Title)
	}
	w.Flush()
	return nil
}

func checkDate(cmd *cobra.Command, args []string) error {
	date := args[0]
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return fmt.Errorf("invalid date %q (expected YYYY-MM-DD)", date)
	}

	items, err := activities.ForDate(cliUser, date)
	if err != nil {
		return err
	}

	if len(items) == 0 {
		fmt.Printf("%s: Home (no activities)\n", date)
		return nil
	}

	// Determine location from primary activity
	loc := "Home"
	for _, a := range items {
		if a.Location != "" {
			loc = a.Location
			break
		}
	}

	fmt.Printf("%s: %s\n", date, loc)
	for _, a := range items {
		fmt.Printf("  • %s [%s]", a.Title, a.Type)
		if a.Location != "" {
			fmt.Printf("  @ %s", a.Location)
		}
		fmt.Println()
	}

	// Flag conflicts: multiple activities with different locations
	locations := map[string]bool{}
	for _, a := range items {
		if a.Location != "" {
			locations[a.Location] = true
		}
	}
	if len(locations) > 1 {
		fmt.Println("\n  ⚠ Location conflict: activities in multiple locations on this date")
	}

	return nil
}

func deleteActivity(cmd *cobra.Command, args []string) error {
	prefix := args[0]

	// Find by prefix match
	all, err := activities.List(cliUser)
	if err != nil {
		return err
	}

	var matches []Activity
	for _, a := range all {
		if strings.HasPrefix(a.ID, prefix) {
			matches = append(matches, a)
		}
	}

	if len(matches) == 0 {
		return fmt.Errorf("no activity found matching %q", prefix)
	}
	if len(matches) > 1 {
		fmt.Fprintf(os.Stderr, "Ambiguous prefix %q matches %d activities:\n", prefix, len(matches))
		for _, a := range matches {
			fmt.Fprintf(os.Stderr, "  %s  %s\n", a.ID[:8], a.Title)
		}
		return fmt.Errorf("provide a longer prefix")
	}

	a := matches[0]
	if err := activities.Delete(a.ID); err != nil {
		return err
	}
	fmt.Printf("Deleted: %s (%s → %s) [%s]\n", a.Title, a.StartDate, a.EndDate, a.Type)
	return nil
}
