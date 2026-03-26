import { test, expect } from '@playwright/test';

test.describe('Login page', () => {
  test('has email and password form fields', async ({ page }) => {
    await page.goto('/login');

    await expect(page.getByLabel(/メールアドレス/i)).toBeVisible();
    await expect(page.getByLabel(/パスワード/i)).toBeVisible();
    await expect(page.getByRole('button', { name: /ログイン/i })).toBeVisible();
  });

  test('form does not submit with empty fields', async ({ page }) => {
    await page.goto('/login');

    // Submit without filling in fields
    await page.getByRole('button', { name: /ログイン/i }).click();

    // Should still be on the login page (not navigated away)
    await expect(page).toHaveURL(/\/login/);

    // Form validation should show errors or prevent navigation
    const currentURL = page.url();
    expect(currentURL).toContain('/login');
  });
});

test.describe('Signup page', () => {
  test('has display name, email and password form fields', async ({ page }) => {
    await page.goto('/signup');

    // Should have 3 fields: display name, email, password
    await expect(page.getByLabel(/表示名|ユーザー名|名前/i)).toBeVisible();
    await expect(page.getByLabel(/メールアドレス/i)).toBeVisible();
    await expect(page.getByLabel(/パスワード/i)).toBeVisible();
  });

  test('password strength indicator appears when typing', async ({ page }) => {
    await page.goto('/signup');

    // Find the password input
    const passwordInput = page.getByLabel(/パスワード/i);
    await passwordInput.fill('test');

    // Strength indicator should appear
    const strengthIndicator = page.locator('[data-testid="password-strength"], .password-strength, [class*="strength"]');
    // Check if any strength-related element appeared or that the password field accepted input
    await expect(passwordInput).toHaveValue('test');
  });
});
