import { test, expect } from '@playwright/test';
import { loginAsTestUser } from './helpers/auth';

/**
 * Chat Functionality Tests
 * 
 * Tests chat interface, message sending, RAG integration
 */
test.describe('Chat Interface', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsTestUser(page);
    await page.goto('/web/chat.html');
  });

  test('should display chat interface', async ({ page }) => {
    await expect(page).toHaveTitle(/Chat/);
    await expect(page.locator('#messages-container')).toBeVisible();
    await expect(page.locator('#message-input')).toBeVisible();
    await expect(page.locator('#send-btn')).toBeVisible();
  });

  test('should show welcome screen for new chat', async ({ page }) => {
    await expect(page.locator('#welcome-screen')).toBeVisible();
    await expect(page.locator('.welcome-content h1')).toContainText('Welcome');
  });

  test('should allow model selection', async ({ page }) => {
    const modelSelect = page.locator('#chat-model-select');
    await expect(modelSelect).toBeVisible();
    
    // Wait for models to load
    await page.waitForTimeout(1000);
    
    const options = await modelSelect.locator('option').count();
    expect(options).toBeGreaterThan(0);
  });

  test('should send a message', async ({ page }) => {
    // Type message
    await page.fill('#message-input', 'Hello, this is a test message');
    
    // Send message
    await page.click('#send-btn');
    
    // Hide welcome screen
    await expect(page.locator('#welcome-screen')).not.toBeVisible();
    
    // User message should appear
    await expect(page.locator('.message-user')).toBeVisible();
    await expect(page.locator('.message-user')).toContainText('Hello, this is a test message');
    
    // Wait for assistant response
    await expect(page.locator('.message-assistant')).toBeVisible({ timeout: 30000 });
  });

  test('should clear input after sending', async ({ page }) => {
    await page.fill('#message-input', 'Test message');
    await page.click('#send-btn');
    
    const inputValue = await page.inputValue('#message-input');
    expect(inputValue).toBe('');
  });

  test('should start new conversation', async ({ page }) => {
    // Send a message first
    await page.fill('#message-input', 'First message');
    await page.click('#send-btn');
    
    await expect(page.locator('.message-user')).toBeVisible();
    
    // Click new chat button
    await page.click('#new-chat-btn');
    
    // Welcome screen should reappear
    await expect(page.locator('#welcome-screen')).toBeVisible();
    
    // Messages should be cleared
    const messageCount = await page.locator('.message').count();
    expect(messageCount).toBe(0);
  });

  test('should toggle model parameters panel', async ({ page }) => {
    const paramsToggle = page.locator('#params-toggle');
    const paramsPanel = page.locator('#params-panel');
    
    // Panel should be hidden initially
    await expect(paramsPanel).not.toBeVisible();
    
    // Click toggle
    await paramsToggle.click();
    
    // Panel should be visible
    await expect(paramsPanel).toBeVisible();
    
    // Click again to hide
    await paramsToggle.click();
    await expect(paramsPanel).not.toBeVisible();
  });

  test('should adjust temperature parameter', async ({ page }) => {
    // Open params panel
    await page.click('#params-toggle');
    
    // Adjust temperature slider
    const tempSlider = page.locator('#chat-temperature');
    await tempSlider.fill('1.5');
    
    // Value input should sync
    const tempValue = await page.inputValue('#chat-temperature-value');
    expect(parseFloat(tempValue)).toBeCloseTo(1.5, 1);
  });

  test('should enable RAG system', async ({ page }) => {
    // Open params panel
    await page.click('#params-toggle');
    
    // RAG toggle should exist
    const ragToggle = page.locator('#rag-enabled');
    await expect(ragToggle).toBeVisible();
    
    // Enable RAG
    await ragToggle.check();
    
    // RAG controls should appear
    await expect(page.locator('#rag-controls')).toBeVisible();
    await expect(page.locator('#rag-sources')).toBeVisible();
  });

  test('should select RAG data sources', async ({ page }) => {
    // Open params and enable RAG
    await page.click('#params-toggle');
    await page.check('#rag-enabled');
    
    // Select a source (if available)
    const sourcesSelect = page.locator('#rag-sources');
    const optionCount = await sourcesSelect.locator('option').count();
    
    if (optionCount > 1) {
      // Select first available source
      await sourcesSelect.selectOption({ index: 1 });
      
      const selectedOptions = await sourcesSelect.evaluate((el: HTMLSelectElement) => 
        Array.from(el.selectedOptions).map(opt => opt.value)
      );
      
      expect(selectedOptions.length).toBeGreaterThan(0);
    }
  });

  test('should apply parameter presets', async ({ page }) => {
    // Open params panel
    await page.click('#params-toggle');
    
    // Click creative preset
    await page.click('button[data-preset="creative"]');
    
    // Temperature should be high
    const tempValue = await page.inputValue('#chat-temperature');
    expect(parseFloat(tempValue)).toBeGreaterThan(0.8);
    
    // Click precise preset
    await page.click('button[data-preset="precise"]');
    
    // Temperature should be low
    const tempValuePrecise = await page.inputValue('#chat-temperature');
    expect(parseFloat(tempValuePrecise)).toBeLessThan(0.5);
  });

  test('should handle multiline input with shift+enter', async ({ page }) => {
    const messageInput = page.locator('#message-input');
    
    // Type first line
    await messageInput.fill('Line 1');
    
    // Press Shift+Enter (should add newline, not send)
    await messageInput.press('Shift+Enter');
    
    // Type second line
    await messageInput.type('Line 2');
    
    // Message should contain newline
    const inputValue = await messageInput.inputValue();
    expect(inputValue).toContain('\n');
  });

  test('should disable send button while streaming', async ({ page }) => {
    const sendBtn = page.locator('#send-btn');
    
    await page.fill('#message-input', 'Test streaming');
    await page.click('#send-btn');
    
    // Button should be disabled during streaming
    await expect(sendBtn).toBeDisabled();
    
    // Wait for streaming to complete
    await expect(page.locator('.message-assistant')).toBeVisible({ timeout: 30000 });
    
    // Button should be enabled again
    await expect(sendBtn).toBeEnabled();
  });
});

