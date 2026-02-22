package store

import (
	"context"
	"fmt"
	"log"
	"os"
)

// New creates a StoreInterface based on the STORE_TYPE environment variable.
// Supported values: "sqlite" (default), "firestore".
func New() (StoreInterface, error) {
	storeType := os.Getenv("STORE_TYPE")
	if storeType == "" {
		storeType = "sqlite"
	}

	switch storeType {
	case "sqlite":
		dbPath := os.Getenv("SQLITE_DB_PATH")
		if dbPath == "" {
			dbPath = "data/travel.db"
		}
		log.Printf("Using SQLite store (path: %s)", dbPath)
		return NewSQLite(dbPath)

	case "firestore":
		projectID := os.Getenv("FIREBASE_PROJECT_ID")
		if projectID == "" {
			projectID = "travel-calendar-dev"
		}
		log.Printf("Using Firestore store (project: %s)", projectID)
		return NewFirestore(context.Background(), projectID)

	default:
		return nil, fmt.Errorf("unknown STORE_TYPE: %q (supported: sqlite, firestore)", storeType)
	}
}
