# Go Embed Integration

## Обзор

Svelte приложение будет встраиваться в Go бинарник через `embed.FS`. При сборке:
1. Svelte компилируется в статические файлы
2. Go embed включает их в бинарник
3. Go HTTP сервер раздаёт файлы из embed.FS

## Текущая архитектура

### internal/web/embed.go (текущий)

```go
package web

import (
	"embed"
)

//go:embed all:static
var StaticFiles embed.FS
```

### Структура static (текущая)

```
internal/web/static/
├── app.js
├── htmx.min.js
├── index.html
├── style.css
├── themes.css
└── websocket.js
```

## Новая архитектура

### Build output структура

```
internal/web/static/
├── _app/
│   ├── immutable/
│   │   ├── assets/
│   │   │   └── *.css
│   │   ├── chunks/
│   │   │   └── *.js
│   │   └── entry/
│   │       ├── app.*.js
│   │       └── start.*.js
│   └── version.json
├── favicon.ico
└── index.html
```

### SvelteKit Static Adapter Configuration

```javascript
// svelte.config.js
import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
  preprocess: vitePreprocess(),
  
  kit: {
    adapter: adapter({
      pages: '../internal/web/static',
      assets: '../internal/web/static',
      fallback: 'index.html',
      precompress: false,
      strict: true
    }),
    
    paths: {
      base: '',
      relative: false
    },
    
    prerender: {
      entries: ['*'],
      handleHttpError: 'warn',
      handleMissingId: 'warn'
    },
    
    // No SSR - fully static
    ssr: false
  }
};

export default config;
```

### Vite Configuration

```javascript
// vite.config.js
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [sveltekit()],
  
  build: {
    target: 'es2020',
    sourcemap: false,
    minify: 'esbuild',
    rollupOptions: {
      output: {
        manualChunks: {
          'vendor': ['svelte', '@sveltejs/kit'],
          'ui': [
            'lucide-svelte',
            'svelte-sonner',
          ]
        }
      }
    }
  },
  
  optimizeDeps: {
    include: ['lucide-svelte']
  }
});
```

## Go Server Integration

### Updated embed.go

```go
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:static
var staticFS embed.FS

// StaticFiles returns the embedded static files as an fs.FS
func StaticFiles() fs.FS {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	return sub
}

// GetStaticFS returns the raw embedded FS (for testing)
func GetStaticFS() embed.FS {
	return staticFS
}
```

### HTTP Handler (routes.go)

```go
package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/ollama-openai-proxy/internal/web"
)

func (s *Server) setupRoutes() {
	// API routes
	api := s.router.PathPrefix("/api").Subrouter()
	s.setupAPIRoutes(api)
	
	// OpenAI compatible routes
	v1 := s.router.PathPrefix("/v1").Subrouter()
	s.setupOpenAIRoutes(v1)
	
	// Static files - SPA fallback
	staticFS := web.StaticFiles()
	s.router.PathPrefix("/").Handler(spaHandler(staticFS))
}

// spaHandler serves static files with SPA fallback
func spaHandler(staticFS fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(staticFS))
	
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		
		// Remove leading slash
		if strings.HasPrefix(path, "/") {
			path = path[1:]
		}
		
		// Check if file exists
		if path != "" {
			if _, err := fs.Stat(staticFS, path); err == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		
		// SPA fallback - serve index.html
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
```

### Alternative: Chi Router

```go
package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/ollama-openai-proxy/internal/web"
)

func (s *Server) setupRoutes() {
	r := chi.NewRouter()
	
	// API routes
	r.Route("/api", func(r chi.Router) {
		s.setupAPIRoutes(r)
	})
	
	// OpenAI compatible routes
	r.Route("/v1", func(r chi.Router) {
		s.setupOpenAIRoutes(r)
	})
	
	// Static files with SPA fallback
	staticFS := web.StaticFiles()
	r.Handle("/*", spaHandler(staticFS))
	
	s.router = r
}

func spaHandler(staticFS fs.FS) http.Handler {
	httpFS := http.FS(staticFS)
	fileServer := http.FileServer(httpFS)
	
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		
		// Try to open the file
		f, err := httpFS.Open(path)
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		
		// File not found - serve index.html for SPA routing
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
```

## Build Process

### Makefile

```makefile
.PHONY: build build-frontend build-backend clean

# Frontend
FRONTEND_DIR := web-svelte
STATIC_DIR := internal/web/static

# Build all
build: build-frontend build-backend

# Build frontend
build-frontend:
	@echo "Building Svelte frontend..."
	cd $(FRONTEND_DIR) && npm run build
	@echo "Frontend built to $(STATIC_DIR)"

# Build backend
build-backend:
	@echo "Building Go backend..."
	go build -o bin/server.exe ./cmd/server

# Development - watch frontend
dev-frontend:
	cd $(FRONTEND_DIR) && npm run dev

# Clean
clean:
	rm -rf $(STATIC_DIR)/*
	rm -rf bin/

# Full rebuild
rebuild: clean build
```

