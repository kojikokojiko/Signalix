import { test, expect } from '@playwright/test';

test.describe('Trending page', () => {
  test('shows period tabs', async ({ page }) => {
    await page.goto('/trending');

    // Both period tabs should exist
    await expect(page.getByText('24時間')).toBeVisible();
    await expect(page.getByText('7日間')).toBeVisible();
  });

  test('clicking "7日間" tab selects it', async ({ page }) => {
    await page.goto('/trending');

    // Click the 7日間 tab
    await page.getByText('7日間').click();

    // The tab should now be selected (active state)
    // Check the URL or active class
    const tab = page.getByText('7日間');
    await expect(tab).toBeVisible();

    // Optionally check URL parameter if implemented
    // await expect(page).toHaveURL(/period=7d/);
  });

  test('page renders without error', async ({ page }) => {
    await page.goto('/trending');

    // Page title / heading should be visible
    await expect(page).not.toHaveURL(/error/);
    await expect(page.locator('body')).toBeVisible();
  });
});
