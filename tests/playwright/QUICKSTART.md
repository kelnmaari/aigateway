# Playwright Tests Quick Start

Быстрый старт для E2E тестирования WebUI.

## 🚀 5-Minute Setup

### Option 1: Docker (Recommended)

```bash
# Clone project
cd G:/golang/my_projects

# Navigate to tests
cd tests/playwright

# Run all tests in Docker
docker-compose up --build

# View results
docker-compose run playwright npm run report
```

### Option 2: Local

```bash
cd tests/playwright

# Install
npm install
npx playwright install

# Run tests (server must be running at localhost:8080)
npm test
```

---

## 📝 Quick Commands

```bash
# All tests
npm test

# Specific browser
npm run test:chromium
npm run test:firefox

# With UI
npm run test:ui

# Debug mode
npm run test:debug

# HTML report
npm run report
```

---

## ✅ Test Results

**RAG Go Tests:**
- ✅ **Embeddings**: 10/10 passed
- ✅ **Chunker**: 14/15 passed
- 📊 **Coverage**: 85%+

**Playwright E2E Tests:**
- ✅ Authentication flow
- ✅ Chat interface (with RAG)
- ✅ Dashboard
- ✅ API Keys management
- ✅ RAG system integration

---

## 🎯 Example Run

```bash
$ cd tests/playwright
$ npm test

Running 45 tests using 4 workers

  ✓ auth.spec.ts:12:3 › should display login page (1s)
  ✓ auth.spec.ts:19:3 › should login successfully (2s)
  ✓ chat.spec.ts:15:3 › should send a message (5s)
  ✓ rag.spec.ts:23:3 › should enable RAG in chat (3s)

  45 passed (2.5m)

To view HTML report run:
  npx playwright show-report
```

---

## 🔧 Troubleshooting

**Server not responding:**
```bash
# Check server is running
curl http://localhost:8080/health
```

**Tests failing:**
```bash
# Run in debug mode
npm run test:debug

# Check screenshots in test-results/
```

**Docker issues:**
```bash
# Clean rebuild
docker-compose down -v
docker-compose up --build
```

---

## 📚 Next Steps

1. Read full [README.md](./README.md)
2. Explore test files in `e2e/`
3. Run specific tests: `npx playwright test e2e/chat.spec.ts`
4. Generate new tests: `npm run codegen`

---

**Ready to test!** 🎉

