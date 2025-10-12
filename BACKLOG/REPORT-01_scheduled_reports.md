# REPORT-01: Scheduled Reports

**Версия:** v1.6.3  
**Приоритет:** LOW  
**Оценка времени:** 6-8 часов  
**Статус:** 📋 Не начато  

---

## 📋 Описание

Автоматическая генерация и отправка отчетов по email. Периодические отчеты по usage statistics, performance summaries, system health с настраиваемым расписанием.

## 🎯 Цели

1. **Scheduled Reports** - автоматическая генерация по расписанию
2. **Email Delivery** - отправка через SMTP
3. **Report Templates** - гибкие шаблоны отчетов
4. **Configurable Schedules** - daily, weekly, monthly

## 🔧 Технические детали

### 1. Report Scheduler

**Пакеты:**
```go
github.com/robfig/cron/v3
gopkg.in/gomail.v2
```

**Файл:** `internal/reports/scheduler.go`

```go
package reports

import (
	"context"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

type Scheduler struct {
	cron      *cron.Cron
	generator *ReportGenerator
	mailer    *EmailMailer
	logger    *logrus.Logger
	config    SchedulerConfig
}

type SchedulerConfig struct {
	Enabled   bool
	Schedules []ReportSchedule
}

type ReportSchedule struct {
	ID          string
	Name        string
	Type        ReportType      // usage, performance, system_health
	Schedule    string          // cron expression: "0 9 * * MON"
	Recipients  []string        // email addresses
	Enabled     bool
}

type ReportType string

const (
	ReportTypeUsage        ReportType = "usage"
	ReportTypePerformance  ReportType = "performance"
	ReportTypeSystemHealth ReportType = "system_health"
	ReportTypeAll          ReportType = "all"
)

func NewScheduler(
	generator *ReportGenerator,
	mailer *EmailMailer,
	logger *logrus.Logger,
	cfg SchedulerConfig,
) *Scheduler {
	return &Scheduler{
		cron:      cron.New(),
		generator: generator,
		mailer:    mailer,
		logger:    logger,
		config:    cfg,
	}
}

// Start initializes all scheduled reports
func (s *Scheduler) Start(ctx context.Context) error {
	if !s.config.Enabled {
		s.logger.Info("Report scheduler disabled")
		return nil
	}

	for _, schedule := range s.config.Schedules {
		if !schedule.Enabled {
			continue
		}

		// Add cron job for each schedule
		_, err := s.cron.AddFunc(schedule.Schedule, func() {
			s.executeReport(ctx, schedule)
		})
		if err != nil {
			return fmt.Errorf("failed to add schedule %s: %w", schedule.Name, err)
		}

		s.logger.WithFields(logrus.Fields{
			"name":     schedule.Name,
			"type":     schedule.Type,
			"schedule": schedule.Schedule,
		}).Info("Report schedule added")
	}

	s.cron.Start()
	s.logger.Info("Report scheduler started")
	
	return nil
}

// Stop halts the scheduler
func (s *Scheduler) Stop() {
	s.cron.Stop()
	s.logger.Info("Report scheduler stopped")
}

// executeReport generates and sends a report
func (s *Scheduler) executeReport(ctx context.Context, schedule ReportSchedule) {
	s.logger.WithField("schedule", schedule.Name).Info("Executing scheduled report")

	// Generate report
	report, err := s.generator.Generate(ctx, schedule.Type, time.Now().Add(-24*time.Hour), time.Now())
	if err != nil {
		s.logger.WithError(err).Error("Failed to generate report")
		return
	}

	// Send via email
	err = s.mailer.SendReport(ctx, EmailReport{
		Recipients: schedule.Recipients,
		Subject:    fmt.Sprintf("[Ollama Proxy] %s Report - %s", schedule.Type, time.Now().Format("2006-01-02")),
		Report:     report,
	})
	if err != nil {
		s.logger.WithError(err).Error("Failed to send report")
		return
	}

	s.logger.WithFields(logrus.Fields{
		"schedule":   schedule.Name,
		"recipients": len(schedule.Recipients),
	}).Info("Report sent successfully")
}
```

