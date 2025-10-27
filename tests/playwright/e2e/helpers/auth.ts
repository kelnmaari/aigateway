import { Page } from '@playwright/test';

/**
 * Authentication Helper Functions
 */

export interface TestUser {
  email: string;
  password: string;
}

export const DEFAULT_TEST_USER: TestUser = {
  email: 'test@example.com',
  password: 'password123',
};

/**
 * Login as test user
 */
export async function loginAsTestUser(page: Page, user: TestUser = DEFAULT_TEST_USER): Promise<void> {
  await page.goto('/web/login.html');
  
  await page.fill('input[name="email"]', user.email);
  await page.fill('input[name="password"]', user.password);
  await page.click('button[type="submit"]');
  
  // Wait for redirect to dashboard or chat
  await page.waitForURL(/\/(dashboard|chat)\.html/, { timeout: 10000 });
}

/**
 * Logout current user
 */
export async function logout(page: Page): Promise<void> {
  await page.click('#logout-btn, button:has-text("Logout")');
  await page.waitForURL(/login\.html/);
}

/**
 * Get stored JWT token
 */
export async function getAuthToken(page: Page): Promise<string | null> {
  return await page.evaluate(() => localStorage.getItem('token'));
}

/**
 * Set JWT token directly (bypass login)
 */
export async function setAuthToken(page: Page, token: string): Promise<void> {
  await page.evaluate((t) => localStorage.setItem('token', t), token);
}

/**
 * Clear authentication data
 */
export async function clearAuth(page: Page): Promise<void> {
  await page.evaluate(() => {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
  });
}

/**
 * Check if user is authenticated
 */
export async function isAuthenticated(page: Page): Promise<boolean> {
  const token = await getAuthToken(page);
  return !!token;
}

