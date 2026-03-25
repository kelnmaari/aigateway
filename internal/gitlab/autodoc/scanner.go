// Package autodoc provides automatic documentation generation for code.
package autodoc

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"aigateway/internal/rag/vector"

	"github.com/sirupsen/logrus"
)

// Scanner scans code for undocumented symbols.
type Scanner struct {
	vectorStore *vector.QdrantStore
	logger      *logrus.Logger
}

// NewScanner creates a new documentation scanner.
func NewScanner(vectorStore *vector.QdrantStore, logger *logrus.Logger) *Scanner {
	return &Scanner{
		vectorStore: vectorStore,
		logger:      logger,
	}
}

// Scan scans a project for undocumented code.
func (s *Scanner) Scan(ctx context.Context, req ScanRequest) (*ScanResult, error) {
	startTime := time.Now()
	scanID := fmt.Sprintf("autodoc-%s", time.Now().Format("20060102-150405"))

	s.logger.WithFields(logrus.Fields{
		"project_id": req.ProjectID,
		"collection": req.CollectionName,
		"scan_id":    scanID,
	}).Info("Starting documentation scan")

	result := &ScanResult{
		ProjectID: req.ProjectID,
		ScanID:    scanID,
		ScannedAt: startTime,
		Status:    "running",
		Symbols:   []UndocumentedSymbol{},
		Summary: ScanSummary{
			ByType:       make(map[string]int),
			ByLanguage:   make(map[string]int),
			ByImportance: make(map[string]int),
		},
	}

	// Set defaults
	maxFiles := req.MaxFiles
	if maxFiles <= 0 {
		maxFiles = 100
	}

	// Collect code files
	files, err := s.collectCodeFiles(ctx, req.CollectionName, req.ProjectID, maxFiles)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("Failed to collect files: %v", err)
		result.Duration = time.Since(startTime).String()
		return result, err
	}

	result.FilesScanned = len(files)

	if len(files) == 0 {
		result.Status = "completed"
		result.Error = "No code files found in index"
		result.Duration = time.Since(startTime).String()
		return result, nil
	}

	s.logger.WithField("files", len(files)).Debug("Collected files for scanning")

	// Scan each file for undocumented symbols
	for _, file := range files {
		symbols := s.scanFile(file, req.ExportedOnly)
		result.Symbols = append(result.Symbols, symbols...)
	}

	// Build summary
	s.buildSummary(result)

	result.Status = "completed"
	result.Duration = time.Since(startTime).String()

	s.logger.WithFields(logrus.Fields{
		"project_id":    req.ProjectID,
		"scan_id":       scanID,
		"undocumented":  len(result.Symbols),
		"files_scanned": result.FilesScanned,
		"duration":      result.Duration,
	}).Info("Documentation scan completed")

	return result, nil
}

// codeFile represents a code file from the index
type codeFile struct {
	Path     string
	Content  string
	Language string
}

// collectCodeFiles retrieves code files from the vector store
func (s *Scanner) collectCodeFiles(ctx context.Context, collection, projectID string, maxFiles int) ([]codeFile, error) {
	var files []codeFile
	seen := make(map[string]bool)

	err := s.vectorStore.ScrollAll(ctx, collection, map[string]any{
		"project_id": projectID,
	}, func(docs []vector.VectorDocument) error {
		for _, doc := range docs {
			if len(files) >= maxFiles {
				return nil
			}

			filePath, ok := doc.Metadata["file_path"].(string)
			if !ok || filePath == "" {
				continue
			}

			// Skip if already seen
			if seen[filePath] {
				continue
			}

			// Only process code files
			if !isCodeFile(filePath) {
				continue
			}

			seen[filePath] = true

			content := doc.Text
			if c, ok := doc.Metadata["content"].(string); ok && c != "" {
				content = c
			}

			files = append(files, codeFile{
				Path:     filePath,
				Content:  content,
				Language: detectLanguage(filePath),
			})
		}
		return nil
	})

	return files, err
}

// scanFile scans a single file for undocumented symbols
func (s *Scanner) scanFile(file codeFile, exportedOnly bool) []UndocumentedSymbol {
	var symbols []UndocumentedSymbol

	switch file.Language {
	case "go":
		symbols = s.scanGoFile(file, exportedOnly)
	case "javascript", "typescript":
		symbols = s.scanJSFile(file, exportedOnly)
	case "python":
		symbols = s.scanPythonFile(file, exportedOnly)
	}

	return symbols
}

