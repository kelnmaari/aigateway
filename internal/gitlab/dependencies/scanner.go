// Package dependencies provides dependency scanning and analysis functionality.
package dependencies

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"aigateway/internal/gitlab/dependencies/parser"
	"aigateway/internal/gitlab/dependencies/registry"
	"aigateway/internal/gitlab/dependencies/security"
	"aigateway/internal/rag/vector"

	"github.com/sirupsen/logrus"
)

// Scanner scans project dependencies for updates and vulnerabilities.
type Scanner struct {
	vectorStore *vector.QdrantStore

	// Parsers
	goModParser *parser.GoModParser
	npmParser   *parser.NPMParser
	pipParser   *parser.PipParser

	// Registry clients
	golangClient *registry.GolangClient
	npmClient    *registry.NPMClient
	pypiClient   *registry.PyPIClient

	// Security
	osvClient *security.OSVClient

	logger *logrus.Logger
}

// NewScanner creates a new dependency scanner.
func NewScanner(vectorStore *vector.QdrantStore, logger *logrus.Logger) *Scanner {
	return &Scanner{
		vectorStore:  vectorStore,
		goModParser:  parser.NewGoModParser(),
		npmParser:    parser.NewNPMParser(),
		pipParser:    parser.NewPipParser(),
		golangClient: registry.NewGolangClient(logger),
		npmClient:    registry.NewNPMClient(logger),
		pypiClient:   registry.NewPyPIClient(logger),
		osvClient:    security.NewOSVClient(logger),
		logger:       logger,
	}
}

// ScanRequest contains parameters for a dependency scan.
type ScanRequest struct {
	ProjectID      string `json:"project_id"`
	CollectionName string `json:"collection_name"`
}

// depFileSpec defines how to find and parse a dependency file
type depFileSpec struct {
	filename  string
	language  string
	ecosystem string // OSV ecosystem name
}

// ScanProject scans a project's dependencies from the indexed code.
func (s *Scanner) ScanProject(ctx context.Context, req ScanRequest) (*ScanResult, error) {
	startTime := time.Now()
	scanID := fmt.Sprintf("dep-%s", time.Now().Format("20060102-150405"))

	s.logger.WithFields(logrus.Fields{
		"project_id": req.ProjectID,
		"collection": req.CollectionName,
		"scan_id":    scanID,
	}).Info("Starting dependency scan")

	result := &ScanResult{
		ProjectID:    req.ProjectID,
		ScanID:       scanID,
		ScannedAt:    startTime,
		Status:       "running",
		Dependencies: []DependencyWithVulns{},
		Summary: ScanSummary{
			ByUpdateType: make(map[string]int),
			BySeverity:   make(map[string]int),
		},
	}

	// Try to find dependency files in priority order
	depFiles := []depFileSpec{
		{"go.mod", "go", "Go"},
		{"package.json", "nodejs", "npm"},
		{"requirements.txt", "python", "PyPI"},
	}

	var deps []parser.Dependency
	var foundFile bool

	for _, spec := range depFiles {
		content, filePath, err := s.findDependencyFile(ctx, req.CollectionName, req.ProjectID, spec.filename)
		if err != nil {
			continue // Try next file type
		}

		s.logger.WithFields(logrus.Fields{
			"file":     spec.filename,
			"language": spec.language,
		}).Debug("Found dependency file")

		result.Language = spec.language
		result.FilePath = filePath

		// Parse based on language
		switch spec.language {
		case "go":
			deps, err = s.goModParser.Parse(content)
		case "nodejs":
			deps, err = s.npmParser.Parse(content)
		case "python":
			deps, err = s.pipParser.Parse(content)
		}

		if err != nil {
			s.logger.WithError(err).WithField("file", spec.filename).Warn("Failed to parse dependency file")
			continue
		}

		foundFile = true
		break // Found and parsed successfully
	}

	if !foundFile {
		result.Status = "completed"
		result.Error = "No dependency files found (go.mod, package.json, requirements.txt)"
		result.Duration = time.Since(startTime).String()
		return result, nil
	}

	s.logger.WithField("dependencies", len(deps)).Debug("Parsed dependencies")

	// Check versions and vulnerabilities concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex
	semaphore := make(chan struct{}, 10) // Limit concurrent requests

	for _, dep := range deps {
		wg.Add(1)
		go func(d parser.Dependency, lang string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			depWithVulns := s.checkDependencyForLanguage(ctx, d, lang)

			mu.Lock()
			result.Dependencies = append(result.Dependencies, depWithVulns)
			mu.Unlock()
		}(dep, result.Language)
	}

	wg.Wait()

	// Build summary
	s.buildSummary(result)

	result.Status = "completed"
	result.Duration = time.Since(startTime).String()

	s.logger.WithFields(logrus.Fields{
		"project_id":   req.ProjectID,
		"scan_id":      scanID,
		"total_deps":   result.Summary.TotalDependencies,
		"outdated":     result.Summary.OutdatedCount,
		"vulnerable":   result.Summary.VulnerableCount,
		"duration":     result.Duration,
	}).Info("Dependency scan completed")

	return result, nil
}

