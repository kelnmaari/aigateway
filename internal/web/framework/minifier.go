// Package framework provides advanced minification with esbuild
package framework

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/sirupsen/logrus"
)

// Minifier handles JavaScript and CSS minification
type Minifier struct {
	logger       *logrus.Logger
	useEsbuild   bool
	useCSSNano   bool
	generateMaps bool
}

// NewMinifier creates a new minifier instance
func NewMinifier(logger *logrus.Logger) *Minifier {
	return &Minifier{
		logger:       logger,
		useEsbuild:   checkEsbuildAvailable(),
		useCSSNano:   false, // Simple CSS minification for now
		generateMaps: false,
	}
}

// checkEsbuildAvailable checks if esbuild is available via npx
func checkEsbuildAvailable() bool {
	cmd := exec.Command("npx", "esbuild", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), ".")
}

// MinifyJS minifies JavaScript using esbuild or fallback
func (m *Minifier) MinifyJS(content []byte) ([]byte, error) {
	if m.useEsbuild {
		return m.minifyJSWithEsbuild(content)
	}
	return m.minifyJSSimple(content), nil
}

// minifyJSWithEsbuild uses esbuild for production-grade minification
func (m *Minifier) minifyJSWithEsbuild(content []byte) ([]byte, error) {
	m.logger.Debug("Minifying JS with esbuild")

	cmd := exec.Command("npx", "esbuild",
		"--loader=js",
		"--minify",
		"--target=es2020",
		"--format=iife",
		"--tree-shaking=true",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdin = bytes.NewReader(content)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		m.logger.WithError(err).WithField("stderr", stderr.String()).Warn("esbuild failed, falling back to simple minification")
		return m.minifyJSSimple(content), nil
	}

	result := stdout.Bytes()
	m.logger.WithFields(logrus.Fields{
		"original_size": len(content),
		"minified_size": len(result),
		"compression":   fmt.Sprintf("%.1f%%", float64(len(result))/float64(len(content))*100),
	}).Debug("JS minified with esbuild")

	return result, nil
}

// minifyJSSimple provides simple JS minification (fallback)
func (m *Minifier) minifyJSSimple(content []byte) []byte {
	s := string(content)

	// Remove single-line comments
	reComment := regexp.MustCompile(`(?m)^\s*//.*$`)
	s = reComment.ReplaceAllString(s, "")

	// Remove multi-line comments (preserve /*! license comments)
	reMultiComment := regexp.MustCompile(`/\*(?!\!)[\s\S]*?\*/`)
	s = reMultiComment.ReplaceAllString(s, "")

	// Remove leading/trailing whitespace from lines
	lines := strings.Split(s, "\n")
	var cleaned []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}

	// Join with spaces
	result := strings.Join(cleaned, " ")

	// Reduce multiple spaces to single
	reSpaces := regexp.MustCompile(`\s+`)
	result = reSpaces.ReplaceAllString(result, " ")

	return []byte(result)
}

// MinifyCSS minifies CSS
func (m *Minifier) MinifyCSS(content []byte) ([]byte, error) {
	return m.minifyCSSSimple(content), nil
}

// minifyCSSSimple provides simple CSS minification
func (m *Minifier) minifyCSSSimple(content []byte) []byte {
	s := string(content)

	// Remove comments
	reComment := regexp.MustCompile(`/\*[\s\S]*?\*/`)
	s = reComment.ReplaceAllString(s, "")

	// Remove newlines and extra spaces
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	
	// Reduce multiple spaces
	reSpaces := regexp.MustCompile(`\s+`)
	s = reSpaces.ReplaceAllString(s, " ")

	// Remove space around special characters
	reSpaceAround := regexp.MustCompile(`\s*([{}:;,>+~])\s*`)
	s = reSpaceAround.ReplaceAllString(s, "$1")

	// Remove trailing semicolon before }
	s = strings.ReplaceAll(s, ";}", "}")

	return []byte(strings.TrimSpace(s))
}

