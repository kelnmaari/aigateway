# AIGateway UI Framework - Production Optimization

**Version:** 3.1.0  
**Status:** ✅ Production Ready

## Overview

AIGateway UI Framework включает production-grade оптимизации для минимизации размера бандлов, ускорения загрузки и улучшения пользовательского опыта.

## 🚀 Implemented Optimizations

### 1. **Advanced Minification**

#### JavaScript Minification (esbuild)
- **Primary**: esbuild через `npx` (если доступен)
- **Fallback**: Simple regex-based minification
- **Features**:
  - Tree shaking
  - Dead code elimination
  - ES2020 target
  - IIFE format for isolation

```go
// Auto-detect esbuild availability
minifier := framework.NewMinifier(logger)

// Minify with esbuild or fallback
minified, err := minifier.MinifyJS(content)
```

#### CSS Minification
- Comment removal
- Whitespace reduction
- Semicolon optimization
- Future: cssnano integration

### 2. **Pre-Compression**

Assets are pre-compressed at build time, eliminating runtime CPU overhead.

#### Brotli Compression
- **Level**: Best Compression (11)
- **Ratio**: ~15-25% of original size
- **Support**: Modern browsers (Chrome, Firefox, Edge)
- **Priority**: Served first if accepted

#### Gzip Compression
- **Ratio**: ~25-35% of original size
- **Support**: Universal
- **Priority**: Fallback for older browsers

#### Content Negotiation
```http
Accept-Encoding: br, gzip

# Server responds with:
Content-Encoding: br
```

### 3. **Bundle Size Analysis**

Real-time analysis of all transformations:

```json
{
  "original_size": 45120,
  "minified_size": 12480,
  "gzip_size": 4230,
  "brotli_size": 3450,
  "minify_ratio": 27.7,
  "gzip_ratio": 9.4,
  "brotli_ratio": 7.6,
  "minifier_used": "esbuild"
}
```

**Access stats:**
```bash
# Dev mode only
curl http://localhost:8080/framework/stats
curl http://localhost:8080/framework/stats.json
```

### 4. **Caching Strategy**

#### Development Mode
```http
Cache-Control: no-cache, no-store, must-revalidate
Pragma: no-cache
Expires: 0
```

#### Production Mode
```http
Cache-Control: public, max-age=31536000, immutable
ETag: "abc123def456..."
```

**ETag Validation:**
- Client sends: `If-None-Match: "abc123def456..."`
- Server responds: `304 Not Modified` (if match)

### 5. **Content Hashing**

Assets are versioned with SHA256 hash:

```
framework.js          → framework.a3f21bc4.js
framework.css         → framework.7d2e9f1a.css
```

**Benefits:**
- Automatic cache busting
- Safe aggressive caching
- CDN-friendly

## 📊 Performance Metrics

### Before Optimization
| Asset         | Size    |
|---------------|---------|
| framework.js  | 45 KB   |
| framework.css | 12 KB   |
| **Total**     | **57 KB** |

### After Optimization (Brotli)
| Asset         | Original | Minified | Brotli | Savings |
|---------------|----------|----------|--------|---------|
| framework.js  | 45 KB    | 12 KB    | 3.5 KB | **92%** |
| framework.css | 12 KB    | 4 KB     | 1.2 KB | **90%** |
| **Total**     | **57 KB** | **16 KB** | **4.7 KB** | **92%** |

### Load Time Impact
- **3G Network (750 Kbps)**:
  - Before: ~610 ms
  - After: ~50 ms
  - **Improvement**: 12x faster

- **4G Network (10 Mbps)**:
  - Before: ~45 ms
  - After: ~4 ms
  - **Improvement**: 11x faster

## 🛠️ Usage

### Enable Production Mode

```go
// Initialize builder with minification
builder := framework.NewBuilder(logger, true) // minify=true

// Build assets
if err := builder.Build(); err != nil {
    log.Fatal(err)
}

// Create handler
handler := framework.NewHandler(builder, logger, false) // devMode=false

// Register routes
handler.RegisterRoutes(router)
```

### Environment Detection

```go
env := os.Getenv("ENV")
minify := env == "production" || env == "staging"

builder := framework.NewBuilder(logger, minify)
```

### Inspect Build Stats

**In logs:**
```
INFO Bundle statistics asset=framework.js minified=12.5KB gzip=4.2KB brotli=3.5KB
INFO Bundle statistics asset=framework.css minified=4.0KB gzip=1.5KB brotli=1.2KB
```

**Via API (dev mode):**
```bash
curl http://localhost:8080/framework/stats | jq
```