// findDependencyFile searches for a dependency file in the indexed code.
func (s *Scanner) findDependencyFile(ctx context.Context, collection, projectID, filename string) (string, string, error) {
	// Search for the file in Qdrant
	var content string
	var filePath string

	err := s.vectorStore.ScrollAll(ctx, collection, map[string]interface{}{
		"project_id": projectID,
	}, func(docs []vector.VectorDocument) error {
		for _, doc := range docs {
			fp, ok := doc.Metadata["file_path"].(string)
			if !ok {
				continue
			}
			
			// Check if this is the file we're looking for
			if fp == filename || strings.HasSuffix(fp, "/"+filename) {
				// Get content
				if c, ok := doc.Metadata["content"].(string); ok && c != "" {
					content = c
					filePath = fp
					return nil // Found, stop scrolling
				}
				if doc.Text != "" {
					content = doc.Text
					filePath = fp
					return nil
				}
			}
		}
		return nil
	})

	if err != nil {
		return "", "", err
	}

	if content == "" {
		return "", "", fmt.Errorf("file %s not found in index", filename)
	}

	return content, filePath, nil
}

// checkDependencyForLanguage checks a single dependency for updates and vulnerabilities.
func (s *Scanner) checkDependencyForLanguage(ctx context.Context, dep parser.Dependency, language string) DependencyWithVulns {
	result := DependencyWithVulns{
		Dependency: Dependency{
			Name:           dep.Name,
			CurrentVersion: dep.CurrentVersion,
			Indirect:       dep.Indirect,
		},
		Vulnerabilities: []Vulnerability{},
		IsVulnerable:    false,
	}

	// Get latest version based on language
	var latest string
	var updateType string
	var err error

	switch language {
	case "go":
		latest, err = s.golangClient.GetLatestVersion(ctx, dep.Name)
		if err == nil {
			updateType = s.golangClient.CompareVersions(dep.CurrentVersion, latest)
		}
	case "nodejs":
		latest, err = s.npmClient.GetLatestVersion(ctx, dep.Name)
		if err == nil {
			updateType = s.npmClient.CompareVersions(dep.CurrentVersion, latest)
		}
	case "python":
		latest, err = s.pypiClient.GetLatestVersion(ctx, dep.Name)
		if err == nil {
			updateType = s.pypiClient.CompareVersions(dep.CurrentVersion, latest)
		}
	default:
		err = fmt.Errorf("unsupported language: %s", language)
	}

	if err != nil {
		s.logger.WithError(err).WithField("package", dep.Name).Debug("Failed to get latest version")
		result.Dependency.LatestVersion = dep.CurrentVersion
		result.Dependency.UpdateType = "unknown"
	} else {
		result.Dependency.LatestVersion = latest
		result.Dependency.UpdateType = updateType
		result.Dependency.HasUpdate = updateType != "none" && updateType != "unknown"
	}

	// Check for vulnerabilities using OSV
	ecosystem := getOSVEcosystem(language)
	version := strings.TrimPrefix(dep.CurrentVersion, "v")
	
	vulns, err := s.osvClient.QueryVulnerabilities(ctx, ecosystem, dep.Name, version)
	if err != nil {
		s.logger.WithError(err).WithField("package", dep.Name).Debug("Failed to check vulnerabilities")
	} else {
		result.Vulnerabilities = vulns
		result.IsVulnerable = len(vulns) > 0
	}

	return result
}

