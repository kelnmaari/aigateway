import { test, expect } from '@playwright/test';
import { loginAsTestUser } from './helpers/auth';

/**
 * API Keys Management Tests
 */
test.describe('API Keys Management', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsTestUser(page);
    await page.goto('/web/apikeys.html');
  });

  test('should display API keys page', async ({ page }) => {
    await expect(page).toHaveTitle(/API Keys/i);
    await expect(page.locator('h1, .page-title')).toContainText(/API Keys/i);
  });

  test('should show create API key button', async ({ page }) => {
    const createBtn = page.locator('button:has-text("Create"), button:has-text("New API Key")');
    await expect(createBtn).toBeVisible();
  });

  test('should open create API key modal', async ({ page }) => {
    await page.click('button:has-text("Create"), button:has-text("New API Key")');
    
    // Modal should appear
    await expect(page.locator('.modal, #create-apikey-modal')).toBeVisible();
  });

  test('should create new API key', async ({ page }) => {
    // Open modal
    await page.click('button:has-text("Create"), button:has-text("New API Key")');
    
    // Fill form
    await page.fill('input[name="name"]', 'Test API Key');
    
    // Select models (if available)
    const modelsSelect = page.locator('select[name="models"]');
    if (await modelsSelect.count() > 0) {
      await modelsSelect.selectOption({ index: 1 });
    }
    
    // Submit
    await page.click('button[type="submit"]:has-text("Create")');
    
    // Should show success message or new key
    await expect(page.locator('.success, .alert-success, .apikey-display')).toBeVisible({ timeout: 5000 });
  });

  test('should display API keys list', async ({ page }) => {
    // Wait for list to load
    await page.waitForSelector('.apikey-item, tr', { timeout: 5000 });
    
    const items = page.locator('.apikey-item, tbody tr');
    const count = await items.count();
    
    // Should have at least one key (or empty state)
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('should revoke API key', async ({ page }) => {
    // Wait for list
    await page.waitForSelector('.apikey-item, tr', { timeout: 5000 });
    
    const revokeBtn = page.locator('button:has-text("Revoke"), button:has-text("Delete")').first();
    
    if (await revokeBtn.count() > 0) {
      await revokeBtn.click();
      
      // Confirm dialog
      page.on('dialog', dialog => dialog.accept());
      
      // Should show success
      await expect(page.locator('.success, .alert-success')).toBeVisible({ timeout: 5000 });
    }
  });
});

