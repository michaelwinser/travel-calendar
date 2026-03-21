/**
 * Trip Quick Entry Parser
 *
 * Parses free-text trip descriptions into structured trip data.
 * Examples:
 *   "Milan Jan 23-27"
 *   "London next week business"
 *   "FOSDEM Brussels Jan 29 - Feb 2"
 *   "Tokyo Mar 5-9 vacation"
 *   "NYC conference Dec 10-12"
 */

import type { TripPurpose } from '@travel-calendar/shared';

export interface ParsedTrip {
	name: string;
	location: string;
	startDate: string; // YYYY-MM-DD
	endDate: string;   // YYYY-MM-DD
	purpose?: TripPurpose;
}

const PURPOSE_KEYWORDS: Record<string, TripPurpose> = {
	'conference': 'conference',
	'conf': 'conference',
	'business': 'business',
	'work': 'business',
	'vacation': 'vacation',
	'holiday': 'vacation',
	'family': 'family',
	'personal': 'other',
	'other': 'other'
};

const MONTH_NAMES: Record<string, number> = {
	'jan': 0, 'january': 0,
	'feb': 1, 'february': 1,
	'mar': 2, 'march': 2,
	'apr': 3, 'april': 3,
	'may': 4,
	'jun': 5, 'june': 5,
	'jul': 6, 'july': 6,
	'aug': 7, 'august': 7,
	'sep': 8, 'september': 8,
	'oct': 9, 'october': 9,
	'nov': 10, 'november': 10,
	'dec': 11, 'december': 11
};

/**
 * Parse a free-text trip description into structured data.
 */
export function parseTrip(input: string): ParsedTrip | null {
	if (!input.trim()) return null;

	const tokens = input.trim().split(/\s+/);
	let purpose: TripPurpose | undefined;
	let startDate: Date | null = null;
	let endDate: Date | null = null;
	const textParts: string[] = [];

	let i = 0;
	while (i < tokens.length) {
		const token = tokens[i];
		const lower = token.toLowerCase().replace(/[,]/g, '');

		// Check for purpose keyword
		if (PURPOSE_KEYWORDS[lower]) {
			purpose = PURPOSE_KEYWORDS[lower];
			i++;
			continue;
		}

		// Try to parse "next week", "next month"
		if (lower === 'next' && i + 1 < tokens.length) {
			const nextWord = tokens[i + 1].toLowerCase();
			if (nextWord === 'week') {
				const { start, end } = getNextWeek();
				startDate = start;
				endDate = end;
				i += 2;
				continue;
			}
		}

		// Try to parse date range: "Jan 23-27", "Jan 23 - 27", "Jan 23 - Feb 2"
		const dateResult = tryParseDateRange(tokens, i);
		if (dateResult) {
			startDate = dateResult.start;
			endDate = dateResult.end;
			i = dateResult.nextIndex;
			continue;
		}

		// Otherwise it's a text token (name/location)
		textParts.push(token);
		i++;
	}

	if (textParts.length === 0 && !startDate) return null;

	const text = textParts.join(' ');

	return {
		name: text || 'Trip',
		location: text,
		startDate: startDate ? formatDate(startDate) : '',
		endDate: endDate ? formatDate(endDate) : (startDate ? formatDate(startDate) : ''),
		purpose
	};
}

/**
 * Parse a natural language date string into YYYY-MM-DD.
 * Handles: "Jan 23", "January 23", "Mar 5", "next Tuesday", "tomorrow"
 */
export function parseNaturalDate(input: string): string {
	if (!input.trim()) return '';

	const trimmed = input.trim().toLowerCase();

	// Already a valid date format
	if (/^\d{4}-\d{2}-\d{2}$/.test(trimmed)) return trimmed;

	// "today"
	if (trimmed === 'today') return formatDate(new Date());

	// "tomorrow"
	if (trimmed === 'tomorrow') {
		const d = new Date();
		d.setDate(d.getDate() + 1);
		return formatDate(d);
	}

	// Day of week: "monday", "next monday", "next tue"
	const dayMatch = tryParseDayOfWeek(trimmed);
	if (dayMatch) return formatDate(dayMatch);

	// "Mon DD" or "Month DD" (e.g., "Jan 23", "March 5")
	const monthDayMatch = trimmed.match(/^([a-z]+)\s+(\d{1,2})$/);
	if (monthDayMatch) {
		const monthNum = MONTH_NAMES[monthDayMatch[1]];
		if (monthNum !== undefined) {
			const day = parseInt(monthDayMatch[2]);
			const date = resolveMonthDay(monthNum, day);
			return formatDate(date);
		}
	}

	// "DD Mon" (e.g., "23 Jan")
	const dayMonthMatch = trimmed.match(/^(\d{1,2})\s+([a-z]+)$/);
	if (dayMonthMatch) {
		const monthNum = MONTH_NAMES[dayMonthMatch[2]];
		if (monthNum !== undefined) {
			const day = parseInt(dayMonthMatch[1]);
			const date = resolveMonthDay(monthNum, day);
			return formatDate(date);
		}
	}

	return '';
}

