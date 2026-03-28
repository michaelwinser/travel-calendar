package app

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/michaelwinser/appbase/server"
)

type displayDay struct {
	Date     string
	Weekday  string
	Day      int
	Month    string
	Location string
	Type     string
	Color    string
	IsToday  bool
	IsPast   bool
	TripName string
}

// HandleDisplay serves a server-rendered HTML display optimized for
// embedded devices (reTerminal, e-ink displays). No JavaScript required.
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

	now := time.Now()
	today := now.Format("2006-01-02")
	from := now.AddDate(0, 0, -7).Format("2006-01-02")
	to := now.AddDate(0, 0, 21).Format("2006-01-02")

	activities, err := s.store.ListRange(p.UserID, from, to)
	if err != nil {
		server.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	tripMap := s.buildTripMap(p.UserID)

	// Current location
	currentLoc := "Home"
	currentTrip := ""
	for _, a := range activities {
		if a.StartDate <= today && a.EndDate >= today {
			if a.Location != "" {
				currentLoc = a.Location
			}
			if t, ok := tripMap[a.TripID]; ok {
				currentTrip = t.Name
			}
			break
		}
	}

	// Build 4 weeks starting from the previous Sunday
	startDate := now.AddDate(0, 0, -int(now.Weekday()))
	var weeks [][]displayDay
	for week := 0; week < 4; week++ {
		var days []displayDay
		for dow := 0; dow < 7; dow++ {
			d := startDate.AddDate(0, 0, week*7+dow)
			dateStr := d.Format("2006-01-02")

			di := displayDay{
				Date:    dateStr,
				Weekday: d.Format("Mon"),
				Day:     d.Day(),
				Month:   d.Format("Jan"),
				IsToday: dateStr == today,
				IsPast:  dateStr < today,
			}

			for _, a := range activities {
				if a.StartDate <= dateStr && a.EndDate >= dateStr {
					if a.Location != "" {
						di.Location = a.Location
					}
					di.Type = a.Type
					di.Color = displayColor(a.Type, theme)
					if t, ok := tripMap[a.TripID]; ok {
						di.TripName = t.Name
					}
					break
				}
			}

			days = append(days, di)
		}
		weeks = append(weeks, days)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=1800")
	renderDisplay(w, handle, theme, currentLoc, currentTrip, now, weeks)
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

func renderDisplay(w http.ResponseWriter, handle, theme, currentLoc, currentTrip string, now time.Time, weeks [][]displayDay) {
	bg := "#fff"
	fg := "#000"
	muted := "#888"
	todayBg := "#e8e8ff"
	pastOp := "0.4"

	if theme == "eink" {
		todayBg = "#eee"
		muted = "#666"
	}

	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=800">
<title>Where is %s?</title>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: sans-serif; background: %s; color: %s; width: 800px; height: 480px; overflow: hidden; }
.header { display: flex; justify-content: space-between; align-items: center; padding: 12px 16px; border-bottom: 2px solid %s; }
.title { font-size: 20px; font-weight: 700; }
.now { text-align: right; font-size: 14px; }
.now-loc { font-size: 18px; font-weight: 700; }
.now-trip { font-size: 13px; color: %s; }
.now-time { font-size: 12px; color: %s; }
.cal { width: 100%%; border-collapse: collapse; table-layout: fixed; }
.cal th { font-size: 11px; color: %s; padding: 6px 2px; text-align: center; font-weight: 500; text-transform: uppercase; }
.cal td { height: 80px; border: 1px solid #e0e0e0; vertical-align: top; padding: 3px 5px; font-size: 12px; }
.cal td.today { background: %s; outline: 2px solid #0000ff; outline-offset: -2px; }
.cal td.past { opacity: %s; }
.day-num { font-size: 13px; font-weight: 600; margin-bottom: 2px; }
.month-lbl { font-size: 10px; color: %s; font-weight: 700; }
.loc { font-size: 11px; margin-top: 1px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.bar { height: 4px; border-radius: 2px; margin-top: 2px; }
.trip { font-size: 10px; color: %s; font-style: italic; }
</style>
</head>
<body>
<div class="header">
  <div class="title">Where is %s?</div>
  <div class="now">
    <div class="now-loc">%s</div>`, handle, bg, fg, fg, muted, muted, muted, todayBg, pastOp, muted, muted, handle, currentLoc)

	if currentTrip != "" {
		fmt.Fprintf(w, `
    <div class="now-trip">%s</div>`, currentTrip)
	}

	fmt.Fprintf(w, `
    <div class="now-time">%s</div>
  </div>
</div>
<table class="cal">
<thead><tr>
  <th>Sun</th><th>Mon</th><th>Tue</th><th>Wed</th><th>Thu</th><th>Fri</th><th>Sat</th>
</tr></thead>
<tbody>`, now.Format("Mon Jan 2, 3:04 PM"))

	for _, week := range weeks {
		fmt.Fprint(w, "<tr>")
		for _, d := range week {
			cls := ""
			if d.IsToday {
				cls = " today"
			} else if d.IsPast {
				cls = " past"
			}

			fmt.Fprintf(w, `<td class="%s">`, cls)

			// Day number + month label on 1st
			if d.Day == 1 {
				fmt.Fprintf(w, `<div class="day-num"><span class="month-lbl">%s</span> %d</div>`, d.Month, d.Day)
			} else {
				fmt.Fprintf(w, `<div class="day-num">%d</div>`, d.Day)
			}

			if d.Location != "" {
				fmt.Fprintf(w, `<div class="loc">%s</div>`, d.Location)
			}

			if d.Color != "" {
				fmt.Fprintf(w, `<div class="bar" style="background:%s"></div>`, d.Color)
			}

			if d.TripName != "" {
				fmt.Fprintf(w, `<div class="trip">%s</div>`, d.TripName)
			}

			fmt.Fprint(w, "</td>")
		}
		fmt.Fprint(w, "</tr>")
	}

	fmt.Fprint(w, `</tbody></table>
</body>
</html>`)
}
