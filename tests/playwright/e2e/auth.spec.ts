import { test, expect } from '@playwright/test';

/**
 * Authentication Flow Tests
 * 
 * Tests login, logout, and session persistence
 */
test.describe('Authentication', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/web/login.html');
  });

  test('should display login page', async ({ page }) => {
    await expect(page).toHaveTitle(/Login/);
    await expect(page.locator('h1')).toContainText('Ollama Proxy');
    await expect(page.locator('input[name="email"]')).toBeVisible();
    await expect(page.locator('input[name="password"]')).toBeVisible();
    await expect(page.locator('button[type="submit"]')).toBeVisible();
  });

  test('should show error for invalid credentials', async ({ page }) => {
    await page.fill('input[name="email"]', 'invalid@example.com');
    await page.fill('input[name="password"]', 'wrongpassword');
    await page.click('button[type="submit"]');

    // Wait for error message
    await expect(page.locator('.error-message, .alert-error')).toBeVisible();
  });

  test('should login successfully with valid credentials', async ({ page }) => {
    // Use test credentials (assumes test user exists)
    await page.fill('input[name="email"]', 'test@example.com');
    await page.fill('input[name="password"]', 'password123');
    await page.click('button[type="submit"]');

    // Should redirect to dashboard or chat
    await page.waitForURL(/\/(dashboard|chat)\.html/, { timeout: 10000 });
    
    // Verify JWT token is stored
    const token = await page.evaluate(() => localStorage.getItem('token'));
    expect(token).toBeTruthy();
  });

  test('should logout successfully', async ({ page }) => {
    // Login first
    await page.fill('input[name="email"]', 'test@example.com');
    await page.fill('input[name="password"]', 'password123');
    await page.click('button[type="submit"]');
    
    await page.waitForURL(/\/(dashboard|chat)\.html/);
    
    // Click logout button
    await page.click('#logout-btn, button:has-text("Logout")');
    
    // Should redirect to login
    await page.waitForURL(/login\.html/);
    
    // Token should be cleared
    const token = await page.evaluate(() => localStorage.getItem('token'));
    expect(token).toBeFalsy();
  });

  test('should persist session after page reload', async ({ page }) => {
    // Login
    await page.fill('input[name="email"]', 'test@example.com');
    await page.fill('input[name="password"]', 'password123');
    await page.click('button[type="submit"]');
    
    await page.waitForURL(/\/(dashboard|chat)\.html/);
    
    // Reload page
    await page.reload();
    
    // Should still be logged in
    await expect(page).not.toHaveURL(/login\.html/);
    
    const token = await page.evaluate(() => localStorage.getItem('token'));
    expect(token).toBeTruthy();
  });

  test('should redirect to login if not authenticated', async ({ page }) => {
    // Try to access protected page without login
    await page.goto('/web/dashboard.html');
    
    // Should redirect to login
    await page.waitForURL(/login\.html/);
  });
});

