import { test, expect } from '@playwright/test';

test.describe('Landing page', () => {
  test('shows hero text and CTA buttons', async ({ page }) => {
    await page.goto('/');

    // Hero text contains "AIが選ぶ"
    await expect(page.getByText('AIが選ぶ')).toBeVisible();

    // "無料で始める" button exists
    await expect(page.getByRole('link', { name: '無料で始める' })).toBeVisible();
  });

  test('clicking "Trendingを見る" navigates to /trending', async ({ page }) => {
    await page.goto('/');

    // Click the Trending link
    await page.getByRole('link', { name: /Trending/i }).first().click();

    await expect(page).toHaveURL(/\/trending/);
  });

  test('navbar has Signalix logo and key navigation links', async ({ page }) => {
    await page.goto('/');

    // Signalix logo
    await expect(page.getByText('Signalix').first()).toBeVisible();

    // Trending link in navbar
    await expect(page.getByRole('link', { name: 'Trending' }).first()).toBeVisible();

    // Login link
    await expect(page.getByRole('link', { name: 'ログイン' })).toBeVisible();

    // Signup link
    await expect(
      page.getByRole('link', { name: /登録|新規登録|アカウント作成/ }).first()
    ).toBeVisible();
  });
});