// scanGoFile scans a Go file for undocumented functions and types
func (s *Scanner) scanGoFile(file codeFile, exportedOnly bool) []UndocumentedSymbol {
	var symbols []UndocumentedSymbol

	lines := strings.Split(file.Content, "\n")

	// Pattern for func declarations
	funcPattern := regexp.MustCompile(`^func\s+(?:\([^)]+\)\s+)?(\w+)\s*\([^)]*\)`)
	// Pattern for type declarations
	typePattern := regexp.MustCompile(`^type\s+(\w+)\s+`)
	// Pattern for doc comments
	docCommentPattern := regexp.MustCompile(`^//`)

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check for func
		if matches := funcPattern.FindStringSubmatch(trimmedLine); len(matches) > 1 {
			name := matches[1]
			isExported := strings.ToUpper(name[:1]) == name[:1]

			if exportedOnly && !isExported {
				continue
			}

			// Skip main, init, test functions
			if name == "main" || name == "init" || strings.HasPrefix(name, "Test") || strings.HasPrefix(name, "Benchmark") {
				continue
			}

			// Check if previous line is a doc comment
			hasDoc := i > 0 && docCommentPattern.MatchString(strings.TrimSpace(lines[i-1]))

			if !hasDoc {
				importance := "medium"
				if isExported {
					importance = "high"
				}

				symbols = append(symbols, UndocumentedSymbol{
					Name:       name,
					Type:       "function",
					FilePath:   file.Path,
					StartLine:  i + 1,
					Language:   "go",
					Signature:  trimmedLine,
					IsExported: isExported,
					Importance: importance,
				})
			}
		}

		// Check for type
		if matches := typePattern.FindStringSubmatch(trimmedLine); len(matches) > 1 {
			name := matches[1]
			isExported := strings.ToUpper(name[:1]) == name[:1]

			if exportedOnly && !isExported {
				continue
			}

			// Check if previous line is a doc comment
			hasDoc := i > 0 && docCommentPattern.MatchString(strings.TrimSpace(lines[i-1]))

			if !hasDoc {
				importance := "medium"
				if isExported {
					importance = "high"
				}

				symbols = append(symbols, UndocumentedSymbol{
					Name:       name,
					Type:       "type",
					FilePath:   file.Path,
					StartLine:  i + 1,
					Language:   "go",
					Signature:  trimmedLine,
					IsExported: isExported,
					Importance: importance,
				})
			}
		}
	}

	return symbols
}

// scanJSFile scans a JavaScript/TypeScript file for undocumented code
func (s *Scanner) scanJSFile(file codeFile, exportedOnly bool) []UndocumentedSymbol {
	var symbols []UndocumentedSymbol

	lines := strings.Split(file.Content, "\n")

	// Patterns
	funcPattern := regexp.MustCompile(`^(?:export\s+)?(?:async\s+)?function\s+(\w+)\s*\(`)
	classPattern := regexp.MustCompile(`^(?:export\s+)?class\s+(\w+)`)
	arrowFuncPattern := regexp.MustCompile(`^(?:export\s+)?(?:const|let)\s+(\w+)\s*=\s*(?:async\s+)?\([^)]*\)\s*=>`)
	jsdocPattern := regexp.MustCompile(`^\s*\*`)

	inJSDoc := false
	lastJSDocEnd := -1

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Track JSDoc blocks
		if strings.Contains(trimmedLine, "/**") {
			inJSDoc = true
		}
		if inJSDoc && strings.Contains(trimmedLine, "*/") {
			inJSDoc = false
			lastJSDocEnd = i
		}

		// Skip if inside JSDoc
		if inJSDoc {
			continue
		}

		var name string
		var symbolType string
		var isExported bool

		if matches := funcPattern.FindStringSubmatch(trimmedLine); len(matches) > 1 {
			name = matches[1]
			symbolType = "function"
			isExported = strings.HasPrefix(trimmedLine, "export")
		} else if matches := classPattern.FindStringSubmatch(trimmedLine); len(matches) > 1 {
			name = matches[1]
			symbolType = "class"
			isExported = strings.HasPrefix(trimmedLine, "export")
		} else if matches := arrowFuncPattern.FindStringSubmatch(trimmedLine); len(matches) > 1 {
			name = matches[1]
			symbolType = "function"
			isExported = strings.HasPrefix(trimmedLine, "export")
		}

		if name != "" {
			if exportedOnly && !isExported {
				continue
			}

			// Check if has JSDoc (previous line ends JSDoc block)
			hasDoc := lastJSDocEnd == i-1 || (i > 0 && jsdocPattern.MatchString(lines[i-1]))

			if !hasDoc {
				importance := "low"
				if isExported {
					importance = "high"
				}

				symbols = append(symbols, UndocumentedSymbol{
					Name:       name,
					Type:       symbolType,
					FilePath:   file.Path,
					StartLine:  i + 1,
					Language:   file.Language,
					Signature:  trimmedLine,
					IsExported: isExported,
					Importance: importance,
				})
			}
		}
	}

	return symbols
}

