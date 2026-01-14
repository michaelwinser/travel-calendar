/**
 * Travel Calendar - Shared Types
 *
 * This package contains TypeScript types generated from the OpenAPI specification.
 * Types are auto-generated - do not edit manually.
 *
 * To regenerate: pnpm generate
 * See ARCHITECTURE.md for conventions.
 */

// Re-export all types from generated API types
export type { paths, components, operations } from './api';

// Convenience aliases for commonly used types
import type { components } from './api';

// Schema types
export type Trip = components['schemas']['Trip'];
export type TripPurpose = components['schemas']['TripPurpose'];
export type TripStatus = components['schemas']['TripStatus'];
export type CreateTripRequest = components['schemas']['CreateTripRequest'];
export type UpdateTripRequest = components['schemas']['UpdateTripRequest'];

export type Item = components['schemas']['Item'];
export type ItemType = components['schemas']['ItemType'];
export type CreateItemRequest = components['schemas']['CreateItemRequest'];

export type Document = components['schemas']['Document'];

export type HealthResponse = components['schemas']['HealthResponse'];
export type ErrorResponse = components['schemas']['Error'];

// Location types
export type BaseLocations = components['schemas']['BaseLocations'];
export type SetBaseLocationsRequest = components['schemas']['SetBaseLocationsRequest'];
export type TripDayLocation = components['schemas']['TripDayLocation'];
export type SetTripLocationsRequest = components['schemas']['SetTripLocationsRequest'];
export type LocationSourceType = components['schemas']['LocationSourceType'];
export type LocationSource = components['schemas']['LocationSource'];
export type LocationOnDateResponse = components['schemas']['LocationOnDateResponse'];
export type LocationRangeSegment = components['schemas']['LocationRangeSegment'];
