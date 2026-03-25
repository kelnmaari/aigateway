// Package framework provides web UI framework build system
// v3.1.0: Optimized JS+CSS bundling with Go
package framework

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"maps"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

//go:embed assets/*.js assets/*.css assets/components/*.js
var assetsFS embed.FS

// Asset represents a bundled asset (JS or CSS)
type Asset struct {
	Type        AssetType // "js" or "css"
	Content     []byte
	ContentGzip []byte // Pre-compressed gzip
	ContentBr   []byte // Pre-compressed brotli
	ContentHash string
	Size        int64
	Minified    bool
	BuildTime   time.Time
}

// AssetType defines asset type
type AssetType string

const (
	AssetTypeJS  AssetType = "js"
	AssetTypeCSS AssetType = "css"
)

// Builder handles framework asset building
type Builder struct {
	logger   *logrus.Logger
	minifier *Minifier
	cache    map[string]*Asset
	mu       sync.RWMutex
	minify   bool
	devMode  bool
	buildNum int64
	stats    map[string]*BundleStats
}

// NewBuilder creates a new framework builder
func NewBuilder(logger *logrus.Logger, minify bool) *Builder {
	return &Builder{
		logger:   logger,
		minifier: NewMinifier(logger),
		cache:    make(map[string]*Asset),
		stats:    make(map[string]*BundleStats),
		minify:   minify,
		devMode:  !minify,
	}
}

// Build builds all framework assets
func (b *Builder) Build() error {
	b.logger.Info("Building AIGateway UI Framework...")
	startTime := time.Now()

	// Build JavaScript bundle
	if err := b.buildJavaScriptBundle(); err != nil {
		return fmt.Errorf("failed to build JS bundle: %w", err)
	}

	// Build CSS bundle
	if err := b.buildCSSBundle(); err != nil {
		return fmt.Errorf("failed to build CSS bundle: %w", err)
	}

	elapsed := time.Since(startTime)
	b.logger.WithFields(logrus.Fields{
		"elapsed_ms": elapsed.Milliseconds(),
		"js_size":    b.getAssetSize("framework.js"),
		"css_size":   b.getAssetSize("framework.css"),
		"minified":   b.minify,
	}).Info("✅ UI Framework built successfully")

	return nil
}

// buildJavaScriptBundle builds the JavaScript bundle
func (b *Builder) buildJavaScriptBundle() error {
	var buf bytes.Buffer

	// Framework header
	buf.WriteString("// AIGateway UI Framework v3.1.0\n")
	buf.WriteString(fmt.Sprintf("// Built: %s\n", time.Now().Format(time.RFC3339)))
	buf.WriteString("// Minified: " + fmt.Sprintf("%v", b.minify) + "\n\n")

	// Core framework modules
	modules := []string{
		"assets/framework.js",
		"assets/components/state.js",
		"assets/components/router.js",
		"assets/components/http.js",
	}

	for _, module := range modules {
		content, err := fs.ReadFile(assetsFS, module)
		if err != nil {
			// Skip if file doesn't exist (we'll create skeleton)
			b.logger.WithField("module", module).Debug("Module not found, will create skeleton")
			continue
		}

		buf.WriteString(fmt.Sprintf("\n// Module: %s\n", module))
		buf.Write(content)
		buf.WriteString("\n")
	}

	// Create asset
	originalContent := buf.Bytes()
	content := originalContent

	if b.minify {
		minified, err := b.minifier.MinifyJS(content)
		if err != nil {
			b.logger.WithError(err).Warn("JS minification failed, using original")
		} else {
			content = minified

			// Analyze bundle
			stats, err := b.minifier.AnalyzeBundle(originalContent, minified)
			if err != nil {
				b.logger.WithError(err).Warn("Bundle analysis failed")
			} else {
				b.stats["framework.js"] = stats
				b.minifier.PrintStats("framework.js", stats)
			}
		}
	}

	// Pre-compress for production
	var contentGzip, contentBr []byte
	if !b.devMode {
		if gzipped, err := b.minifier.CompressGzip(content); err == nil {
			contentGzip = gzipped
		}
		if brotlied, err := b.minifier.CompressBrotli(content); err == nil {
			contentBr = brotlied
		}
	}

	asset := &Asset{
		Type:        AssetTypeJS,
		Content:     content,
		ContentGzip: contentGzip,
		ContentBr:   contentBr,
		ContentHash: b.hashContent(content),
		Size:        int64(len(content)),
		Minified:    b.minify,
		BuildTime:   time.Now(),
	}

	b.mu.Lock()
	b.cache["framework.js"] = asset
	b.mu.Unlock()

	return nil
}