// scanPythonFile scans a Python file for undocumented functions and classes
func (s *Scanner) scanPythonFile(file codeFile, exportedOnly bool) []UndocumentedSymbol {
	var symbols []UndocumentedSymbol

	lines := strings.Split(file.Content, "\n")

	funcPattern := regexp.MustCompile(`^def\s+(\w+)\s*\(`)
	classPattern := regexp.MustCompile(`^class\s+(\w+)`)
	docstringPattern := regexp.MustCompile(`^\s*("""|\''')`)

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		var name string
		var symbolType string

		if matches := funcPattern.FindStringSubmatch(trimmedLine); len(matches) > 1 {
			name = matches[1]
			symbolType = "function"
		} else if matches := classPattern.FindStringSubmatch(trimmedLine); len(matches) > 1 {
			name = matches[1]
			symbolType = "class"
		}

		if name != "" {
			// Skip private functions (starting with _)
			isPrivate := strings.HasPrefix(name, "_")
			if exportedOnly && isPrivate {
				continue
			}

			// Check if next line is a docstring
			hasDoc := i+1 < len(lines) && docstringPattern.MatchString(lines[i+1])

			if !hasDoc {
				importance := "medium"
				if !isPrivate && symbolType == "class" {
					importance = "high"
				}
				if isPrivate {
					importance = "low"
				}

				symbols = append(symbols, UndocumentedSymbol{
					Name:       name,
					Type:       symbolType,
					FilePath:   file.Path,
					StartLine:  i + 1,
					Language:   "python",
					Signature:  trimmedLine,
					IsExported: !isPrivate,
					Importance: importance,
				})
			}
		}
	}

	return symbols
}

// buildSummary builds the scan summary
func (s *Scanner) buildSummary(result *ScanResult) {
	fileStats := make(map[string]*FileStats)

	for _, sym := range result.Symbols {
		result.Summary.ByType[sym.Type]++
		result.Summary.ByLanguage[sym.Language]++
		result.Summary.ByImportance[sym.Importance]++

		if sym.IsExported {
			result.Summary.ExportedCount++
		}

		if _, ok := fileStats[sym.FilePath]; !ok {
			fileStats[sym.FilePath] = &FileStats{FilePath: sym.FilePath}
		}
		fileStats[sym.FilePath].Count++
		if sym.IsExported {
			fileStats[sym.FilePath].Exported++
		}
	}

	result.Summary.TotalSymbols = len(result.Symbols)

	// Top affected files
	var files []*FileStats
	for _, fs := range fileStats {
		files = append(files, fs)
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Count > files[j].Count
	})

	for i := 0; i < min(10, len(files)); i++ {
		result.Summary.TopAffectedFiles = append(result.Summary.TopAffectedFiles, *files[i])
	}
}

// Helper functions

func isCodeFile(path string) bool {
	extensions := []string{".go", ".js", ".ts", ".tsx", ".py", ".rs", ".java"}
	for _, ext := range extensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

func detectLanguage(path string) string {
	switch {
	case strings.HasSuffix(path, ".go"):
		return "go"
	case strings.HasSuffix(path, ".ts"), strings.HasSuffix(path, ".tsx"):
		return "typescript"
	case strings.HasSuffix(path, ".js"):
		return "javascript"
	case strings.HasSuffix(path, ".py"):
		return "python"
	default:
		return "unknown"
	}
}