**Response:**
```json
{
  "version": "3.1.0",
  "devMode": false,
  "js": {
    "size": 12800,
    "hash": "a3f21bc4...",
    "minified": true,
    "build_time": "2025-11-16T18:30:00Z"
  },
  "css": {
    "size": 4096,
    "hash": "7d2e9f1a...",
    "minified": true,
    "build_time": "2025-11-16T18:30:00Z"
  },
  "bundle_analysis": {
    "framework.js": {
      "original_size": 45120,
      "minified_size": 12800,
      "gzip_size": 4300,
      "brotli_size": 3500,
      "minify_ratio": 28.4,
      "gzip_ratio": 9.5,
      "brotli_ratio": 7.8,
      "minifier_used": "esbuild"
    },
    "framework.css": { ... }
  }
}
```

## 🔧 Configuration

### Install esbuild (optional, for best results)

```bash
# Global installation
npm install -g esbuild

# Per-project
npm install esbuild --save-dev
```

**If esbuild is not available:**
- Framework automatically falls back to simple minification
- No errors, just less aggressive optimization
- Still produces valid, working code

### Verify esbuild Detection

```bash
# Check if esbuild is available
npx esbuild --version

# If installed:
# 0.19.5 (or similar)
```

## 📈 Optimization Roadmap

### Completed ✅
- [x] esbuild integration with fallback
- [x] Advanced JS minification
- [x] CSS minification
- [x] Gzip pre-compression
- [x] Brotli pre-compression
- [x] Bundle size analysis
- [x] Content hashing
- [x] ETag caching
- [x] Stats endpoints

### Future Enhancements 🔮
- [ ] Source maps generation (dev mode)
- [ ] CSS splitting (critical/non-critical)
- [ ] Dynamic import support
- [ ] Service worker pre-caching
- [ ] HTTP/2 Server Push hints
- [ ] WebP image optimization
- [ ] Font subsetting

## 🔍 Debugging

### Enable Verbose Logging

```bash
# Set log level to debug
export LOG_LEVEL=debug

# Run server
./bin/server.exe
```

**Output:**
```
DEBUG JS minified with esbuild original_size=45120 minified_size=12800 compression=28.4%
DEBUG Gzip compression complete original=12800 compressed=4300 compression=33.6%
DEBUG Brotli compression complete original=12800 compressed=3500 compression=27.3%
DEBUG Framework asset served asset=framework.js size=3500 encoding=brotli
```

### Test Compression

```bash
# Test Brotli
curl -H "Accept-Encoding: br" http://localhost:8080/framework/framework.js -v

# Test Gzip
curl -H "Accept-Encoding: gzip" http://localhost:8080/framework/framework.js -v

# Test no compression
curl http://localhost:8080/framework/framework.js -v
```

### Verify Cache Headers

```bash
# Production
curl -I http://localhost:8080/framework/framework.js

# Response:
# Cache-Control: public, max-age=31536000, immutable
# ETag: "a3f21bc4..."
# Content-Encoding: br
```

## 🎯 Best Practices

### 1. **Always minify in production**
```go
minify := os.Getenv("ENV") != "development"
```

### 2. **Use versioned URLs**
```html
<!-- Automatic cache busting -->
<script src="/framework/framework.a3f21bc4.js"></script>
```

### 3. **Monitor bundle size**
- Add size budgets to CI/CD
- Alert on >10% size increase
- Review bundle stats regularly

### 4. **Test with real network conditions**
```bash
# Chrome DevTools: Network tab → Throttling
# Fast 3G, Slow 3G, Offline
```

### 5. **Measure actual impact**
```javascript
// Client-side performance
performance.getEntriesByType('resource')
  .filter(r => r.name.includes('framework'))
  .forEach(r => console.log(r.name, r.transferSize, r.duration));
```

## 📚 References

- [esbuild](https://esbuild.github.io/) - JavaScript bundler
- [Brotli](https://github.com/google/brotli) - Compression algorithm
- [HTTP Caching](https://developer.mozilla.org/en-US/docs/Web/HTTP/Caching) - MDN Guide
- [Web Vitals](https://web.dev/vitals/) - Performance metrics

## 🐛 Troubleshooting

### esbuild not found
**Symptom:** Warning in logs: "esbuild failed, falling back to simple minification"

**Solution:**
```bash
npm install -g esbuild
# or
npm install esbuild --save-dev
```

### Large bundle size
**Symptom:** Bundle > 50 KB after minification

**Actions:**
1. Check for duplicate dependencies
2. Remove unused code
3. Use dynamic imports for large features
4. Review imported libraries

### Cache not working
**Symptom:** Assets re-downloaded every time

**Check:**
- ETag header present?
- Cache-Control header correct?
- Content hash changing?
- Browser dev mode disabled?

### Brotli not served
**Symptom:** Always serving gzip/uncompressed

**Verify:**
- Client sends `Accept-Encoding: br`?
- Production mode enabled?
- Assets pre-compressed at build?

---

**Need help?** Check [FRAMEWORK.md](./FRAMEWORK.md) for general framework docs.