### 2. Report Generator

**Файл:** `internal/reports/generator.go`

```go
package reports

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"time"

	"internal/storage"
)

type ReportGenerator struct {
	storage  storage.Storage
	template *template.Template
}

type Report struct {
	Type        ReportType
	Period      string
	GeneratedAt time.Time
	Data        interface{}
	HTML        string
}

type UsageReportData struct {
	TotalRequests      int64
	TotalTokens        int64
	UniqueUsers        int
	TopModels          []ModelUsage
	TopUsers           []UserUsage
	RequestsByHour     []HourlyUsage
	ErrorRate          float64
}

type PerformanceReportData struct {
	AvgLatencyMS       float64
	P50LatencyMS       float64
	P95LatencyMS       float64
	P99LatencyMS       float64
	AvgTokensPerSec    float64
	RequestsPerSec     float64
	SlowRequestsCount  int
	ErrorCount         int
}

type SystemHealthReportData struct {
	Uptime             time.Duration
	AvgMemoryMB        float64
	PeakMemoryMB       float64
	AvgGoroutines      int
	DatabaseSize       int64
	ActiveAPIKeys      int
	ActiveUsers        int
	OllamaStatus       string
}

func NewReportGenerator(storage storage.Storage) (*ReportGenerator, error) {
	// Load HTML templates
	tmpl, err := template.ParseGlob("internal/reports/templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return &ReportGenerator{
		storage:  storage,
		template: tmpl,
	}, nil
}

// Generate creates a report based on type and period
func (g *ReportGenerator) Generate(ctx context.Context, reportType ReportType, start, end time.Time) (*Report, error) {
	report := &Report{
		Type:        reportType,
		Period:      fmt.Sprintf("%s - %s", start.Format("2006-01-02"), end.Format("2006-01-02")),
		GeneratedAt: time.Now(),
	}

	switch reportType {
	case ReportTypeUsage:
		data, err := g.generateUsageReport(ctx, start, end)
		if err != nil {
			return nil, err
		}
		report.Data = data
		report.HTML, err = g.renderTemplate("usage.html", data)
		if err != nil {
			return nil, err
		}

	case ReportTypePerformance:
		data, err := g.generatePerformanceReport(ctx, start, end)
		if err != nil {
			return nil, err
		}
		report.Data = data
		report.HTML, err = g.renderTemplate("performance.html", data)
		if err != nil {
			return nil, err
		}

	case ReportTypeSystemHealth:
		data, err := g.generateSystemHealthReport(ctx)
		if err != nil {
			return nil, err
		}
		report.Data = data
		report.HTML, err = g.renderTemplate("system_health.html", data)
		if err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("unsupported report type: %s", reportType)
	}

	return report, nil
}

func (g *ReportGenerator) generateUsageReport(ctx context.Context, start, end time.Time) (*UsageReportData, error) {
	// Query database for usage statistics
	// TODO: Implement database queries
	return &UsageReportData{
		TotalRequests: 1000,
		TotalTokens:   500000,
		UniqueUsers:   25,
		// ... more data
	}, nil
}

func (g *ReportGenerator) generatePerformanceReport(ctx context.Context, start, end time.Time) (*PerformanceReportData, error) {
	// Query metrics for performance data
	return &PerformanceReportData{
		AvgLatencyMS:  250.5,
		P95LatencyMS:  500.0,
		P99LatencyMS:  1000.0,
		// ... more data
	}, nil
}

func (g *ReportGenerator) generateSystemHealthReport(ctx context.Context) (*SystemHealthReportData, error) {
	// Get current system health
	return &SystemHealthReportData{
		Uptime:        24 * time.Hour,
		AvgMemoryMB:   512,
		PeakMemoryMB:  768,
		ActiveAPIKeys: 15,
		ActiveUsers:   10,
		// ... more data
	}, nil
}

func (g *ReportGenerator) renderTemplate(name string, data interface{}) (string, error) {
	var buf bytes.Buffer
	err := g.template.ExecuteTemplate(&buf, name, data)
	if err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}
	return buf.String(), nil
}
```