// CompressGzip compresses content with gzip
func (m *Minifier) CompressGzip(content []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	
	if _, err := gzWriter.Write(content); err != nil {
		return nil, fmt.Errorf("gzip write failed: %w", err)
	}
	
	if err := gzWriter.Close(); err != nil {
		return nil, fmt.Errorf("gzip close failed: %w", err)
	}

	compressed := buf.Bytes()
	ratio := float64(len(compressed)) / float64(len(content)) * 100

	m.logger.WithFields(logrus.Fields{
		"original":    len(content),
		"compressed":  len(compressed),
		"compression": fmt.Sprintf("%.1f%%", ratio),
	}).Debug("Gzip compression complete")

	return compressed, nil
}

// CompressBrotli compresses content with brotli
func (m *Minifier) CompressBrotli(content []byte) ([]byte, error) {
	var buf bytes.Buffer
	brWriter := brotli.NewWriterLevel(&buf, brotli.BestCompression)
	
	if _, err := brWriter.Write(content); err != nil {
		return nil, fmt.Errorf("brotli write failed: %w", err)
	}
	
	if err := brWriter.Close(); err != nil {
		return nil, fmt.Errorf("brotli close failed: %w", err)
	}

	compressed := buf.Bytes()
	ratio := float64(len(compressed)) / float64(len(content)) * 100

	m.logger.WithFields(logrus.Fields{
		"original":    len(content),
		"compressed":  len(compressed),
		"compression": fmt.Sprintf("%.1f%%", ratio),
	}).Debug("Brotli compression complete")

	return compressed, nil
}

// BundleStats represents bundle size analysis
type BundleStats struct {
	OriginalSize     int     `json:"original_size"`
	MinifiedSize     int     `json:"minified_size"`
	GzipSize         int     `json:"gzip_size"`
	BrotliSize       int     `json:"brotli_size"`
	MinifyRatio      float64 `json:"minify_ratio"`
	GzipRatio        float64 `json:"gzip_ratio"`
	BrotliRatio      float64 `json:"brotli_ratio"`
	MinifierUsed     string  `json:"minifier_used"`
}

// AnalyzeBundle analyzes bundle size and compression
func (m *Minifier) AnalyzeBundle(original, minified []byte) (*BundleStats, error) {
	gzipped, err := m.CompressGzip(minified)
	if err != nil {
		return nil, fmt.Errorf("gzip analysis failed: %w", err)
	}

	brotlied, err := m.CompressBrotli(minified)
	if err != nil {
		return nil, fmt.Errorf("brotli analysis failed: %w", err)
	}

	minifier := "simple"
	if m.useEsbuild {
		minifier = "esbuild"
	}

	stats := &BundleStats{
		OriginalSize: len(original),
		MinifiedSize: len(minified),
		GzipSize:     len(gzipped),
		BrotliSize:   len(brotlied),
		MinifyRatio:  float64(len(minified)) / float64(len(original)) * 100,
		GzipRatio:    float64(len(gzipped)) / float64(len(original)) * 100,
		BrotliRatio:  float64(len(brotlied)) / float64(len(original)) * 100,
		MinifierUsed: minifier,
	}

	return stats, nil
}

// PrintStats prints bundle statistics
func (m *Minifier) PrintStats(name string, stats *BundleStats) {
	m.logger.WithFields(logrus.Fields{
		"asset":          name,
		"original":       formatBytes(stats.OriginalSize),
		"minified":       formatBytes(stats.MinifiedSize),
		"gzip":           formatBytes(stats.GzipSize),
		"brotli":         formatBytes(stats.BrotliSize),
		"minify_ratio":   fmt.Sprintf("%.1f%%", stats.MinifyRatio),
		"gzip_ratio":     fmt.Sprintf("%.1f%%", stats.GzipRatio),
		"brotli_ratio":   fmt.Sprintf("%.1f%%", stats.BrotliRatio),
		"minifier":       stats.MinifierUsed,
	}).Info("Bundle statistics")
}

// formatBytes formats bytes to human readable
func formatBytes(b int) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// ExportStats exports statistics as JSON
func (m *Minifier) ExportStats(stats map[string]*BundleStats) (string, error) {
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal stats: %w", err)
	}
	return string(data), nil
}

