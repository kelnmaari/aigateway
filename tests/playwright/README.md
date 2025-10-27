# Playwright E2E Tests for Ollama-OpenAI Proxy

Comprehensive end-to-end tests для WebUI с поддержкой Docker.

## 📋 Test Coverage

### Core Features
- ✅ **Authentication** - Login, logout, session persistence
- ✅ **Chat Interface** - Message sending, streaming, model selection
- ✅ **Dashboard** - Stats, navigation
- ✅ **API Keys** - Create, list, revoke
- ✅ **RAG System** - Source management, chat integration (v1.13.0+)

### Browser Support
- ✅ Chromium (Desktop)
- ✅ Firefox (Desktop)
- ✅ WebKit (Safari)
- ✅ Mobile Chrome (Pixel 5)
- ✅ Mobile Safari (iPhone 12)

---

## 🚀 Quick Start

### Prerequisites
- Node.js 18+ (для локального запуска)
- Docker & Docker Compose (для запуска в контейнерах)
- Running Ollama-Proxy server

### Local Setup

```bash
cd tests/playwright

# Install dependencies
npm install

# Install browsers
npx playwright install

# Run tests (requires running server at localhost:8080)
npm test

# Run with UI mode
npm run test:ui

# Run in headed mode (see browser)
npm run test:headed

# Debug mode
npm run test:debug
```

### Docker Setup

```bash
cd tests/playwright

# Run tests in Docker
docker-compose up --build

# Run specific browser
docker-compose run playwright npx playwright test --project=chromium

# Run with HTML report
docker-compose run playwright npx playwright test --reporter=html
```

---

## 📁 Project Structure

```
tests/playwright/
├── e2e/                      # Test files
│   ├── auth.spec.ts          # Authentication tests
│   ├── chat.spec.ts          # Chat interface tests
│   ├── dashboard.spec.ts     # Dashboard tests
│   ├── apikeys.spec.ts       # API Keys management
│   ├── rag.spec.ts           # RAG system tests (v1.13.0+)
│   └── helpers/
│       └── auth.ts           # Auth helper functions
├── playwright.config.ts      # Playwright configuration
├── package.json              # Dependencies
├── Dockerfile                # Docker image for tests
├── docker-compose.yml        # Full stack setup
└── README.md                 # This file
```

---

## 🧪 Running Tests

### All Tests

```bash
npm test
```

### Specific Browser

```bash
npm run test:chromium
npm run test:firefox
npm run test:webkit
npm run test:mobile
```

### Specific Test File

```bash
npx playwright test e2e/auth.spec.ts
npx playwright test e2e/chat.spec.ts
npx playwright test e2e/rag.spec.ts
```

### With Filtering

```bash
# Run tests matching pattern
npx playwright test --grep "should login"

# Skip tests matching pattern
npx playwright test --grep-invert "Mobile"
```

### Debug Mode

```bash
# Interactive debug
npm run test:debug

# With specific test
npx playwright test e2e/auth.spec.ts --debug
```

---

## 📊 Test Reports

### HTML Report

```bash
# Generate report
npm test

# Open report
npm run report

# Or
npx playwright show-report
```

### JSON Report

Results saved to `test-results/results.json`

### JUnit Report

Results saved to `test-results/junit.xml` (для CI/CD integration)

---

## 🐳 Docker Usage

### Full Stack Testing

```bash
# Start app + run tests
docker-compose up --build

# View results
docker-compose run playwright npm run report
```

### Custom Commands

```bash
# Run specific tests
docker-compose run playwright npx playwright test e2e/chat.spec.ts

# Run with specific browser
docker-compose run playwright npx playwright test --project=firefox

# Generate codegen
docker-compose run playwright npx playwright codegen http://app:8080
```

### Cleanup

```bash
docker-compose down -v
```

---

## 🎯 Test Configuration

### Environment Variables

Create `.env` file:

```env
# Base URL для application
BASE_URL=http://localhost:8080

# Test user credentials
TEST_USER_EMAIL=test@example.com
TEST_USER_PASSWORD=password123

# CI mode (reduces parallelism)
CI=false
```

### playwright.config.ts

Ключевые настройки:

```typescript
{
  baseURL: 'http://localhost:8080',
  timeout: 60000,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  use: {
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    trace: 'on-first-retry',
  }
}
```

---

## 📝 Writing New Tests

### Test Template

```typescript
import { test, expect } from '@playwright/test';
import { loginAsTestUser } from './helpers/auth';

test.describe('Feature Name', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsTestUser(page);
    await page.goto('/web/feature.html');
  });

  test('should do something', async ({ page }) => {
    // Arrange
    const element = page.locator('#selector');
    
    // Act
    await element.click();
    
    // Assert
    await expect(element).toHaveText('Expected');
  });
});
```

### Best Practices

1. **Use descriptive test names**: `should login successfully with valid credentials`
2. **One assertion per test**: Focus on single behavior
3. **Setup in beforeEach**: Clean state for each test
4. **Use helpers**: Reuse auth, navigation logic
5. **Wait for elements**: Use `waitForSelector`, `waitForURL`
6. **Handle async**: Always `await` async operations
7. **Screenshots on failure**: Automatic with config
8. **Run with race detector**: `--workers=1` для debugging

---

## 🔧 Debugging

### Visual Debug

```bash
# Opens browser and pauses at each step
npm run test:debug
```

### Playwright Inspector

```bash
# Step through tests
npx playwright test --debug
```

### Screenshots

Failed tests automatically save screenshots to `test-results/`

### Videos

Failed tests automatically record videos (if configured)

### Trace Viewer

```bash
# View trace for failed test
npx playwright show-trace test-results/.../trace.zip
```

---

## 🚨 Common Issues

### Port Already in Use

```bash
# Kill process on port 8080
lsof -ti:8080 | xargs kill -9  # Mac/Linux
netstat -ano | findstr :8080   # Windows
```

### Browser Not Found

```bash
# Reinstall browsers
npx playwright install
```

### Timeout Errors

Increase timeout in `playwright.config.ts`:

```typescript
timeout: 120000, // 2 minutes
```

### Authentication Failures

Ensure test user exists:

```bash
# Create test user via API or admin panel
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'
```

---

## 📈 CI/CD Integration

### GitHub Actions

```yaml
name: E2E Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Docker Compose
        run: docker-compose -f tests/playwright/docker-compose.yml up --build --exit-code-from playwright
      
      - name: Upload test results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: playwright-report
          path: tests/playwright/playwright-report
```

### GitLab CI

```yaml
e2e-tests:
  stage: test
  image: docker:latest
  services:
    - docker:dind
  script:
    - cd tests/playwright
    - docker-compose up --build --exit-code-from playwright
  artifacts:
    when: always
    paths:
      - tests/playwright/playwright-report
      - tests/playwright/test-results
```

---

## 📚 Resources

- [Playwright Documentation](https://playwright.dev/)
- [Best Practices](https://playwright.dev/docs/best-practices)
- [API Reference](https://playwright.dev/docs/api/class-playwright)
- [Selectors Guide](https://playwright.dev/docs/selectors)

---

## 🤝 Contributing

При добавлении новых tests:

1. Follow existing patterns
2. Add to appropriate spec file or create new one
3. Update this README
4. Run all tests before commit
5. Ensure tests pass in CI

---

**Version:** 1.0.0  
**Last Updated:** 2025-10-26  
**Playwright Version:** 1.40.0

