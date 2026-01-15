import { test, expect } from '@playwright/test';

/**
 * Item CRUD Operations Browser Tests
 * Tests the Add Item UI and item management on trip detail page
 */
test.describe('Item CRUD Operations', () => {
	// Tests must run serially - they share tripId and depend on each other
	test.describe.configure({ mode: 'serial' });

	let tripId: string | null = null;
	const testTripName = `Item Test Trip ${Date.now()}`;

	test.beforeAll(async ({ browser }) => {
		// Create a trip to use for all tests
		const page = await browser.newPage();
		await page.goto('/trips/new');

		await page.locator('#name').fill(testTripName);
		await page.locator('#startDate').fill('2026-06-01');
		await page.locator('#endDate').fill('2026-06-05');
		await page.getByText('Vacation').click();

		await page.getByRole('button', { name: /save/i }).click();
		await expect(page).toHaveURL(/\/trips\/[a-f0-9-]+/, { timeout: 10000 });

		const url = page.url();
		const match = url.match(/\/trips\/([a-f0-9-]+)/);
		if (match) tripId = match[1];

		await page.close();
	});

	test.afterAll(async ({ browser }) => {
		// Cleanup: delete the test trip
		if (tripId) {
			const page = await browser.newPage();
			await page.goto(`/trips/${tripId}`);
			page.on('dialog', (dialog) => dialog.accept());
			await page.locator('button[title="Delete trip"]').click();
			await expect(page).toHaveURL('/trips', { timeout: 10000 });
			await page.close();
		}
	});

	test('displays Add Item section on trip detail', async ({ page }) => {
		await page.goto(`/trips/${tripId}`);
		await expect(page.getByText('Loading trip...')).toBeHidden({ timeout: 10000 });

		// Check for Add Item section button
		const addItemButton = page.getByRole('button', { name: /Add Item/i });
		await expect(addItemButton).toBeVisible();
	});

	test('can expand Add Item section and see type selector', async ({ page }) => {
		await page.goto(`/trips/${tripId}`);
		await expect(page.getByText('Loading trip...')).toBeHidden({ timeout: 10000 });

		// Click to expand Add Item section
		await page.getByRole('button', { name: /Add Item/i }).click();

		// Should see type selector with all 5 types (exact match for button names)
		await expect(page.getByRole('button', { name: 'Flight', exact: true })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Hotel', exact: true })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Train', exact: true })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Drive', exact: true })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Event', exact: true })).toBeVisible();
	});

	test('can add a flight item [UC-TRP-003]', async ({ page }) => {
		await page.goto(`/trips/${tripId}`);
		await expect(page.getByText('Loading trip...')).toBeHidden({ timeout: 10000 });

		// Expand Add Item and select Flight
		await page.getByRole('button', { name: /Add Item/i }).click();
		await page.getByRole('button', { name: 'Flight', exact: true }).click();

		// Fill flight form
		await page.locator('#from').fill('JFK');
		await page.locator('#to').fill('LAX');
		await page.locator('#date').fill('2026-06-01');
		await page.locator('#time').fill('10:30');
		await page.locator('#carrier').fill('Delta');
		await page.locator('#flightNumber').fill('DL456');
		await page.locator('#confirmation').fill('ABC123');

		// Submit
		await page.locator('button[type="submit"]').click();

		// Wait for reload and verify item appears
		await page.waitForTimeout(500);
		await expect(page.getByText('JFK → LAX')).toBeVisible();
		await expect(page.getByText('ABC123')).toBeVisible();
	});

	test('can add a hotel item', async ({ page }) => {
		await page.goto(`/trips/${tripId}`);
		await expect(page.getByText('Loading trip...')).toBeHidden({ timeout: 10000 });

		// Expand Add Item and select Hotel
		await page.getByRole('button', { name: /Add Item/i }).click();
		await page.getByRole('button', { name: 'Hotel', exact: true }).click();

		// Fill hotel form
		await page.locator('#name').fill('Hilton Garden Inn');
		await page.locator('#location').fill('Downtown LA');
		await page.locator('#checkIn').fill('2026-06-01');
		await page.locator('#checkOut').fill('2026-06-05');
		await page.locator('#confirmation').fill('HGI789');

		// Submit
		await page.locator('button[type="submit"]').click();

		// Wait for reload and verify item appears
		await page.waitForTimeout(500);
		await expect(page.getByText('Hilton Garden Inn')).toBeVisible();
		await expect(page.getByText('4 nights')).toBeVisible();
	});

	test('can add an event item', async ({ page }) => {
		await page.goto(`/trips/${tripId}`);
		await expect(page.getByText('Loading trip...')).toBeHidden({ timeout: 10000 });

		// Expand Add Item and select Event
		await page.getByRole('button', { name: /Add Item/i }).click();
		await page.getByRole('button', { name: 'Event', exact: true }).click();

		// Fill event form
		await page.locator('#name').fill('Conference Keynote');
		await page.locator('#location').fill('Convention Center');
		await page.locator('#date').fill('2026-06-02');
		await page.locator('#time').fill('09:00');

		// Submit
		await page.locator('button[type="submit"]').click();

		// Wait for reload and verify item appears
		await page.waitForTimeout(500);
		await expect(page.getByText('Conference Keynote')).toBeVisible();
		await expect(page.getByText('Convention Center')).toBeVisible();
	});

	test('can delete an item', async ({ page }) => {
		await page.goto(`/trips/${tripId}`);
		await expect(page.getByText('Loading trip...')).toBeHidden({ timeout: 10000 });

		// Handle confirmation dialog
		page.on('dialog', (dialog) => dialog.accept());

		// Find the flight item and delete it
		const flightCard = page.locator('.item-card-flight').first();
		await expect(flightCard).toBeVisible();

		// Click delete button on the card
		await flightCard.locator('button[title="Delete"]').click();

		// Wait and verify it's gone
		await page.waitForTimeout(500);
		await expect(page.getByText('JFK → LAX')).toBeHidden();
	});

	test('shows empty state when no items', async ({ page }) => {
		// Create a fresh trip with no items
		await page.goto('/trips/new');
		const emptyTripName = `Empty Trip ${Date.now()}`;

		await page.locator('#name').fill(emptyTripName);
		await page.locator('#startDate').fill('2026-07-01');
		await page.locator('#endDate').fill('2026-07-03');
		await page.getByText('Business').click();

		await page.getByRole('button', { name: /save/i }).click();
		await expect(page).toHaveURL(/\/trips\/[a-f0-9-]+/, { timeout: 10000 });

		// Check for empty state
		await expect(page.getByText('No items yet')).toBeVisible();
		await expect(page.getByText('Add flights, hotels, events, and more')).toBeVisible();

		// Cleanup: delete this trip
		page.on('dialog', (dialog) => dialog.accept());
		await page.locator('button[title="Delete trip"]').click();
		await expect(page).toHaveURL('/trips', { timeout: 10000 });
	});

	test('can cancel adding an item', async ({ page }) => {
		await page.goto(`/trips/${tripId}`);
		await expect(page.getByText('Loading trip...')).toBeHidden({ timeout: 10000 });

		// Expand Add Item and select Flight
		await page.getByRole('button', { name: /Add Item/i }).click();
		await page.getByRole('button', { name: 'Flight', exact: true }).click();

		// Start filling form
		await page.locator('#from').fill('SFO');

		// Cancel should close the entire Add Item section
		await page.getByRole('button', { name: 'Cancel' }).click();

		// Form should be hidden and Add Item section should be collapsed
		await expect(page.locator('#from')).toBeHidden();
		// The collapsible Add Item button should still be visible (section collapsed)
		await expect(page.getByRole('button', { name: /Add Item/i })).toBeVisible();
	});
});
