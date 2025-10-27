package reports

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/sirupsen/logrus"
	"gopkg.in/gomail.v2"
)

// EmailMailer handles email delivery via SMTP.
type EmailMailer struct {
	config SMTPConfig
	logger *logrus.Logger
}

// SMTPConfig holds SMTP server configuration.
type SMTPConfig struct {
	Enabled  bool   `mapstructure:"enabled"`  // Enable/disable email sending
	Host     string `mapstructure:"host"`     // SMTP server host
	Port     int    `mapstructure:"port"`     // SMTP server port
	Username string `mapstructure:"username"` // SMTP username
	Password string `mapstructure:"password"` // SMTP password
	From     string `mapstructure:"from"`     // Sender email address
	UseTLS   bool   `mapstructure:"use_tls"`  // Use TLS/SSL
}

// NewEmailMailer creates a new EmailMailer instance.
func NewEmailMailer(cfg SMTPConfig, logger *logrus.Logger) *EmailMailer {
	return &EmailMailer{
		config: cfg,
		logger: logger,
	}
}

// SendReport sends a report via email to specified recipients.
//
// Parameters:
//   - ctx: Context for cancellation
//   - emailReport: Email report to send
//
// Returns error if email sending fails.
func (m *EmailMailer) SendReport(ctx context.Context, emailReport EmailReport) error {
	if !m.config.Enabled {
		m.logger.Warn("Email mailer is disabled, skipping email send")
		return nil
	}

	// Validate configuration
	if m.config.Host == "" || m.config.Port == 0 {
		return fmt.Errorf("invalid SMTP configuration: host or port missing")
	}

	if len(emailReport.Recipients) == 0 {
		return fmt.Errorf("no recipients specified")
	}

	m.logger.WithFields(logrus.Fields{
		"recipients": len(emailReport.Recipients),
		"subject":    emailReport.Subject,
		"report_type": emailReport.Report.Type,
	}).Info("Sending report via email")

	// Create email message
	msg := gomail.NewMessage()
	msg.SetHeader("From", m.config.From)
	msg.SetHeader("To", emailReport.Recipients...)
	msg.SetHeader("Subject", emailReport.Subject)
	msg.SetBody("text/html", emailReport.Report.HTML)

	// Create SMTP dialer
	d := gomail.NewDialer(
		m.config.Host,
		m.config.Port,
		m.config.Username,
		m.config.Password,
	)

	// Configure TLS if enabled
	if m.config.UseTLS {
		d.TLSConfig = &tls.Config{
			ServerName:         m.config.Host,
			InsecureSkipVerify: false,
		}
	}

	// Send email
	if err := d.DialAndSend(msg); err != nil {
		m.logger.WithError(err).Error("Failed to send email")
		return fmt.Errorf("failed to send email: %w", err)
	}

	m.logger.WithFields(logrus.Fields{
		"recipients": emailReport.Recipients,
		"subject":    emailReport.Subject,
	}).Info("Email sent successfully")

	return nil
}

// TestConnection tests SMTP connection without sending email.
func (m *EmailMailer) TestConnection() error {
	if !m.config.Enabled {
		return fmt.Errorf("email mailer is disabled")
	}

	d := gomail.NewDialer(
		m.config.Host,
		m.config.Port,
		m.config.Username,
		m.config.Password,
	)

	if m.config.UseTLS {
		d.TLSConfig = &tls.Config{
			ServerName:         m.config.Host,
			InsecureSkipVerify: false,
		}
	}

	// Test connection
	closer, err := d.Dial()
	if err != nil {
		return fmt.Errorf("SMTP connection test failed: %w", err)
	}
	defer closer.Close()

	m.logger.Info("SMTP connection test successful")
	return nil
}


