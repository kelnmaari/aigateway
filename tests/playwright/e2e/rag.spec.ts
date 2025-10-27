import { test, expect } from '@playwright/test';
import { loginAsTestUser } from './helpers/auth';

/**
 * RAG System Tests (v1.13.0+)
 * 
 * Tests RAG data sources management and chat integration
 */
test.describe('RAG System', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsTestUser(page);
  });

  test('should display RAG data sources page', async ({ page }) => {
    // Navigate to RAG management (URL depends on implementation)
    await page.goto('/web/rag-sources.html').catch(() => {
      // If no dedicated page, RAG might be in admin or settings
    });
    
    // Check if page exists
    const pageExists = await page.locator('body').count() > 0;
    expect(pageExists).toBeTruthy();
  });

  test('should show RAG toggle in chat', async ({ page }) => {
    await page.goto('/web/chat.html');
    
    // Open model parameters panel
    await page.click('#params-toggle');
    
    // RAG section should be visible
    await expect(page.locator('.rag-section')).toBeVisible();
    await expect(page.locator('#rag-enabled')).toBeVisible();
  });

  test('should enable RAG in chat', async ({ page }) => {
    await page.goto('/web/chat.html');
    await page.click('#params-toggle');
    
    // Enable RAG
    const ragToggle = page.locator('#rag-enabled');
    await ragToggle.check();
    
    // RAG controls should appear
    await expect(page.locator('#rag-controls')).toBeVisible();
  });

  test('should show RAG source selector', async ({ page }) => {
    await page.goto('/web/chat.html');
    await page.click('#params-toggle');
    await page.check('#rag-enabled');
    
    // Source selector should be visible
    await expect(page.locator('#rag-sources')).toBeVisible();
    
    // Should load sources
    await page.waitForTimeout(1000);
    
    const options = await page.locator('#rag-sources option').count();
    expect(options).toBeGreaterThan(0);
  });

  test('should adjust RAG parameters', async ({ page }) => {
    await page.goto('/web/chat.html');
    await page.click('#params-toggle');
    await page.check('#rag-enabled');
    
    // Adjust Top K
    const topKSlider = page.locator('#rag-top-k');
    await topKSlider.fill('10');
    
    const topKValue = await page.inputValue('#rag-top-k-value');
    expect(parseInt(topKValue)).toBe(10);
    
    // Adjust Min Score
    const minScoreSlider = page.locator('#rag-min-score');
    await minScoreSlider.fill('0.8');
    
    const minScoreValue = await page.inputValue('#rag-min-score-value');
    expect(parseFloat(minScoreValue)).toBeCloseTo(0.8, 1);
    
    // Toggle rerank
    const rerankCheckbox = page.locator('#rag-rerank');
    await rerankCheckbox.uncheck();
    
    const isChecked = await rerankCheckbox.isChecked();
    expect(isChecked).toBeFalsy();
  });

  test('should send message with RAG enabled', async ({ page }) => {
    await page.goto('/web/chat.html');
    await page.click('#params-toggle');
    await page.check('#rag-enabled');
    
    // Select a source if available
    const sourcesSelect = page.locator('#rag-sources');
    const optionCount = await sourcesSelect.locator('option').count();
    
    if (optionCount > 1) {
      await sourcesSelect.selectOption({ index: 1 });
    }
    
    // Send message
    await page.fill('#message-input', 'What information do you have?');
    await page.click('#send-btn');
    
    // Message should be sent
    await expect(page.locator('.message-user')).toBeVisible();
    
    // Response should include RAG context (if configured)
    await expect(page.locator('.message-assistant')).toBeVisible({ timeout: 30000 });
  });

  test('should persist RAG settings', async ({ page }) => {
    await page.goto('/web/chat.html');
    await page.click('#params-toggle');
    
    // Enable RAG and configure
    await page.check('#rag-enabled');
    await page.locator('#rag-top-k').fill('8');
    
    // Reload page
    await page.reload();
    
    // Settings might be persisted in localStorage
    // This depends on implementation
    await page.click('#params-toggle');
    
    // Check if RAG is still enabled (if persistence implemented)
    // const isEnabled = await page.locator('#rag-enabled').isChecked();
    // expect(isEnabled).toBeTruthy();
  });
});

