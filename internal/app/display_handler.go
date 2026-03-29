package app

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/michaelwinser/appbase/server"
)

type displayDay struct {
	Date       string
	Day        int
	Month      string
	Location   string
	Type       string
	Color      string
	IsToday    bool
	IsPast     bool
	TripName   string
	TripColor  string
	ActivityLabel string // e.g. "UA 16 EWR → CDG"
}

type tripBar struct {
	Name     string
	Color    string
	StartCol int
	EndCol   int
}

type displayWeek struct {
	Days     [7]displayDay
	TripBars []tripBar
}

func (s *ActivityServer) HandleDisplay(w http.ResponseWriter, r *http.Request) {
	handle := r.PathValue("handle")
	if handle == "" {
		handle = strings.TrimPrefix(r.URL.Path, "/public/")
		handle = strings.TrimSuffix(handle, "/display")
	}

	p, err := s.publicProfiles.GetByHandle(handle)
	if err != nil || p == nil || !p.Enabled {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("<html><body style='background:#fff;color:#000;font-family:sans-serif;padding:2rem'><p>Profile not found.</p></body></html>"))
		return
	}

	theme := r.URL.Query().Get("theme")
	if theme == "" {
		theme = "desk"
	}
	showHeader := r.URL.Query().Get("header") == "true"

	numWeeks := 6
	if w := r.URL.Query().Get("weeks"); w != "" {
		if n, err := fmt.Sscanf(w, "%d", &numWeeks); n == 1 && err == nil && numWeeks >= 2 && numWeeks <= 12 {
			// valid
		} else {
			numWeeks = 6
		}
	}

	now := time.Now()
	today := now.Format("2006-01-02")
	from := now.AddDate(0, 0, -7).Format("2006-01-02")
	to := now.AddDate(0, 0, numWeeks*7).Format("2006-01-02")

	activities, err := s.store.ListRange(p.UserID, from, to)
	if err != nil {
		server.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	tripMap := s.buildTripMap(p.UserID)

	// Compute trip spans
	type tripSpan struct {
		name      string
		startDate string
		endDate   string
		location  string
	}
	tripSpans := map[string]*tripSpan{}
	for _, a := range activities {
		if a.TripID == "" {
			continue
		}
		t, ok := tripMap[a.TripID]
		if !ok {
			continue
		}
		span, exists := tripSpans[a.TripID]
		if !exists {
			span = &tripSpan{name: t.Name, startDate: a.StartDate, endDate: a.EndDate}
			tripSpans[a.TripID] = span
		}
		if a.StartDate < span.startDate {
			span.startDate = a.StartDate
		}
		if a.EndDate > span.endDate {
			span.endDate = a.EndDate
		}
		if a.Location != "" {
			span.location = a.Location
		}
	}

	// Date → tripSpan lookup
	tripByDate := map[string]*tripSpan{}
	for _, span := range tripSpans {
		sd, _ := time.Parse("2006-01-02", span.startDate)
		ed, _ := time.Parse("2006-01-02", span.endDate)
		for d := sd; !d.After(ed); d = d.AddDate(0, 0, 1) {
			tripByDate[d.Format("2006-01-02")] = span
		}
	}

	// Current location
	currentLoc := "Home"
	currentTrip := ""
	if span, ok := tripByDate[today]; ok {
		currentLoc = span.location
		currentTrip = span.name
	}
	for _, a := range activities {
		if a.StartDate <= today && a.EndDate >= today && a.Location != "" {
			currentLoc = a.Location
			break
		}
	}

	// Build 6 weeks
	startDate := now.AddDate(0, 0, -int(now.Weekday()))

	var weeks []displayWeek
	for week := 0; week < numWeeks; week++ {
		wd := displayWeek{}
		for dow := 0; dow < 7; dow++ {
			d := startDate.AddDate(0, 0, week*7+dow)
			dateStr := d.Format("2006-01-02")

			dd := displayDay{
				Date:    dateStr,
				Day:     d.Day(),
				Month:   d.Format("Jan"),
				IsToday: dateStr == today,
				IsPast:  dateStr < today,
			}

			// Activity on this day
			for _, a := range activities {
				if a.StartDate <= dateStr && a.EndDate >= dateStr {
					if a.Location != "" {
						dd.Location = a.Location
					}
					dd.Type = a.Type
					dd.Color = displayColor(a.Type, theme)
					// Build activity label for travel types
					if a.Type == "travel" {
						dd.ActivityLabel = a.Title
					}
					break
				}
			}

			wd.Days[dow] = dd
		}

		// Compute trip bars for this week
		weekStart := startDate.AddDate(0, 0, week*7).Format("2006-01-02")
		weekEnd := startDate.AddDate(0, 0, week*7+6).Format("2006-01-02")

		seen := map[string]bool{}
		for _, span := range tripSpans {
			if span.endDate < weekStart || span.startDate > weekEnd {
				continue
			}
			if seen[span.name] {
				continue
			}
			seen[span.name] = true

			startCol := 0
			if span.startDate > weekStart {
				sd, _ := time.Parse("2006-01-02", span.startDate)
				ws, _ := time.Parse("2006-01-02", weekStart)
				startCol = int(sd.Sub(ws).Hours() / 24)
			}
			endCol := 6
			if span.endDate < weekEnd {
				ed, _ := time.Parse("2006-01-02", span.endDate)
				ws, _ := time.Parse("2006-01-02", weekStart)
				endCol = int(ed.Sub(ws).Hours() / 24)
			}

			label := span.name
			if span.location != "" {
				label += " in " + span.location
			}

			wd.TripBars = append(wd.TripBars, tripBar{
				Name:     label,
				Color:    "#0000ff",
				StartCol: startCol,
				EndCol:   endCol,
			})
		}

		weeks = append(weeks, wd)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=1800")
	renderFlexDisplay(w, handle, theme, showHeader, numWeeks, currentLoc, currentTrip, now, weeks)
}

func displayColor(actType, theme string) string {
	if theme == "eink" {
		return "#000"
	}
	switch actType {
	case "travel":
		return "#0000ff"
	case "stay":
		return "#008800"
	case "conference":
		return "#ff0000"
	case "vacation":
		return "#ccaa00"
	case "commitment":
		return "#ff0000"
	default:
		return "#000"
	}
}

func renderFlexDisplay(w http.ResponseWriter, handle, theme string, showHeader bool, numWeeks int, currentLoc, currentTrip string, now time.Time, weeks []displayWeek) {
	// Compute row height to fill 480px
	headerHeight := 0
	if showHeader {
		headerHeight = 30
	}
	dowHeight := 24
	available := 480 - headerHeight - dowHeight
	rowHeight := available / numWeeks

	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=800">
<title>Where is %s?</title>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: sans-serif; background: #fff; color: #000; width: 800px; height: 480px; overflow: hidden; }
.header { display: flex; justify-content: space-between; align-items: center; padding: 4px 12px; background: #000; color: #fff; height: %dpx; }
.title { font-size: 14px; font-weight: 700; }
.now-loc { font-size: 13px; font-weight: 700; }
.dow-row { display: flex; background: #000; color: #fff; height: %dpx; }
.dow-cell { flex: 1; text-align: center; font-size: 13px; font-weight: 600; line-height: %dpx; }
.week { position: relative; display: flex; border-bottom: 1px solid #ccc; height: %dpx; overflow: hidden; }
.day { flex: 1; border-right: 1px solid #ddd; padding: 2px 3px; overflow: hidden; }
.day:last-child { border-right: none; }
.day-date { display: flex; justify-content: space-between; align-items: center; }
.day-num { font-size: 18px; font-weight: 700; }
.day-num.today { background: #ff0000; color: #fff; border-radius: 50%%; width: 24px; height: 24px; line-height: 24px; text-align: center; font-size: 16px; display: inline-block; }
.month-lbl { font-size: 12px; font-weight: 700; color: #000; }
.activities { }
.act-label { font-size: 11px; color: #000; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.loc { font-size: 11px; color: #000; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.trip-bar { position: absolute; height: 26px; background: #0000ff; color: #fff; font-size: 15px; font-weight: 700; line-height: 26px; padding: 0 6px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; z-index: 10; }
</style>
</head>
<body>`, handle, headerHeight, dowHeight, dowHeight, rowHeight)

	if showHeader {
		fmt.Fprintf(w, `<div class="header"><div class="title">%s</div><div class="now-loc">%s</div></div>`, handle, currentLoc)
	}

	// Day-of-week header
	fmt.Fprint(w, `<div class="dow-row">`)
	for _, d := range []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"} {
		fmt.Fprintf(w, `<div class="dow-cell">%s</div>`, d)
	}
	fmt.Fprint(w, `</div>`)

	dayWidth := 100.0 / 7.0

	for _, week := range weeks {
		fmt.Fprint(w, `<div class="week">`)

		// Trip bars (absolutely positioned)
		for _, bar := range week.TripBars {
			left := float64(bar.StartCol) * dayWidth
			width := float64(bar.EndCol-bar.StartCol+1) * dayWidth
			// Position below the day numbers
			fmt.Fprintf(w, `<div class="trip-bar" style="left:%.4f%%;width:%.4f%%;top:28px;background:%s">%s</div>`,
				left, width, bar.Color, bar.Name)
		}

		// Day cells
		for _, d := range week.Days {
			fmt.Fprint(w, `<div class="day">`)

			// Date row
			fmt.Fprint(w, `<div class="day-date">`)
			if d.Day == 1 {
				fmt.Fprintf(w, `<span class="month-lbl">%s</span>`, d.Month)
			} else {
				fmt.Fprint(w, `<span></span>`)
			}
			if d.IsToday {
				fmt.Fprintf(w, `<span class="day-num today">%d</span>`, d.Day)
			} else {
				fmt.Fprintf(w, `<span class="day-num">%d</span>`, d.Day)
			}
			fmt.Fprint(w, `</div>`)

			// Activities (normal flow, trip bar overlays on top via z-index)
			fmt.Fprint(w, `<div class="activities">`)
			if d.ActivityLabel != "" {
				fmt.Fprintf(w, `<div class="act-label">%s</div>`, d.ActivityLabel)
			}
			if d.Location != "" && d.TripName == "" {
				fmt.Fprintf(w, `<div class="loc">%s</div>`, d.Location)
			}
			fmt.Fprint(w, `</div>`)

			fmt.Fprint(w, `</div>`)
		}

		fmt.Fprint(w, `</div>`)
	}

	fmt.Fprint(w, `</body></html>`)
}