// getOSVEcosystem returns the OSV ecosystem name for a language
func getOSVEcosystem(language string) string {
	switch language {
	case "go":
		return "Go"
	case "nodejs":
		return "npm"
	case "python":
		return "PyPI"
	default:
		return language
	}
}

// buildSummary builds summary statistics from the scan results.
func (s *Scanner) buildSummary(result *ScanResult) {
	for _, dep := range result.Dependencies {
		result.Summary.TotalDependencies++

		if !dep.Dependency.Indirect {
			result.Summary.DirectDependencies++
		}

		if dep.Dependency.HasUpdate {
			result.Summary.OutdatedCount++
			result.Summary.ByUpdateType[dep.Dependency.UpdateType]++
		} else {
			result.Summary.UpToDateCount++
		}

		if dep.IsVulnerable {
			result.Summary.VulnerableCount++
			for _, vuln := range dep.Vulnerabilities {
				result.Summary.BySeverity[vuln.Severity]++
				if vuln.Severity == "critical" {
					result.Summary.CriticalVulns++
				}
			}
		}
	}
}

// ScanProjectAllEcosystems scans ALL dependency files in a project (monorepo support).
func (s *Scanner) ScanProjectAllEcosystems(ctx context.Context, req ScanRequest) (*MultiEcosystemScanResult, error) {
	startTime := time.Now()
	scanID := fmt.Sprintf("dep-%s", time.Now().Format("20060102-150405"))

	s.logger.WithFields(logrus.Fields{
		"project_id": req.ProjectID,
		"collection": req.CollectionName,
		"scan_id":    scanID,
	}).Info("Starting multi-ecosystem dependency scan")

	result := &MultiEcosystemScanResult{
		ProjectID:  req.ProjectID,
		ScanID:     scanID,
		ScannedAt:  startTime,
		Status:     "running",
		Ecosystems: []ScanResult{},
		TotalSummary: ScanSummary{
			ByUpdateType: make(map[string]int),
			BySeverity:   make(map[string]int),
		},
	}

	// All supported dependency file specs
	depFiles := []depFileSpec{
		{"go.mod", "go", "Go"},
		{"package.json", "nodejs", "npm"},
		{"requirements.txt", "python", "PyPI"},
	}

	// Find ALL dependency files
	allFiles := s.findAllDependencyFiles(ctx, req.CollectionName, req.ProjectID, depFiles)

	if len(allFiles) == 0 {
		result.Status = "completed"
		result.Error = "No dependency files found (go.mod, package.json, requirements.txt)"
		result.Duration = time.Since(startTime).String()
		return result, nil
	}

	s.logger.WithField("files_found", len(allFiles)).Info("Found dependency files")

	// Scan each ecosystem concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex
	ecosystemResults := make(chan ScanResult, len(allFiles))

	for _, fileInfo := range allFiles {
		wg.Add(1)
		go func(fi foundDepFile) {
			defer wg.Done()

			ecosystemResult := s.scanSingleEcosystem(ctx, req.ProjectID, scanID, fi)
			ecosystemResults <- ecosystemResult
		}(fileInfo)
	}

	// Wait and collect results
	go func() {
		wg.Wait()
		close(ecosystemResults)
	}()

	for ecosystemResult := range ecosystemResults {
		mu.Lock()
		result.Ecosystems = append(result.Ecosystems, ecosystemResult)
		mu.Unlock()
	}

	// Build total summary from all ecosystems
	s.buildTotalSummary(result)

	result.Status = "completed"
	result.Duration = time.Since(startTime).String()

	s.logger.WithFields(logrus.Fields{
		"project_id":   req.ProjectID,
		"scan_id":      scanID,
		"ecosystems":   len(result.Ecosystems),
		"total_deps":   result.TotalSummary.TotalDependencies,
		"outdated":     result.TotalSummary.OutdatedCount,
		"vulnerable":   result.TotalSummary.VulnerableCount,
		"duration":     result.Duration,
	}).Info("Multi-ecosystem dependency scan completed")

	return result, nil
}

// foundDepFile represents a found dependency file with its content
type foundDepFile struct {
	spec     depFileSpec
	content  string
	filePath string
}

