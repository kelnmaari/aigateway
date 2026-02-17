// Package reports provides scheduled report generation and email delivery.
package reports

import "time"

// ReportType represents the type of report to generate.
type ReportType string

const (
	ReportTypeUsage        ReportType = "usage"         // Usage statistics report
	ReportTypePerformance  ReportType = "performance"   // Performance metrics report
	ReportTypeSystemHealth ReportType = "system_health" // System health report
	ReportTypeAll          ReportType = "all"           // Combined report
)

// ReportSchedule defines a scheduled report configuration.
type ReportSchedule struct {
	ID         string     `mapstructure:"id"`         // Unique identifier
	Name       string     `mapstructure:"name"`       // Human-readable name
	Type       ReportType `mapstructure:"type"`       // Report type
	Schedule   string     `mapstructure:"schedule"`   // Cron expression
	Recipients []string   `mapstructure:"recipients"` // Email addresses
	Enabled    bool       `mapstructure:"enabled"`    // Enable/disable schedule
}

// Report represents a generated report.
type Report struct {
	Type        ReportType  // Report type
	Period      string      // Time period covered
	GeneratedAt time.Time   // Generation timestamp
	Data        interface{} // Report-specific data
	HTML        string      // Rendered HTML content
}

// EmailReport represents an email to be sent.
type EmailReport struct {
	Recipients []string // Email addresses
	Subject    string   // Email subject
	Report     *Report  // Report content
}

// UsageReportData contains usage statistics.
type UsageReportData struct {
	Period             string
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	TotalTokens        int64
	UniqueUsers        int
	UniqueModels       int
	ErrorRate          float64
	TopModels          []ModelUsage
	TopUsers           []UserUsage
	RequestsByHour     []HourlyUsage
}

// ModelUsage represents usage statistics for a model.
type ModelUsage struct {
	Model    string
	Requests int64
	Tokens   int64
}

// UserUsage represents usage statistics for a user.
type UserUsage struct {
	Username string
	Requests int64
	Tokens   int64
}

// HourlyUsage represents usage statistics by hour.
type HourlyUsage struct {
	Hour     int
	Requests int64
}

// PerformanceReportData contains performance metrics.
type PerformanceReportData struct {
	Period              string
	TotalRequests       int64
	AvgLatencyMS        float64
	P50LatencyMS        float64
	P95LatencyMS        float64
	P99LatencyMS        float64
	SlowRequestsCount   int
	SlowRequestThreshold time.Duration
	ErrorCount          int
	ErrorRate           float64
	RequestsPerSecond   float64
	FastestRequest      float64
	SlowestRequest      float64
}

// SystemHealthReportData contains system health information.
type SystemHealthReportData struct {
	Timestamp         time.Time
	Uptime            time.Duration
	CurrentMemoryMB   float64
	AvgMemoryMB       float64
	PeakMemoryMB      float64
	CurrentGoroutines int
	AvgGoroutines     int
	PeakGoroutines    int
	DatabaseSizeMB    float64
	ActiveAPIKeys     int
	ActiveUsers       int
	TotalUsers        int
	InferenceStatus   string
	InferenceVersion  string
	LastBackup        *time.Time
}


