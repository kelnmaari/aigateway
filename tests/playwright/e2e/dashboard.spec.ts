import { test, expect } from '@playwright/test';
import { loginAsTestUser } from './helpers/auth';

/**
 * Dashboard Tests
 * 
 * Tests main dashboard functionality
 */
test.describe('Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsTestUser(page);
    await page.goto('/web/dashboard.html');
  });

  test('should display dashboard', async ({ page }) => {
    await expect(page).toHaveTitle(/Dashboard/);
    await expect(page.locator('h1, .page-title')).toContainText(/Dashboard|Welcome/i);
  });

  test('should show user info', async ({ page }) => {
    // User info should be visible
    await expect(page.locator('#user-info, .user-profile')).toBeVisible();
  });

  test('should display stats cards', async ({ page }) => {
    // Stats cards should be visible
    const statsCards = page.locator('.stat-card, .metric-card');
    await expect(statsCards.first()).toBeVisible();
  });

  test('should navigate to chat page', async ({ page }) => {
    // Find and click chat link
    const chatLink = page.locator('a[href*="chat.html"], button:has-text("Chat")');
    
    if (await chatLink.count() > 0) {
      await chatLink.first().click();
      await expect(page).toHaveURL(/chat\.html/);
    }
  });

  test('should navigate to admin page if admin', async ({ page }) => {
    const adminLink = page.locator('a[href*="admin.html"]');
    
    if (await adminLink.count() > 0) {
      await adminLink.click();
      await expect(page).toHaveURL(/admin\.html/);
    }
  });
});

