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
	showHeader := r.URL.Query().Get("header") == "true"

	now := time.Now()
	today := now.Format("2006-01-02")
	from := now.AddDate(0, 0, -7).Format("2006-01-02")
	to := now.AddDate(0, 0, 42).Format("2006-01-02")

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

	// Build 6 weeks starting from the previous Sunday
	startDate := now.AddDate(0, 0, -int(now.Weekday()))
	var weeks [][]displayDay
	for week := 0; week < 6; week++ {
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
	renderDisplay(w, handle, theme, showHeader, currentLoc, currentTrip, now, weeks)
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

func renderDisplay(w http.ResponseWriter, handle, theme string, showHeader bool, currentLoc, currentTrip string, now time.Time, weeks [][]displayDay) {
	todayBg := "#e8e8ff"
	pastOp := "0.35"
	if theme == "eink" {
		todayBg = "#ddd"
	}

	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=800">
<title>Where is %s?</title>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: sans-serif; background: #fff; color: #000; width: 800px; overflow: hidden; }
.header { display: flex; justify-content: space-between; align-items: center; padding: 8px 12px; border-bottom: 2px solid #000; }
.title { font-size: 18px; font-weight: 700; }
.now { text-align: right; }
.now-loc { font-size: 16px; font-weight: 700; }
.now-trip { font-size: 12px; color: #666; }
.cal { width: 100%%; border-collapse: collapse; table-layout: fixed; }
.cal th { font-size: 13px; color: #666; padding: 4px 2px; text-align: center; font-weight: 600; text-transform: uppercase; }
.cal td { border: 1px solid #ccc; vertical-align: top; padding: 2px 4px; height: 68px; }
.cal td.today { background: %s; outline: 2px solid #0000ff; outline-offset: -2px; }
.cal td.past { opacity: %s; }
.day-num { font-size: 15px; font-weight: 700; }
.month-lbl { font-size: 11px; color: #666; }
.loc { font-size: 13px; margin-top: 1px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.bar { height: 5px; border-radius: 2px; margin-top: 2px; }
.trip { font-size: 12px; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
</style>
</head>
<body>`, handle, todayBg, pastOp)

	if showHeader {
		fmt.Fprintf(w, `<div class="header">
  <div class="title">Where is %s?</div>
  <div class="now">
    <div class="now-loc">%s</div>`, handle, currentLoc)
		if currentTrip != "" {
			fmt.Fprintf(w, `<div class="now-trip">%s</div>`, currentTrip)
		}
		fmt.Fprint(w, `</div></div>`)
	}

	fmt.Fprint(w, `<table class="cal">
<thead><tr>
  <th>Sun</th><th>Mon</th><th>Tue</th><th>Wed</th><th>Thu</th><th>Fri</th><th>Sat</th>
</tr></thead>
<tbody>`)

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

			if d.Day == 1 {
				fmt.Fprintf(w, `<div class="day-num"><span class="month-lbl">%s </span>%d</div>`, d.Month, d.Day)
			} else {
				fmt.Fprintf(w, `<div class="day-num">%d</div>`, d.Day)
			}

			// Show trip name if present (takes priority for display)
			if d.TripName != "" {
				fmt.Fprintf(w, `<div class="trip" style="color:%s">%s</div>`, d.Color, d.TripName)
			}

			if d.Location != "" {
				fmt.Fprintf(w, `<div class="loc">%s</div>`, d.Location)
			} else if d.Color != "" {
				fmt.Fprintf(w, `<div class="bar" style="background:%s"></div>`, d.Color)
			}

			fmt.Fprint(w, "</td>")
		}
		fmt.Fprint(w, "</tr>")
	}

	fmt.Fprint(w, `</tbody></table>
</body>
</html>`)
}
