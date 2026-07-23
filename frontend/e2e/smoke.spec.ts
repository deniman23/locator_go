import { test, expect } from '@playwright/test'

test.describe('smoke', () => {
  test('frontend responds', async ({ page }) => {
    const res = await page.goto('/')
    expect(res, 'navigation should return a response').not.toBeNull()
    expect(res!.status()).toBeLessThan(500)
    await expect(page.locator('body')).toBeVisible()
  })
})