// buildCSSBundle builds the CSS bundle
func (b *Builder) buildCSSBundle() error {
	var buf bytes.Buffer

	// Framework header
	buf.WriteString("/* AIGateway UI Framework v3.1.0 */\n")
	buf.WriteString(fmt.Sprintf("/* Built: %s */\n", time.Now().Format(time.RFC3339)))
	buf.WriteString(fmt.Sprintf("/* Minified: %v */\n\n", b.minify))

	// Core CSS modules
	modules := []string{
		"assets/framework.css",
		"assets/components/skeleton.css",
		"assets/components/buttons.css",
		"assets/components/forms.css",
	}

	for _, module := range modules {
		content, err := fs.ReadFile(assetsFS, module)
		if err != nil {
			b.logger.WithField("module", module).Debug("Module not found, will create skeleton")
			continue
		}

		buf.WriteString(fmt.Sprintf("\n/* Module: %s */\n", module))
		buf.Write(content)
		buf.WriteString("\n")
	}

	// Create asset
	originalContent := buf.Bytes()
	content := originalContent

	if b.minify {
		minified, err := b.minifier.MinifyCSS(content)
		if err != nil {
			b.logger.WithError(err).Warn("CSS minification failed, using original")
		} else {
			content = minified

			// Analyze bundle
			stats, err := b.minifier.AnalyzeBundle(originalContent, minified)
			if err != nil {
				b.logger.WithError(err).Warn("Bundle analysis failed")
			} else {
				b.stats["framework.css"] = stats
				b.minifier.PrintStats("framework.css", stats)
			}
		}
	}

	// Pre-compress for production
	var contentGzip, contentBr []byte
	if !b.devMode {
		if gzipped, err := b.minifier.CompressGzip(content); err == nil {
			contentGzip = gzipped
		}
		if brotlied, err := b.minifier.CompressBrotli(content); err == nil {
			contentBr = brotlied
		}
	}

	asset := &Asset{
		Type:        AssetTypeCSS,
		Content:     content,
		ContentGzip: contentGzip,
		ContentBr:   contentBr,
		ContentHash: b.hashContent(content),
		Size:        int64(len(content)),
		Minified:    b.minify,
		BuildTime:   time.Now(),
	}

	b.mu.Lock()
	b.cache["framework.css"] = asset
	b.mu.Unlock()

	return nil
}

// GetAsset returns a cached asset
func (b *Builder) GetAsset(name string) (*Asset, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	asset, ok := b.cache[name]
	return asset, ok
}

// GetAssetWithHash returns asset with versioned name
func (b *Builder) GetAssetWithHash(name string) (string, *Asset, bool) {
	asset, ok := b.GetAsset(name)
	if !ok {
		return "", nil, false
	}

	// Generate versioned name: framework.{hash}.js
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	versionedName := fmt.Sprintf("%s.%s%s", base, asset.ContentHash[:8], ext)

	return versionedName, asset, true
}

// hashContent generates SHA256 hash for content
func (b *Builder) hashContent(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

// getAssetSize returns asset size in KB
func (b *Builder) getAssetSize(name string) string {
	asset, ok := b.GetAsset(name)
	if !ok {
		return "0 KB"
	}
	return fmt.Sprintf("%.2f KB", float64(asset.Size)/1024)
}

// GetStats returns build statistics
func (b *Builder) GetStats() map[string]*BundleStats {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make(map[string]*BundleStats)
	maps.Copy(result, b.stats)
	return result
}

// ExportStatsJSON exports statistics as JSON
func (b *Builder) ExportStatsJSON() (string, error) {
	return b.minifier.ExportStats(b.GetStats())
}

// Rebuild triggers a rebuild (for dev mode hot reload)
func (b *Builder) Rebuild() error {
	b.buildNum++
	b.logger.WithField("build_num", b.buildNum).Info("Rebuilding framework assets...")
	return b.Build()
}
