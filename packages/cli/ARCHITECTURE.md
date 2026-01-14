# CLI Architecture

**Read this file completely before making any changes to the CLI.**

## Overview

The CLI is a command-line interface for managing travel trips, built with Go:
- **Cobra** - CLI framework with command hierarchy
- **Viper** - Configuration management
- **oapi-codegen** - OpenAPI-generated HTTP client

## Source of Truth

The OpenAPI specification (`packages/api/openapi.yaml`) is the single source of truth for:
- HTTP client types (generated)
- API request/response shapes
- Available endpoints

When the API changes, regenerate the client:

```bash
cd packages/cli && go generate ./...
```

## Directory Structure

```
packages/cli/
├── ARCHITECTURE.md           # This file - read first!
├── go.mod
├── go.sum
├── cmd/
│   └── travel/
│       └── main.go           # Entry point
└── internal/
    ├── client/
    │   └── client.gen.go     # Generated HTTP client (DO NOT EDIT)
    ├── cmd/
    │   ├── root.go           # Root command, flags, client init
    │   ├── trips.go          # trips subcommand
    │   ├── items.go          # items subcommand
    │   ├── documents.go      # documents subcommand
    │   └── completion.go     # Shell completion command
    └── output/
        └── output.go         # Output formatting (table/JSON)
```

## Command Hierarchy

```
travel
├── trips
│   ├── list [--upcoming] [--past] [--purpose X]
│   ├── get <id>
│   ├── create --name X --purpose X [--start X] [--end X]
│   ├── update <id> [--name X] [--purpose X] [--status X]
│   ├── delete <id>
│   └── search <query>
├── items
│   ├── list <trip-id>
│   ├── add <trip-id> <type> [--from X] [--to X] [--date X] ...
│   └── delete <id>
├── documents
│   └── list [--trip X] [--unassociated]
└── completion
    ├── bash
    ├── zsh
    ├── fish
    └── powershell
```

## Core Principles

### 1. OpenAPI-Generated Client

The HTTP client is generated from the OpenAPI spec:

```go
// internal/client/client.gen.go (generated)
type ClientInterface interface {
    ListTrips(ctx context.Context, params *ListTripsParams) (*http.Response, error)
    CreateTrip(ctx context.Context, body CreateTripRequest) (*http.Response, error)
    GetTrip(ctx context.Context, tripId string) (*http.Response, error)
    // ...
}
```

Never edit `client.gen.go` directly. Regenerate when API changes.

### 2. Cobra Command Pattern

Each command follows this pattern:

```go
// internal/cmd/trips.go
var tripsCmd = &cobra.Command{
    Use:   "trips",
    Short: "Manage trips",
}

var tripsListCmd = &cobra.Command{
    Use:   "list",
    Short: "List trips",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Get flags
        upcoming, _ := cmd.Flags().GetBool("upcoming")

        // Call API
        params := &client.ListTripsParams{Upcoming: &upcoming}
        resp, err := apiClient.ListTrips(context.Background(), params)
        if err != nil {
            return err
        }

        // Parse and display
        trips := parseTripsResponse(resp)
        output.PrintTrips(trips)
        return nil
    },
}

func init() {
    tripsCmd.AddCommand(tripsListCmd)
    tripsListCmd.Flags().Bool("upcoming", false, "Show only upcoming trips")
    tripsListCmd.Flags().Bool("past", false, "Show only past trips")
    tripsListCmd.Flags().String("purpose", "", "Filter by purpose")
}
```

### 3. Global Flags

Global flags available on all commands:

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--api-url` | `TRAVEL_API_URL` | `http://localhost:3000` | Backend API URL |
| `--json` | - | `false` | Output in JSON format |

### 4. Output Formatting

The CLI supports two output modes:

```go
// internal/output/output.go
var JSONOutput bool  // Set by --json flag

func PrintTrips(trips []Trip) {
    if JSONOutput {
        json.NewEncoder(os.Stdout).Encode(trips)
        return
    }
    // Pretty table output
    for _, trip := range trips {
        fmt.Printf("%s  %s  %s\n", trip.ID, trip.Name, trip.Purpose)
    }
}
```

### 5. Error Handling

Errors are returned, not printed. Cobra handles display:

```go
// Good: return error
func (cmd *cobra.Command) RunE(...) error {
    if err := doSomething(); err != nil {
        return fmt.Errorf("failed to do something: %w", err)
    }
    return nil
}

// Bad: print error and exit
func (cmd *cobra.Command) Run(...) {
    if err := doSomething(); err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)  // Don't do this
    }
}
```

## Configuration

The CLI uses environment variables for configuration:

```bash
# Set API URL
export TRAVEL_API_URL=http://localhost:3000

# Use CLI
travel trips list
```

Or pass as flag:

```bash
travel --api-url http://localhost:3000 trips list
```

## Shell Completion

Generate completion scripts:

```bash
# Bash
travel completion bash > /etc/bash_completion.d/travel

# Zsh
travel completion zsh > ~/.zsh/completions/_travel

# Fish
travel completion fish > ~/.config/fish/completions/travel.fish
```

## Building

Build the CLI binary:

```bash
cd packages/cli && go build -o travel ./cmd/travel
```

Or use the convenience script (if running Go on host):

```bash
cd packages/cli && go build -o travel ./cmd/travel
./travel --help
```

## Regenerating Client

When the OpenAPI spec changes:

```bash
cd packages/cli
oapi-codegen -generate types,client -package client \
  ../api/openapi.yaml > internal/client/client.gen.go
```

## Adding a New Command

1. **Create command file** in `internal/cmd/` (or add to existing)
2. **Define command** with Use, Short, Long, RunE
3. **Add flags** in init()
4. **Register** with parent command
5. **Add output formatter** if new entity type

Example:

```go
// internal/cmd/example.go
var exampleCmd = &cobra.Command{
    Use:   "example <arg>",
    Short: "Do something",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        id := args[0]
        flag, _ := cmd.Flags().GetString("flag")

        result, err := apiClient.DoSomething(context.Background(), id, flag)
        if err != nil {
            return err
        }

        output.PrintResult(result)
        return nil
    },
}

func init() {
    rootCmd.AddCommand(exampleCmd)
    exampleCmd.Flags().String("flag", "", "Some flag")
}
```

## Forbidden Patterns

- Editing `internal/client/client.gen.go` directly
- Direct HTTP calls (use generated client)
- Business logic (belongs in backend)
- Calling os.Exit() in commands (return errors instead)
- Hard-coded API URLs (use flags/env vars)