### 3. Email Mailer

**Файл:** `internal/reports/mailer.go`

```go
package reports

import (
	"context"
	"crypto/tls"
	"fmt"

	"gopkg.in/gomail.v2"
)

type EmailMailer struct {
	config SMTPConfig
}

type SMTPConfig struct {
	Enabled  bool
	Host     string
	Port     int
	Username string
	Password string
	From     string
	UseTLS   bool
}

type EmailReport struct {
	Recipients []string
	Subject    string
	Report     *Report
}

func NewEmailMailer(cfg SMTPConfig) *EmailMailer {
	return &EmailMailer{config: cfg}
}

// SendReport sends a report via email
func (m *EmailMailer) SendReport(ctx context.Context, emailReport EmailReport) error {
	if !m.config.Enabled {
		return fmt.Errorf("email mailer is disabled")
	}

	msg := gomail.NewMessage()
	msg.SetHeader("From", m.config.From)
	msg.SetHeader("To", emailReport.Recipients...)
	msg.SetHeader("Subject", emailReport.Subject)
	msg.SetBody("text/html", emailReport.Report.HTML)

	// Create dialer
	d := gomail.NewDialer(m.config.Host, m.config.Port, m.config.Username, m.config.Password)

	if m.config.UseTLS {
		d.TLSConfig = &tls.Config{
			ServerName: m.config.Host,
		}
	}

	// Send email
	if err := d.DialAndSend(msg); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
```

### 4. Report Templates

**Файл:** `internal/reports/templates/usage.html`

```html
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Usage Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: #2c3e50; color: white; padding: 20px; }
        .section { margin: 20px 0; padding: 15px; border: 1px solid #ddd; }
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 10px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background: #f5f5f5; }
        .metric { font-size: 24px; font-weight: bold; color: #2c3e50; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Usage Report</h1>
        <p>Period: {{.Period}}</p>
        <p>Generated: {{.GeneratedAt.Format "2006-01-02 15:04:05"}}</p>
    </div>

    <div class="section">
        <h2>Summary</h2>
        <p>Total Requests: <span class="metric">{{.TotalRequests}}</span></p>
        <p>Total Tokens: <span class="metric">{{.TotalTokens}}</span></p>
        <p>Unique Users: <span class="metric">{{.UniqueUsers}}</span></p>
        <p>Error Rate: <span class="metric">{{printf "%.2f" .ErrorRate}}%</span></p>
    </div>

    <div class="section">
        <h2>Top Models</h2>
        <table>
            <tr>
                <th>Model</th>
                <th>Requests</th>
                <th>Tokens</th>
            </tr>
            {{range .TopModels}}
            <tr>
                <td>{{.Model}}</td>
                <td>{{.Requests}}</td>
                <td>{{.Tokens}}</td>
            </tr>
            {{end}}
        </table>
    </div>

    <div class="section">
        <h2>Top Users</h2>
        <table>
            <tr>
                <th>User</th>
                <th>Requests</th>
                <th>Tokens</th>
            </tr>
            {{range .TopUsers}}
            <tr>
                <td>{{.Username}}</td>
                <td>{{.Requests}}</td>
                <td>{{.Tokens}}</td>
            </tr>
            {{end}}
        </table>
    </div>
</body>
</html>
```

## 📁 Структура файлов

```
internal/
├── reports/
│   ├── scheduler.go          # Cron scheduler
│   ├── scheduler_test.go
│   ├── generator.go          # Report generator
│   ├── generator_test.go
│   ├── mailer.go             # Email sender
│   ├── mailer_test.go
│   └── templates/
│       ├── usage.html        # Usage report template
│       ├── performance.html  # Performance report template
│       └── system_health.html
```

## 🧪 Тестирование

### Unit Tests

