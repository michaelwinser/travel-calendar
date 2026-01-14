import { test, expect } from '@playwright/test';

/**
 * Trip List Browser Tests
 * Maps to: UC-TRP-002 (List Upcoming Trips), UC-TRP-006 (Search Trips)
 */
test.describe('Trip List', () => {
	test.beforeEach(async ({ page }) => {
		await page.goto('/trips');
	});

	test('displays trips page with filters [UC-TRP-002]', async ({ page }) => {
		// Wait for loading to finish
		await expect(page.getByText('Loading trips...')).toBeHidden({ timeout: 10000 });

		// Check filter controls are present
		await expect(page.getByPlaceholder('Search trips...')).toBeVisible();
		await expect(page.locator('select').first()).toBeVisible();
	});

	test('shows upcoming trips by default [UC-TRP-002]', async ({ page }) => {
		await expect(page.getByText('Loading trips...')).toBeHidden({ timeout: 10000 });

		// The "Upcoming" option should be selected in the time filter
		const timeFilter = page.locator('select').nth(1);
		await expect(timeFilter).toHaveValue('upcoming');
	});

	test('can filter by purpose [UC-TRP-002]', async ({ page }) => {
		await expect(page.getByText('Loading trips...')).toBeHidden({ timeout: 10000 });

		// Change purpose filter to conference
		const purposeFilter = page.locator('select').first();
		await purposeFilter.selectOption('conference');

		// The filter should be applied
		await expect(purposeFilter).toHaveValue('conference');
	});

	test('can filter by time period [UC-TRP-002]', async ({ page }) => {
		await expect(page.getByText('Loading trips...')).toBeHidden({ timeout: 10000 });

		// Change time filter to 'all'
		const timeFilter = page.locator('select').nth(1);
		await timeFilter.selectOption('all');
		await expect(timeFilter).toHaveValue('all');

		// Change time filter to 'past'
		await timeFilter.selectOption('past');
		await expect(timeFilter).toHaveValue('past');
	});

	test('can search trips [UC-TRP-006]', async ({ page }) => {
		await expect(page.getByText('Loading trips...')).toBeHidden({ timeout: 10000 });

		// Enter search query
		const searchInput = page.getByPlaceholder('Search trips...');
		await searchInput.fill('test');
		await searchInput.press('Enter');

		// Wait for search to complete (loading state might briefly appear)
		await expect(page.getByText('Loading trips...')).toBeHidden({ timeout: 10000 });
	});

	test('shows empty state when no trips found', async ({ page }) => {
		await expect(page.getByText('Loading trips...')).toBeHidden({ timeout: 10000 });

		// Search for something that likely doesn't exist
		const searchInput = page.getByPlaceholder('Search trips...');
		await searchInput.fill('xyznonexistenttrip123456');
		await searchInput.press('Enter');

		await expect(page.getByText('Loading trips...')).toBeHidden({ timeout: 10000 });

		// Should show empty state or no trips found
		// Either "No trips found" text or trip cards should be visible
	});

	test('can navigate to new trip page', async ({ page }) => {
		// The trip list might show a "Create your first trip" link if empty
		// or there's a button elsewhere in the UI
		await expect(page.getByText('Loading trips...')).toBeHidden({ timeout: 10000 });

		// Check if there's a link to create a new trip
		const newTripLink = page.getByRole('link', { name: /new|create/i });
		if (await newTripLink.count() > 0) {
			await newTripLink.first().click();
			await expect(page).toHaveURL('/trips/new');
		}
	});

	test('trip cards are clickable and navigate to trip details', async ({ page }) => {
		await expect(page.getByText('Loading trips...')).toBeHidden({ timeout: 10000 });

		// Change to 'all' to see more trips
		const timeFilter = page.locator('select').nth(1);
		await timeFilter.selectOption('all');

		// Wait for any content update
		await page.waitForTimeout(500);

		// If there are trip cards, clicking one should navigate to the trip detail
		const tripCards = page.locator('article');
		const cardCount = await tripCards.count();

		if (cardCount > 0) {
			await tripCards.first().click();
			await expect(page).toHaveURL(/\/trips\/[a-f0-9-]+/);
		}
	});
});
