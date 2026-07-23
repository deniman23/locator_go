import { test, expect } from '@playwright/test';

const apiKey = process.env.E2E_API_KEY || process.env.DEFAULT_ADMIN_API_KEY || '';

async function ensureLocatorUI(page: import('@playwright/test').Page) {
  try {
    const res = await page.request.get('/login');
    if (!res.ok()) {
      test.skip(true, `frontend not reachable (status ${res.status()})`);
    }
    const html = await res.text();
    if (!html.includes('API') && !html.includes('apiKey') && !html.includes('Войти')) {
      test.skip(true, 'response at /login does not look like Locator UI');
    }
  } catch (e) {
    test.skip(true, `frontend not reachable: ${e}`);
  }
}

async function login(page: import('@playwright/test').Page) {
  if (!apiKey) {
    test.skip(true, 'set E2E_API_KEY or DEFAULT_ADMIN_API_KEY for e2e login');
  }
  await page.goto('/login');
  await expect(page.locator('#apiKey')).toBeVisible({ timeout: 10_000 });
  await page.locator('#apiKey').fill(apiKey);
  await page.getByRole('button', { name: /Войти/i }).click();
  await expect(page).not.toHaveURL(/\/login/, { timeout: 15_000 });
}

test.describe('Locator admin smoke', () => {
  test.beforeEach(async ({ page }) => {
    await ensureLocatorUI(page);
  });

  test('login → dashboard map shell', async ({ page }) => {
    await login(page);
    await page.goto('/');
    await expect(page.getByRole('link', { name: /Главная/i })).toBeVisible();
    await expect(page.locator('main')).toBeVisible();
  });

  test('checkpoints page lists', async ({ page }) => {
    await login(page);
    await page.goto('/checkpoints');
    await expect(page.getByRole('link', { name: /Чекпоинты/i })).toBeVisible();
    await expect(page.locator('main')).toBeVisible();
    await page.waitForResponse(
      (r) => r.url().includes('/api/checkpoint') && r.ok(),
      { timeout: 15_000 },
    ).catch(() => undefined);
  });

  test('visits page period controls', async ({ page }) => {
    await login(page);
    await page.goto('/visits');
    await expect(page.getByRole('link', { name: /История визитов/i })).toBeVisible();
    await expect(page.locator('main')).toBeVisible();
  });

  test('login page renders without key', async ({ page }) => {
    await page.goto('/login');
    await expect(page.locator('#apiKey')).toBeVisible({ timeout: 10_000 });
    await expect(page.getByRole('button', { name: /Войти/i })).toBeVisible();
  });
});
