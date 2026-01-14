import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright configuration for browser-based E2E tests.
 * Tests are located in tests/browser/ and map to PRD use cases (UC-TRP-XXX).
 *
 * @see https://playwright.dev/docs/test-configuration
 */

// Use system Chromium in Docker (set via PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH)
const chromiumExecutable = process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH;

// Video recording: 'on' to always record, 'retain-on-failure' to keep only failed tests
const videoMode = process.env.PLAYWRIGHT_VIDEO === 'on' ? 'on' : 'retain-on-failure';

export default defineConfig({
	testDir: './tests/browser',
	fullyParallel: true,
	forbidOnly: !!process.env.CI,
	retries: process.env.CI ? 2 : 0,
	workers: process.env.CI ? 1 : undefined,
	reporter: [['html', { outputFolder: './tests/browser/playwright-report' }]],
	use: {
		baseURL: process.env.PLAYWRIGHT_BASE_URL || 'http://localhost:3000',
		trace: 'on-first-retry',
		screenshot: 'only-on-failure',
		video: videoMode
	},
	projects: [
		{
			name: 'chromium',
			use: {
				...devices['Desktop Chrome'],
				...(chromiumExecutable && {
					launchOptions: {
						executablePath: chromiumExecutable,
						args: ['--no-sandbox', '--disable-setuid-sandbox']
					}
				})
			}
		}
	],
	// Only use webServer when not running in Docker (Docker uses pre-started backend)
	...(process.env.PLAYWRIGHT_SKIP_WEBSERVER
		? {}
		: {
				webServer: {
					command: 'cd ../.. && ./tc start --fg backend',
					url: 'http://localhost:3000/health',
					reuseExistingServer: !process.env.CI,
					timeout: 120 * 1000
				}
			})
});
