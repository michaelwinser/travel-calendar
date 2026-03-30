//go:build desktop

// Desktop wrapper for travel-calendar using Wails.
//
// Build:
//
//	cd cmd/desktop && wails build
//
// Or via ./dev:
//
//	./dev build desktop
package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"github.com/michaelwinser/appbase"
	appcli "github.com/michaelwinser/appbase/cli"
	travelapp "github.com/michaelwinser/travel-calendar/internal/app"
	"github.com/michaelwinser/travel-calendar/api"
)

//go:embed all:dist
var assets embed.FS

// App struct for Wails binding (minimal)
type App struct{}

func main() {
	// Set up local mode: ~/.config/travel-calendar/app.db, no auth
	appcli.SetupLocalMode("travel-calendar")

	app, err := appbase.New(appbase.Config{Name: "travel-calendar", Quiet: true})
	if err != nil {
		log.Fatal(err)
	}
	defer app.Close()

	activities, err := travelapp.NewActivityStore(app.DB())
	if err != nil {
		log.Fatal(err)
	}
	trips, err := travelapp.NewTripStore(app.DB())
	if err != nil {
		log.Fatal(err)
	}
	parseHistory, err := travelapp.NewParseHistoryStore(app.DB())
	if err != nil {
		log.Fatal(err)
	}
	shareLinks, err := travelapp.NewShareLinkStore(app.DB())
	if err != nil {
		log.Fatal(err)
	}
	shares, err := travelapp.NewShareStore(app.DB())
	if err != nil {
		log.Fatal(err)
	}
	publicProfiles, err := travelapp.NewPublicProfileStore(app.DB())
	if err != nil {
		log.Fatal(err)
	}
	places, err := travelapp.NewPlaceStore(app.DB())
	if err != nil {
		log.Fatal(err)
	}

	activityServer := travelapp.NewActivityServer(activities, trips, parseHistory, shareLinks, shares, publicProfiles, places)
	api.HandlerFromMux(activityServer, app.Server().Router())

	wailsApp := &App{}
	err = wails.Run(&options.App{
		Title:  "Travel Calendar",
		Width:  1100,
		Height: 750,
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: app.LocalHandler(),
		},
		Bind: []interface{}{wailsApp},
	})
	if err != nil {
		log.Fatal(err)
	}
}