// --- Internal helpers ---

interface DateRange {
	start: Date;
	end: Date;
	nextIndex: number;
}

function tryParseDateRange(tokens: string[], startIdx: number): DateRange | null {
	if (startIdx >= tokens.length) return null;

	const t0 = tokens[startIdx].toLowerCase().replace(/[,]/g, '');

	// Check if t0 is a month name
	const month0 = MONTH_NAMES[t0];
	if (month0 === undefined) return null;

	// Need at least "Mon DD"
	if (startIdx + 1 >= tokens.length) return null;

	const t1 = tokens[startIdx + 1].replace(/[,]/g, '');

	// "Jan 23-27" (day range with hyphen)
	const dayRangeMatch = t1.match(/^(\d{1,2})\s*[-–]\s*(\d{1,2})$/);
	if (dayRangeMatch) {
		const startDay = parseInt(dayRangeMatch[1]);
		const endDay = parseInt(dayRangeMatch[2]);
		return {
			start: resolveMonthDay(month0, startDay),
			end: resolveMonthDay(month0, endDay),
			nextIndex: startIdx + 2
		};
	}

	const startDay = parseInt(t1);
	if (isNaN(startDay)) return null;

	// "Jan 23 - 27" or "Jan 23 - Feb 2"
	if (startIdx + 2 < tokens.length) {
		const t2 = tokens[startIdx + 2].replace(/[,]/g, '');

		if (t2 === '-' || t2 === '–') {
			if (startIdx + 3 < tokens.length) {
				const t3 = tokens[startIdx + 3].replace(/[,]/g, '');

				// "Jan 23 - Feb 2"
				const month1 = MONTH_NAMES[t3.toLowerCase()];
				if (month1 !== undefined && startIdx + 4 < tokens.length) {
					const endDay = parseInt(tokens[startIdx + 4].replace(/[,]/g, ''));
					if (!isNaN(endDay)) {
						return {
							start: resolveMonthDay(month0, startDay),
							end: resolveMonthDay(month1, endDay),
							nextIndex: startIdx + 5
						};
					}
				}

				// "Jan 23 - 27"
				const endDay = parseInt(t3);
				if (!isNaN(endDay)) {
					return {
						start: resolveMonthDay(month0, startDay),
						end: resolveMonthDay(month0, endDay),
						nextIndex: startIdx + 4
					};
				}
			}
		}

		// "Jan 23-27" where hyphen is attached: already handled above
		// "Jan 23" with no end date
	}

	// Just "Jan 23" — single date (start = end)
	return {
		start: resolveMonthDay(month0, startDay),
		end: resolveMonthDay(month0, startDay),
		nextIndex: startIdx + 2
	};
}

function resolveMonthDay(month: number, day: number): Date {
	const now = new Date();
	const year = now.getFullYear();
	const candidate = new Date(year, month, day);

	// If the date has passed, assume next year
	const today = new Date(year, now.getMonth(), now.getDate());
	if (candidate < today) {
		return new Date(year + 1, month, day);
	}
	return candidate;
}

function getNextWeek(): { start: Date; end: Date } {
	const now = new Date();
	const dayOfWeek = now.getDay();
	const daysUntilMonday = ((8 - dayOfWeek) % 7) || 7;
	const monday = new Date(now);
	monday.setDate(now.getDate() + daysUntilMonday);
	const friday = new Date(monday);
	friday.setDate(monday.getDate() + 4);
	return { start: monday, end: friday };
}

const DAY_NAMES: Record<string, number> = {
	'sun': 0, 'sunday': 0,
	'mon': 1, 'monday': 1,
	'tue': 2, 'tuesday': 2,
	'wed': 3, 'wednesday': 3,
	'thu': 4, 'thursday': 4,
	'fri': 5, 'friday': 5,
	'sat': 6, 'saturday': 6
};

function tryParseDayOfWeek(input: string): Date | null {
	let target = input;
	if (target.startsWith('next ')) {
		target = target.substring(5);
	}

	const dayNum = DAY_NAMES[target];
	if (dayNum === undefined) return null;

	const now = new Date();
	const currentDay = now.getDay();
	let diff = dayNum - currentDay;
	if (diff <= 0) diff += 7;

	const result = new Date(now);
	result.setDate(now.getDate() + diff);
	return result;
}

function formatDate(d: Date): string {
	const year = d.getFullYear();
	const month = String(d.getMonth() + 1).padStart(2, '0');
	const day = String(d.getDate()).padStart(2, '0');
	return `${year}-${month}-${day}`;
}
