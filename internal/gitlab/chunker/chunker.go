// Package chunker provides code chunking for GitLab MR analysis
package chunker

import (
	"fmt"
	"regexp"
	"strings"
)

// ChunkStrategy defines how to chunk code
type ChunkStrategy string

const (
	// ChunkByFile chunks entire file as one chunk
	ChunkByFile ChunkStrategy = "file"
	
	// ChunkByHunk chunks by diff hunks
	ChunkByHunk ChunkStrategy = "hunk"
	
	// ChunkByFunction chunks by function/method boundaries
	ChunkByFunction ChunkStrategy = "function"
	
	// ChunkByLines chunks by fixed line count with overlap
	ChunkByLines ChunkStrategy = "lines"
)

// Chunk represents a code chunk
type Chunk struct {
	ID          string            `json:"id"`
	FilePath    string            `json:"file_path"`
	Language    string            `json:"language"`
	ChangeType  string            `json:"change_type"` // added, modified, deleted
	LineStart   int               `json:"line_start"`
	LineEnd     int               `json:"line_end"`
	Content     string            `json:"content"`
	ChunkIndex  int               `json:"chunk_index"`
	TotalChunks int               `json:"total_chunks"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ChunkerConfig configuration for chunker
type ChunkerConfig struct {
	Strategy      ChunkStrategy
	MaxChunkSize  int // Max characters per chunk
	ChunkOverlap  int // Characters of overlap between chunks
	MaxChunks     int // Max chunks per file (0 = unlimited)
}

// DefaultConfig returns default chunker configuration
func DefaultConfig() ChunkerConfig {
	return ChunkerConfig{
		Strategy:     ChunkByHunk,
		MaxChunkSize: 4000, // ~1000 tokens
		ChunkOverlap: 200,
		MaxChunks:    20,
	}
}

// Chunker interface for code chunking
type Chunker interface {
	Chunk(filePath string, content string, changeType string) ([]Chunk, error)
}

// CodeChunker chunks code based on strategy
type CodeChunker struct {
	config ChunkerConfig
}

// NewCodeChunker creates a new code chunker
func NewCodeChunker(config ChunkerConfig) *CodeChunker {
	return &CodeChunker{config: config}
}

// Chunk chunks content based on configured strategy
func (c *CodeChunker) Chunk(filePath string, content string, changeType string) ([]Chunk, error) {
	lang := DetectLanguage(filePath)
	
	switch c.config.Strategy {
	case ChunkByFile:
		return c.chunkByFile(filePath, content, lang, changeType)
	case ChunkByHunk:
		return c.chunkByHunk(filePath, content, lang, changeType)
	case ChunkByFunction:
		return c.chunkByFunction(filePath, content, lang, changeType)
	case ChunkByLines:
		return c.chunkByLines(filePath, content, lang, changeType)
	default:
		return c.chunkByHunk(filePath, content, lang, changeType)
	}
}

// ChunkDiff chunks a diff string
func (c *CodeChunker) ChunkDiff(filePath string, diff string, changeType string) ([]Chunk, error) {
	lang := DetectLanguage(filePath)
	return c.chunkByHunk(filePath, diff, lang, changeType)
}

// chunkByFile treats entire file as one chunk
func (c *CodeChunker) chunkByFile(filePath, content, lang, changeType string) ([]Chunk, error) {
	// If content is too large, fall back to line chunking
	if len(content) > c.config.MaxChunkSize*2 {
		return c.chunkByLines(filePath, content, lang, changeType)
	}
	
	return []Chunk{{
		ID:          fmt.Sprintf("%s:0", filePath),
		FilePath:    filePath,
		Language:    lang,
		ChangeType:  changeType,
		LineStart:   1,
		LineEnd:     countLines(content),
		Content:     content,
		ChunkIndex:  0,
		TotalChunks: 1,
	}}, nil
}

// chunkByHunk chunks by diff hunks
func (c *CodeChunker) chunkByHunk(filePath, diff, lang, changeType string) ([]Chunk, error) {
	hunks := parseDiffHunks(diff)
	
	if len(hunks) == 0 {
		// Not a diff format, treat as regular content
		return c.chunkByLines(filePath, diff, lang, changeType)
	}
	
	chunks := make([]Chunk, 0, len(hunks))
	
	for i, hunk := range hunks {
		// Skip if exceeds max chunks
		if c.config.MaxChunks > 0 && i >= c.config.MaxChunks {
			break
		}
		
		// If hunk is too large, split it
		if len(hunk.Content) > c.config.MaxChunkSize {
			subChunks := c.splitLargeContent(filePath, hunk.Content, lang, changeType, i)
			chunks = append(chunks, subChunks...)
			continue
		}
		
		chunks = append(chunks, Chunk{
			ID:          fmt.Sprintf("%s:%d", filePath, i),
			FilePath:    filePath,
			Language:    lang,
			ChangeType:  changeType,
			LineStart:   hunk.LineStart,
			LineEnd:     hunk.LineEnd,
			Content:     hunk.Content,
			ChunkIndex:  i,
			TotalChunks: len(hunks),
		})
	}
	
	// Update total chunks count
	for i := range chunks {
		chunks[i].TotalChunks = len(chunks)
	}
	
	return chunks, nil
}

// chunkByFunction chunks by function boundaries
func (c *CodeChunker) chunkByFunction(filePath, content, lang, changeType string) ([]Chunk, error) {
	functions := parseFunctions(content, lang)
	
	if len(functions) == 0 {
		// No functions found, fall back to line chunking
		return c.chunkByLines(filePath, content, lang, changeType)
	}
	
	chunks := make([]Chunk, 0, len(functions))
	
	for i, fn := range functions {
		if c.config.MaxChunks > 0 && i >= c.config.MaxChunks {
			break
		}
		
		chunks = append(chunks, Chunk{
			ID:          fmt.Sprintf("%s:%s", filePath, fn.Name),
			FilePath:    filePath,
			Language:    lang,
			ChangeType:  changeType,
			LineStart:   fn.LineStart,
			LineEnd:     fn.LineEnd,
			Content:     fn.Content,
			ChunkIndex:  i,
			TotalChunks: len(functions),
			Metadata: map[string]string{
				"function_name": fn.Name,
			},
		})
	}
	
	return chunks, nil
}

// chunkByLines chunks by fixed line count
func (c *CodeChunker) chunkByLines(filePath, content, lang, changeType string) ([]Chunk, error) {
	lines := strings.Split(content, "\n")
	
	// Calculate lines per chunk based on max chunk size
	avgLineLen := len(content) / max(len(lines), 1)
	linesPerChunk := c.config.MaxChunkSize / max(avgLineLen, 50)
	if linesPerChunk < 10 {
		linesPerChunk = 10
	}
	if linesPerChunk > 100 {
		linesPerChunk = 100
	}
	
	overlapLines := c.config.ChunkOverlap / max(avgLineLen, 50)
	if overlapLines < 2 {
		overlapLines = 2
	}
	
	var chunks []Chunk
	chunkIndex := 0
	
	for i := 0; i < len(lines); {
		if c.config.MaxChunks > 0 && chunkIndex >= c.config.MaxChunks {
			break
		}
		
		end := i + linesPerChunk
		if end > len(lines) {
			end = len(lines)
		}
		
		chunkLines := lines[i:end]
		chunkContent := strings.Join(chunkLines, "\n")
		
		chunks = append(chunks, Chunk{
			ID:          fmt.Sprintf("%s:%d", filePath, chunkIndex),
			FilePath:    filePath,
			Language:    lang,
			ChangeType:  changeType,
			LineStart:   i + 1,
			LineEnd:     end,
			Content:     chunkContent,
			ChunkIndex:  chunkIndex,
		})
		
		chunkIndex++
		
		// Move with overlap
		i = end - overlapLines
		if i <= chunks[len(chunks)-1].LineStart {
			i = end // Prevent infinite loop
		}
	}
	
	// Set total chunks
	for i := range chunks {
		chunks[i].TotalChunks = len(chunks)
	}
	
	return chunks, nil
}

// splitLargeContent splits content that exceeds max chunk size
func (c *CodeChunker) splitLargeContent(filePath, content, lang, changeType string, baseIndex int) []Chunk {
	subChunks, _ := c.chunkByLines(filePath, content, lang, changeType)
	
	// Adjust IDs and indices
	for i := range subChunks {
		subChunks[i].ID = fmt.Sprintf("%s:%d.%d", filePath, baseIndex, i)
		subChunks[i].ChunkIndex = baseIndex*100 + i
	}
	
	return subChunks
}

// DiffHunk represents a hunk from a diff
type DiffHunk struct {
	LineStart int
	LineEnd   int
	Content   string
}

// parseDiffHunks parses diff hunks from diff content
func parseDiffHunks(diff string) []DiffHunk {
	// Match hunk headers: @@ -start,count +start,count @@
	hunkPattern := regexp.MustCompile(`@@\s*-\d+(?:,\d+)?\s*\+(\d+)(?:,(\d+))?\s*@@`)
	
	matches := hunkPattern.FindAllStringSubmatchIndex(diff, -1)
	if len(matches) == 0 {
		return nil
	}
	
	hunks := make([]DiffHunk, 0, len(matches))
	
	for i, match := range matches {
		hunkStart := match[0]
		
		// Find hunk end (next hunk start or end of diff)
		hunkEnd := len(diff)
		if i+1 < len(matches) {
			hunkEnd = matches[i+1][0]
		}
		
		// Extract line number from match
		lineStart := 1
		if match[2] != -1 && match[3] != -1 {
			fmt.Sscanf(diff[match[2]:match[3]], "%d", &lineStart)
		}
		
		lineCount := 1
		if match[4] != -1 && match[5] != -1 {
			fmt.Sscanf(diff[match[4]:match[5]], "%d", &lineCount)
		}
		
		content := diff[hunkStart:hunkEnd]
		
		hunks = append(hunks, DiffHunk{
			LineStart: lineStart,
			LineEnd:   lineStart + lineCount - 1,
			Content:   strings.TrimSpace(content),
		})
	}
	
	return hunks
}

// FunctionInfo represents a parsed function
type FunctionInfo struct {
	Name      string
	LineStart int
	LineEnd   int
	Content   string
}

// parseFunctions extracts functions from code (basic implementation)
func parseFunctions(content, lang string) []FunctionInfo {
	var pattern *regexp.Regexp
	
	switch lang {
	case "go":
		// func name(...) or func (receiver) name(...)
		pattern = regexp.MustCompile(`(?m)^func\s+(?:\([^)]+\)\s+)?(\w+)\s*\(`)
	case "javascript", "typescript":
		// function name(...) or async function name(...) or name = function(...) or name = (...) =>
		pattern = regexp.MustCompile(`(?m)(?:async\s+)?function\s+(\w+)\s*\(|(\w+)\s*=\s*(?:async\s*)?\(?[^)]*\)?\s*=>|(\w+)\s*=\s*function\s*\(`)
	case "python":
		// def name(...)
		pattern = regexp.MustCompile(`(?m)^def\s+(\w+)\s*\(`)
	case "java", "kotlin", "csharp":
		// public/private/protected type name(...)
		pattern = regexp.MustCompile(`(?m)^\s*(?:public|private|protected)?\s*(?:static\s+)?(?:\w+)\s+(\w+)\s*\(`)
	case "rust":
		// fn name(...) or pub fn name(...)
		pattern = regexp.MustCompile(`(?m)^\s*(?:pub\s+)?fn\s+(\w+)`)
	default:
		return nil
	}
	
	matches := pattern.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return nil
	}
	
	lines := strings.Split(content, "\n")
	functions := make([]FunctionInfo, 0, len(matches))
	
	for i, match := range matches {
		// Find function name
		var name string
		for j := 2; j < len(match); j += 2 {
			if match[j] != -1 && match[j+1] != -1 {
				name = content[match[j]:match[j+1]]
				break
			}
		}
		if name == "" {
			continue
		}
		
		funcStart := match[0]
		
		// Find line number
		lineStart := countLinesUntil(content, funcStart) + 1
		
		// Find function end (next function or end)
		funcEnd := len(content)
		if i+1 < len(matches) {
			funcEnd = matches[i+1][0]
		}
		
		// For better boundary detection, find closing brace
		funcEnd = findFunctionEnd(content, funcStart, funcEnd, lang)
		
		lineEnd := countLinesUntil(content, funcEnd)
		
		functions = append(functions, FunctionInfo{
			Name:      name,
			LineStart: lineStart,
			LineEnd:   lineEnd,
			Content:   strings.TrimSpace(content[funcStart:funcEnd]),
		})
	}
	
	return functions
}

// findFunctionEnd finds the end of a function by matching braces
func findFunctionEnd(content string, start, maxEnd int, lang string) int {
	// Find opening brace
	braceStart := strings.Index(content[start:], "{")
	if braceStart == -1 {
		// No brace found (might be Python)
		if lang == "python" {
			// For Python, look for next def or class at same indentation
			return maxEnd
		}
		return maxEnd
	}
	
	braceStart += start
	braceCount := 1
	i := braceStart + 1
	
	for i < len(content) && braceCount > 0 {
		switch content[i] {
		case '{':
			braceCount++
		case '}':
			braceCount--
		case '"', '\'', '`':
			// Skip strings
			quote := content[i]
			i++
			for i < len(content) && content[i] != quote {
				if content[i] == '\\' {
					i++ // Skip escaped char
				}
				i++
			}
		}
		i++
	}
	
	if i > maxEnd {
		return maxEnd
	}
	return i
}

// Helper functions

func countLines(s string) int {
	return strings.Count(s, "\n") + 1
}

func countLinesUntil(s string, pos int) int {
	return strings.Count(s[:pos], "\n")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// DetectLanguage detects language from file path
func DetectLanguage(filepath string) string {
	extensions := map[string]string{
		".go":     "go",
		".py":     "python",
		".js":     "javascript",
		".ts":     "typescript",
		".tsx":    "typescript",
		".jsx":    "javascript",
		".java":   "java",
		".kt":     "kotlin",
		".rs":     "rust",
		".rb":     "ruby",
		".php":    "php",
		".c":      "c",
		".cpp":    "cpp",
		".cc":     "cpp",
		".h":      "c",
		".hpp":    "cpp",
		".cs":     "csharp",
		".swift":  "swift",
		".m":      "objective-c",
		".sql":    "sql",
		".sh":     "bash",
		".yaml":   "yaml",
		".yml":    "yaml",
		".json":   "json",
		".xml":    "xml",
		".html":   "html",
		".css":    "css",
		".scss":   "scss",
		".vue":    "vue",
		".svelte": "svelte",
	}
	
	filepath = strings.ToLower(filepath)
	for ext, lang := range extensions {
		if strings.HasSuffix(filepath, ext) {
			return lang
		}
	}
	
	return "text"
}

