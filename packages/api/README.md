# Travel Calendar API Specification

This package contains the OpenAPI specification for the Travel Calendar REST API.

## Files

- `openapi.yaml` - OpenAPI 3.1 specification (single source of truth)

## Usage

### Generate Go Server Types (Backend)

```bash
cd packages/backend
oapi-codegen -generate types,chi-server -package api \
  ../api/openapi.yaml > internal/api/openapi.gen.go
```

### Generate Go Client (CLI)

```bash
cd packages/cli
oapi-codegen -generate types,client -package client \
  ../api/openapi.yaml > internal/client/client.gen.go
```

### Generate TypeScript Types (Frontend)

```bash
cd packages/shared
npx openapi-typescript ../api/openapi.yaml -o ./types/api.ts
```

### Validate Specification

```bash
npx @redocly/cli lint packages/api/openapi.yaml
```

### Preview Documentation

```bash
npx @redocly/cli preview-docs packages/api/openapi.yaml
```

## Endpoints Summary

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/api/trips` | List trips (with filters) |
| POST | `/api/trips` | Create trip |
| GET | `/api/trips/search` | Search trips |
| GET | `/api/trips/{tripId}` | Get trip |
| PATCH | `/api/trips/{tripId}` | Update trip |
| DELETE | `/api/trips/{tripId}` | Delete trip |
| GET | `/api/trips/{tripId}/items` | List trip items |
| POST | `/api/trips/{tripId}/items` | Add item to trip |
| DELETE | `/api/items/{itemId}` | Delete item |
| GET | `/api/documents` | List documents |

## Schemas

- **Trip** - A travel trip with dates, purpose, and status
- **Item** - A trip item (flight, hotel, train, drive, event)
- **Document** - An attached document (confirmation, itinerary, etc.)
- **Error** - Standard error response

## Making Changes

1. Edit `openapi.yaml`
2. Validate: `npx @redocly/cli lint packages/api/openapi.yaml`
3. Regenerate code in affected packages
4. Update tests as needed