```go
func TestScheduler_Start(t *testing.T) {
	generator := &ReportGenerator{}
	mailer := &EmailMailer{}
	logger := logrus.New()
	
	cfg := SchedulerConfig{
		Enabled: true,
		Schedules: []ReportSchedule{
			{
				ID:          "daily-usage",
				Name:        "Daily Usage Report",
				Type:        ReportTypeUsage,
				Schedule:    "0 9 * * *",
				Recipients:  []string{"admin@example.com"},
				Enabled:     true,
			},
		},
	}
	
	scheduler := NewScheduler(generator, mailer, logger, cfg)
	err := scheduler.Start(context.Background())
	require.NoError(t, err)
	
	defer scheduler.Stop()
}

func TestReportGenerator_Generate(t *testing.T) {
	generator, err := NewReportGenerator(mockStorage)
	require.NoError(t, err)
	
	report, err := generator.Generate(
		context.Background(),
		ReportTypeUsage,
		time.Now().Add(-24*time.Hour),
		time.Now(),
	)
	require.NoError(t, err)
	assert.NotEmpty(t, report.HTML)
}
```

## 📝 Конфигурация

```yaml
# configs/dev.yaml
reports:
  enabled: true
  
  # SMTP configuration
  smtp:
    enabled: true
    host: "smtp.gmail.com"
    port: 587
    username: "your-email@gmail.com"
    password: "${SMTP_PASSWORD}"  # Use env variable
    from: "ollama-proxy@example.com"
    use_tls: true
  
  # Scheduled reports
  schedules:
    - id: "daily-usage"
      name: "Daily Usage Report"
      type: "usage"
      schedule: "0 9 * * *"  # Every day at 9 AM
      recipients:
        - "admin@example.com"
        - "team@example.com"
      enabled: true
      
    - id: "weekly-performance"
      name: "Weekly Performance Summary"
      type: "performance"
      schedule: "0 10 * * MON"  # Every Monday at 10 AM
      recipients:
        - "admin@example.com"
      enabled: true
      
    - id: "monthly-health"
      name: "Monthly System Health"
      type: "system_health"
      schedule: "0 9 1 * *"  # First day of month at 9 AM
      recipients:
        - "admin@example.com"
      enabled: true
```

## 🎯 Критерии успеха

- [ ] Scheduler корректно запускается
- [ ] Cron expressions работают правильно
- [ ] Report generator создает HTML отчеты
- [ ] Email mailer отправляет письма
- [ ] Templates рендерятся без ошибок
- [ ] SMTP authentication работает
- [ ] Unit tests покрывают >90%
- [ ] Документация обновлена

## 📖 Документация

**Обновить:**
- `docs/CONFIGURATION.md` - секция Reports
- `README.md` - упомянуть scheduled reports
- Добавить примеры cron expressions

**Примеры cron:**
```
0 9 * * *      # Daily at 9:00 AM
0 */6 * * *    # Every 6 hours
0 10 * * MON   # Every Monday at 10:00 AM
0 9 1 * *      # First day of month at 9:00 AM
0 9 * * 1-5    # Weekdays at 9:00 AM
```

## 🔗 Зависимости

**Go packages:**
```bash
go get github.com/robfig/cron/v3
go get gopkg.in/gomail.v2
```

## 🚀 Внедрение

**cmd/server/main.go:**
```go
// Initialize report generator
reportGenerator, err := reports.NewReportGenerator(storage)
if err != nil {
	log.Fatalf("Failed to initialize report generator: %v", err)
}

// Initialize email mailer
mailer := reports.NewEmailMailer(cfg.Reports.SMTP)

// Start report scheduler
scheduler := reports.NewScheduler(reportGenerator, mailer, logger, cfg.Reports)
err = scheduler.Start(context.Background())
if err != nil {
	log.Fatalf("Failed to start report scheduler: %v", err)
}
defer scheduler.Stop()
```

---

**Оценка времени:** 6-8 часов  
**Сложность:** MEDIUM  
**Зависимости:** Нет  
**Блокирует:** Нет

## 💡 Future Enhancements

- Slack/Discord webhooks вместо email
- PDF export для отчетов
- Custom report templates через WebUI
- Report preview в админке
- Webhook delivery для reports
- Multi-language support для templates

