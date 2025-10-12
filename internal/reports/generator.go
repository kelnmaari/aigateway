package reports

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/observability"
	"ollama-openai-proxy/internal/storage"
)

//go:embed templates/*.html
var templatesFS embed.FS

// ReportGenerator generates various types of reports.
type ReportGenerator struct {
	db               storage.Database
	perfMonitor      *observability.PerformanceMonitor
	logger           *logrus.Logger
	templates        *template.Template
	startTime        time.Time
	lastBackupTime   *time.Time
}

// NewReportGenerator creates a new ReportGenerator instance.
func NewReportGenerator(
	db storage.Database,
	perfMonitor *observability.PerformanceMonitor,
	logger *logrus.Logger,
	startTime time.Time,
) (*ReportGenerator, error) {
	// Parse embedded templates
	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return &ReportGenerator{
		db:          db,
		perfMonitor: perfMonitor,
		logger:      logger,
		templates:   tmpl,
		startTime:   startTime,
	}, nil
}

// Generate creates a report based on type and period.
//
// Parameters:
//   - ctx: Context for database operations
//   - reportType: Type of report to generate
//   - start: Start time for report period
//   - end: End time for report period
//
// Returns generated report or error.
func (g *ReportGenerator) Generate(
	ctx context.Context,
	reportType ReportType,
	start, end time.Time,
) (*Report, error) {
	g.logger.WithFields(logrus.Fields{
		"type":  reportType,
		"start": start.Format("2006-01-02"),
		"end":   end.Format("2006-01-02"),
	}).Info("Generating report")

	report := &Report{
		Type:        reportType,
		Period:      fmt.Sprintf("%s - %s", start.Format("2006-01-02"), end.Format("2006-01-02")),
		GeneratedAt: time.Now(),
	}

	var err error
	switch reportType {
	case ReportTypeUsage:
		report.Data, err = g.generateUsageReport(ctx, start, end)
		if err != nil {
			return nil, fmt.Errorf("failed to generate usage report: %w", err)
		}
		report.HTML, err = g.renderTemplate("usage.html", struct {
			*UsageReportData
			Period      string
			GeneratedAt time.Time
		}{
			UsageReportData: report.Data.(*UsageReportData),
			Period:          report.Period,
			GeneratedAt:     report.GeneratedAt,
		})

	case ReportTypePerformance:
		report.Data, err = g.generatePerformanceReport(ctx, start, end)
		if err != nil {
			return nil, fmt.Errorf("failed to generate performance report: %w", err)
		}
		report.HTML, err = g.renderTemplate("performance.html", struct {
			*PerformanceReportData
			Period      string
			GeneratedAt time.Time
		}{
			PerformanceReportData: report.Data.(*PerformanceReportData),
			Period:                report.Period,
			GeneratedAt:           report.GeneratedAt,
		})

	case ReportTypeSystemHealth:
		report.Data, err = g.generateSystemHealthReport(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to generate system health report: %w", err)
		}
		report.HTML, err = g.renderTemplate("system_health.html", report.Data)

	default:
		return nil, fmt.Errorf("unsupported report type: %s", reportType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	g.logger.WithField("type", reportType).Info("Report generated successfully")
	return report, nil
}

// generateUsageReport generates usage statistics report.
func (g *ReportGenerator) generateUsageReport(ctx context.Context, start, end time.Time) (*UsageReportData, error) {
	if g.db == nil {
		return &UsageReportData{
			Period:        fmt.Sprintf("%s - %s", start.Format("2006-01-02"), end.Format("2006-01-02")),
			TotalRequests: 0,
		}, nil
	}

	// Get usage statistics from database
	stats, err := g.db.GetUsageStats(ctx, start, end)
	if err != nil {
		g.logger.WithError(err).Error("Failed to get usage stats from database")
		// Return empty data instead of failing
		return &UsageReportData{
			Period: fmt.Sprintf("%s - %s", start.Format("2006-01-02"), end.Format("2006-01-02")),
		}, nil
	}

	// Calculate error rate
	errorRate := 0.0
	if stats.TotalRequests > 0 {
		errorRate = float64(stats.FailedRequests) / float64(stats.TotalRequests) * 100
	}

	return &UsageReportData{
		Period:             fmt.Sprintf("%s - %s", start.Format("2006-01-02"), end.Format("2006-01-02")),
		TotalRequests:      stats.TotalRequests,
		SuccessfulRequests: stats.SuccessfulRequests,
		FailedRequests:     stats.FailedRequests,
		TotalTokens:        stats.TotalTokens,
		UniqueUsers:        stats.UniqueUsers,
		UniqueModels:       stats.UniqueModels,
		ErrorRate:          errorRate,
		TopModels:          convertToModelUsage(stats.TopModels),
		TopUsers:           convertToUserUsage(stats.TopUsers),
	}, nil
}

// generatePerformanceReport generates performance metrics report.
func (g *ReportGenerator) generatePerformanceReport(ctx context.Context, start, end time.Time) (*PerformanceReportData, error) {
	if g.db == nil {
		return &PerformanceReportData{
			Period: fmt.Sprintf("%s - %s", start.Format("2006-01-02"), end.Format("2006-01-02")),
		}, nil
	}

	// Get performance metrics from database
	stats, err := g.db.GetPerformanceStats(ctx, start, end)
	if err != nil {
		g.logger.WithError(err).Error("Failed to get performance stats from database")
		return &PerformanceReportData{
			Period: fmt.Sprintf("%s - %s", start.Format("2006-01-02"), end.Format("2006-01-02")),
		}, nil
	}

	// Calculate error rate
	errorRate := 0.0
	if stats.TotalRequests > 0 {
		errorRate = float64(stats.ErrorCount) / float64(stats.TotalRequests) * 100
	}

	return &PerformanceReportData{
		Period:               fmt.Sprintf("%s - %s", start.Format("2006-01-02"), end.Format("2006-01-02")),
		TotalRequests:        stats.TotalRequests,
		AvgLatencyMS:         stats.AvgLatencyMS,
		P50LatencyMS:         stats.P50LatencyMS,
		P95LatencyMS:         stats.P95LatencyMS,
		P99LatencyMS:         stats.P99LatencyMS,
		SlowRequestsCount:    stats.SlowRequestsCount,
		SlowRequestThreshold: 5 * time.Second, // Default threshold
		ErrorCount:           stats.ErrorCount,
		ErrorRate:            errorRate,
		RequestsPerSecond:    stats.RequestsPerSecond,
		FastestRequest:       stats.FastestRequest,
		SlowestRequest:       stats.SlowestRequest,
	}, nil
}

// generateSystemHealthReport generates system health report.
func (g *ReportGenerator) generateSystemHealthReport(ctx context.Context) (*SystemHealthReportData, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	currentMemoryMB := float64(m.HeapAlloc) / 1024 / 1024
	currentGoroutines := runtime.NumGoroutine()

	data := &SystemHealthReportData{
		Timestamp:         time.Now(),
		Uptime:            time.Since(g.startTime).Round(time.Second),
		CurrentMemoryMB:   currentMemoryMB,
		AvgMemoryMB:       currentMemoryMB, // Will be improved with actual metrics
		PeakMemoryMB:      currentMemoryMB,
		CurrentGoroutines: currentGoroutines,
		AvgGoroutines:     currentGoroutines,
		PeakGoroutines:    currentGoroutines,
		OllamaStatus:      "unknown",
	}

	// Get performance monitor metrics if available
	if g.perfMonitor != nil {
		metrics := g.perfMonitor.GetMetrics()
		if metrics != nil {
			data.CurrentMemoryMB = metrics.HeapAllocMB
			data.PeakMemoryMB = metrics.HeapSysMB
			data.CurrentGoroutines = metrics.NumGoroutines
		}
	}

	// Get database statistics if available
	if g.db != nil {
		// Count active users
		activeUsers, err := g.db.CountActiveUsers(ctx, 30*24*time.Hour) // Active in last 30 days
		if err == nil {
			data.ActiveUsers = activeUsers
		}

		// Count total users
		totalUsers, err := g.db.CountTotalUsers(ctx)
		if err == nil {
			data.TotalUsers = totalUsers
		}

		// Count active API keys
		activeKeys, err := g.db.CountActiveAPIKeys(ctx)
		if err == nil {
			data.ActiveAPIKeys = activeKeys
		}
	}

	// Set last backup time if available
	if g.lastBackupTime != nil {
		data.LastBackup = g.lastBackupTime
	}

	return data, nil
}

// renderTemplate renders HTML template with data.
func (g *ReportGenerator) renderTemplate(name string, data interface{}) (string, error) {
	var buf bytes.Buffer
	err := g.templates.ExecuteTemplate(&buf, name, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}
	return buf.String(), nil
}

// Helper functions for data conversion
func convertToModelUsage(models []struct {
	Model    string
	Requests int64
	Tokens   int64
}) []ModelUsage {
	result := make([]ModelUsage, len(models))
	for i, m := range models {
		result[i] = ModelUsage{
			Model:    m.Model,
			Requests: m.Requests,
			Tokens:   m.Tokens,
		}
	}
	return result
}

func convertToUserUsage(users []struct {
	Username string
	Requests int64
	Tokens   int64
}) []UserUsage {
	result := make([]UserUsage, len(users))
	for i, u := range users {
		result[i] = UserUsage{
			Username: u.Username,
			Requests: u.Requests,
			Tokens:   u.Tokens,
		}
	}
	return result
}

// SetLastBackupTime updates the last backup timestamp.
func (g *ReportGenerator) SetLastBackupTime(t time.Time) {
	g.lastBackupTime = &t
}

