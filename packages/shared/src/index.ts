/**
 * Travel Calendar - Shared Types
 *
 * This package contains TypeScript types only.
 * No runtime code, no dependencies.
 * See ARCHITECTURE.md for conventions.
 */

// ===========================================
// Trip Types
// ===========================================

export type TripPurpose = 'conference' | 'work' | 'vacation' | 'family' | 'personal';
export type TripStatus = 'planned' | 'confirmed' | 'completed' | 'cancelled';

export interface Trip {
  id: string;
  name: string;
  purpose: TripPurpose;
  status: TripStatus;
  startDate: string;
  endDate: string;
  notes?: string;
  createdAt: string;
  updatedAt: string;
}

export interface TripWithItems extends Trip {
  items: Item[];
}

// ===========================================
// Item Types
// ===========================================

export type ItemType = 'flight' | 'hotel' | 'train' | 'drive' | 'event';

interface BaseItem {
  id: string;
  tripId: string;
  type: ItemType;
  date: string;
  notes?: string;
  createdAt: string;
}

export interface FlightItem extends BaseItem {
  type: 'flight';
  from: string;
  to: string;
  departureTime?: string;
  arrivalTime?: string;
  carrier?: string;
  flightNumber?: string;
  confirmation?: string;
}

export interface HotelItem extends BaseItem {
  type: 'hotel';
  name: string;
  location: string;
  checkIn: string;
  checkOut: string;
  confirmation?: string;
}

export interface TrainItem extends BaseItem {
  type: 'train';
  from: string;
  to: string;
  departureTime?: string;
  arrivalTime?: string;
  carrier?: string;
  trainNumber?: string;
  confirmation?: string;
}

export interface DriveItem extends BaseItem {
  type: 'drive';
  from: string;
  to: string;
  rentalCompany?: string;
  confirmation?: string;
}

export interface EventItem extends BaseItem {
  type: 'event';
  name: string;
  location?: string;
  startTime?: string;
  endTime?: string;
}

export type Item = FlightItem | HotelItem | TrainItem | DriveItem | EventItem;

// ===========================================
// Document Types
// ===========================================

export type DocumentType = 'confirmation' | 'receipt' | 'ticket' | 'hotel' | 'visa' | 'insurance' | 'other';

export interface Document {
  id: string;
  tripId?: string;
  itemId?: string;
  type: DocumentType;
  name: string;
  filePath: string;
  mimeType?: string;
  fileSize?: number;
  vendor?: string;
  amount?: number;
  currency?: string;
  createdAt: string;
}

// ===========================================
// API Types
// ===========================================

export interface CreateTripInput {
  name: string;
  purpose: TripPurpose;
  startDate: string;
  endDate: string;
  notes?: string;
}

export interface UpdateTripInput {
  name?: string;
  purpose?: TripPurpose;
  status?: TripStatus;
  startDate?: string;
  endDate?: string;
  notes?: string;
}

export interface TripFilters {
  upcoming?: boolean;
  past?: boolean;
  status?: TripStatus;
  purpose?: TripPurpose;
  location?: string;
  dateRange?: [string, string];
  search?: string;
}