### NPM Scripts (package.json)

```json
{
  "name": "aigateway-frontend",
  "version": "0.0.1",
  "private": true,
  "scripts": {
    "dev": "vite dev",
    "build": "vite build",
    "preview": "vite preview",
    "check": "svelte-kit sync && svelte-check --tsconfig ./tsconfig.json",
    "check:watch": "svelte-kit sync && svelte-check --tsconfig ./tsconfig.json --watch",
    "lint": "eslint .",
    "format": "prettier --write ."
  },
  "devDependencies": {
    "@sveltejs/adapter-static": "^3.0.0",
    "@sveltejs/kit": "^2.16.0",
    "@sveltejs/vite-plugin-svelte": "^4.0.0",
    "svelte": "^5.0.0",
    "svelte-check": "^4.0.0",
    "typescript": "^5.0.0",
    "vite": "^5.0.0"
  }
}
```

## Development Workflow

### Вариант 1: Proxy через Vite

```javascript
// vite.config.js (dev mode)
export default defineConfig({
  plugins: [sveltekit()],
  
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true
      },
      '/v1': {
        target: 'http://localhost:8080',
        changeOrigin: true
      }
    }
  }
});
```

**Workflow:**
1. Запустить Go сервер: `go run ./cmd/server`
2. Запустить Vite dev: `cd web-svelte && npm run dev`
3. Открыть http://localhost:5173

### Вариант 2: Go serve dev files

```go
// cmd/server/main.go
func main() {
	cfg := config.Load()
	
	var staticFS fs.FS
	if cfg.Development {
		// Serve files directly from disk in dev mode
		staticFS = os.DirFS("web-svelte/build")
	} else {
		// Serve embedded files in production
		staticFS = web.StaticFiles()
	}
	
	// ...
}
```

## Caching Strategy

### HTTP Cache Headers

```go
func spaHandler(staticFS fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(staticFS))
	
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		
		// Cache immutable assets forever
		if strings.Contains(path, "/_app/immutable/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else if path == "" || path == "index.html" {
			// Never cache index.html
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		} else {
			// Cache other static assets for 1 hour
			w.Header().Set("Cache-Control", "public, max-age=3600")
		}
		
		// ... serve file or fallback
	})
}
```

### ETag Support

```go
func spaHandlerWithETag(staticFS fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		
		if f, err := staticFS.Open(path); err == nil {
			defer f.Close()
			
			if stat, err := f.Stat(); err == nil {
				etag := fmt.Sprintf(`"%x-%x"`, stat.ModTime().Unix(), stat.Size())
				w.Header().Set("ETag", etag)
				
				if r.Header.Get("If-None-Match") == etag {
					w.WriteHeader(http.StatusNotModified)
					return
				}
			}
		}
		
		// ... serve file
	})
}
```

## Compression

### Gzip Middleware

```go
import "github.com/NYTimes/gzip-handler"

func main() {
	// Wrap static handler with gzip
	staticHandler := spaHandler(web.StaticFiles())
	compressedHandler := gziphandler.GzipHandler(staticHandler)
	
	r.Handle("/*", compressedHandler)
}
```

### Pre-compressed Assets (Optional)

```javascript
// vite.config.js
import viteCompression from 'vite-plugin-compression';

export default defineConfig({
  plugins: [
    sveltekit(),
    viteCompression({
      algorithm: 'gzip',
      ext: '.gz'
    }),
    viteCompression({
      algorithm: 'brotliCompress',
      ext: '.br'
    })
  ]
});
```

## Testing

### Unit Test for Static Handler

```go
func TestSPAHandler(t *testing.T) {
	handler := spaHandler(web.StaticFiles())
	
	tests := []struct {
		name     string
		path     string
		wantCode int
	}{
		{"index", "/", 200},
		{"app route", "/dashboard", 200},
		{"nested route", "/admin/users", 200},
		{"static asset", "/_app/immutable/app.js", 200},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rec := httptest.NewRecorder()
			
			handler.ServeHTTP(rec, req)
			
			if rec.Code != tt.wantCode {
				t.Errorf("got %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Build

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Node
        uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'
          cache-dependency-path: web-svelte/package-lock.json
      
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      
      - name: Build Frontend
        run: |
          cd web-svelte
          npm ci
          npm run build
      
      - name: Build Backend
        run: go build -o bin/server ./cmd/server
      
      - name: Test
        run: go test -race ./...
```