// findAllDependencyFiles searches for ALL dependency files in the indexed code.
func (s *Scanner) findAllDependencyFiles(ctx context.Context, collection, projectID string, specs []depFileSpec) []foundDepFile {
	var found []foundDepFile
	var mu sync.Mutex

	// Build a map of filenames we're looking for
	fileSpecMap := make(map[string]depFileSpec)
	for _, spec := range specs {
		fileSpecMap[spec.filename] = spec
	}

	err := s.vectorStore.ScrollAll(ctx, collection, map[string]interface{}{
		"project_id": projectID,
	}, func(docs []vector.VectorDocument) error {
		for _, doc := range docs {
			fp, ok := doc.Metadata["file_path"].(string)
			if !ok {
				continue
			}

			// Check if this is any of the dependency files we're looking for
			for filename, spec := range fileSpecMap {
				if fp == filename || strings.HasSuffix(fp, "/"+filename) {
					var content string
					if c, ok := doc.Metadata["content"].(string); ok && c != "" {
						content = c
					} else if doc.Text != "" {
						content = doc.Text
					}

					if content != "" {
						mu.Lock()
						found = append(found, foundDepFile{
							spec:     spec,
							content:  content,
							filePath: fp,
						})
						mu.Unlock()

						s.logger.WithFields(logrus.Fields{
							"file":     fp,
							"language": spec.language,
						}).Debug("Found dependency file")
					}
					break
				}
			}
		}
		return nil
	})

	if err != nil {
		s.logger.WithError(err).Error("Failed to search for dependency files")
	}

	return found
}

// scanSingleEcosystem scans dependencies for a single ecosystem file.
func (s *Scanner) scanSingleEcosystem(ctx context.Context, projectID, scanID string, fi foundDepFile) ScanResult {
	startTime := time.Now()

	result := ScanResult{
		ProjectID:    projectID,
		ScanID:       scanID,
		Language:     fi.spec.language,
		FilePath:     fi.filePath,
		ScannedAt:    startTime,
		Status:       "running",
		Dependencies: []DependencyWithVulns{},
		Summary: ScanSummary{
			ByUpdateType: make(map[string]int),
			BySeverity:   make(map[string]int),
		},
	}

	// Parse based on language
	var deps []parser.Dependency
	var err error

	switch fi.spec.language {
	case "go":
		deps, err = s.goModParser.Parse(fi.content)
	case "nodejs":
		deps, err = s.npmParser.Parse(fi.content)
	case "python":
		deps, err = s.pipParser.Parse(fi.content)
	}

	if err != nil {
		s.logger.WithError(err).WithField("file", fi.filePath).Warn("Failed to parse dependency file")
		result.Status = "failed"
		result.Error = err.Error()
		result.Duration = time.Since(startTime).String()
		return result
	}

	s.logger.WithFields(logrus.Fields{
		"file":         fi.filePath,
		"language":     fi.spec.language,
		"dependencies": len(deps),
	}).Debug("Parsed dependencies")

	// Check versions and vulnerabilities concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex
	semaphore := make(chan struct{}, 10) // Limit concurrent requests

	for _, dep := range deps {
		wg.Add(1)
		go func(d parser.Dependency) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			depWithVulns := s.checkDependencyForLanguage(ctx, d, fi.spec.language)

			mu.Lock()
			result.Dependencies = append(result.Dependencies, depWithVulns)
			mu.Unlock()
		}(dep)
	}

	wg.Wait()

	// Build summary
	s.buildSummary(&result)

	result.Status = "completed"
	result.Duration = time.Since(startTime).String()

	return result
}

// buildTotalSummary aggregates summary from all ecosystems.
func (s *Scanner) buildTotalSummary(result *MultiEcosystemScanResult) {
	for _, eco := range result.Ecosystems {
		result.TotalSummary.TotalDependencies += eco.Summary.TotalDependencies
		result.TotalSummary.DirectDependencies += eco.Summary.DirectDependencies
		result.TotalSummary.OutdatedCount += eco.Summary.OutdatedCount
		result.TotalSummary.VulnerableCount += eco.Summary.VulnerableCount
		result.TotalSummary.UpToDateCount += eco.Summary.UpToDateCount
		result.TotalSummary.CriticalVulns += eco.Summary.CriticalVulns

		for k, v := range eco.Summary.ByUpdateType {
			result.TotalSummary.ByUpdateType[k] += v
		}
		for k, v := range eco.Summary.BySeverity {
			result.TotalSummary.BySeverity[k] += v
		}
	}
}

