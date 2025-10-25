// Package ldap provides LDAP/Active Directory authentication client
// Version: 1.11.3+ (Enterprise Suite - LDAP Integration)
package ldap

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/config"
)

// Client представляет LDAP client для аутентификации
type Client struct {
	config *config.LDAPConfig
	logger *logrus.Logger
}

// NewClient создает новый LDAP client
func NewClient(cfg *config.LDAPConfig, logger *logrus.Logger) (*Client, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("LDAP is not enabled")
	}

	if logger == nil {
		logger = logrus.New()
	}

	// Validate configuration
	if cfg.URL == "" {
		return nil, fmt.Errorf("LDAP URL is required")
	}
	if cfg.BindDN == "" {
		return nil, fmt.Errorf("LDAP Bind DN is required")
	}
	if cfg.BindPassword == "" {
		return nil, fmt.Errorf("LDAP Bind Password is required")
	}
	if cfg.UserBaseDN == "" {
		return nil, fmt.Errorf("LDAP User Base DN is required")
	}

	// Set defaults
	if cfg.UserFilter == "" {
		cfg.UserFilter = "(uid={username})"
	}
	if cfg.UserIDAttribute == "" {
		cfg.UserIDAttribute = "uid"
	}
	if cfg.UserEmailAttribute == "" {
		cfg.UserEmailAttribute = "mail"
	}
	if cfg.UserNameAttribute == "" {
		cfg.UserNameAttribute = "cn"
	}
	if cfg.GroupFilter == "" {
		cfg.GroupFilter = "(member={userdn})"
	}
	if cfg.GroupNameAttribute == "" {
		cfg.GroupNameAttribute = "cn"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	logger.WithFields(logrus.Fields{
		"url":          cfg.URL,
		"user_base_dn": cfg.UserBaseDN,
	}).Info("LDAP client initialized")

	return &Client{
		config: cfg,
		logger: logger,
	}, nil
}

// Authenticate выполняет LDAP bind authentication
func (c *Client) Authenticate(username, password string) (*User, error) {
	if username == "" || password == "" {
		return nil, fmt.Errorf("username and password are required")
	}

	c.logger.WithField("username", username).Debug("Attempting LDAP authentication")

	// Connect to LDAP server
	conn, err := c.connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP server: %w", err)
	}
	defer conn.Close()

	// Bind as service account
	if err := conn.Bind(c.config.BindDN, c.config.BindPassword); err != nil {
		c.logger.WithError(err).Error("Failed to bind with service account")
		return nil, fmt.Errorf("LDAP bind failed: %w", err)
	}

	// Search for user
	userFilter := strings.ReplaceAll(c.config.UserFilter, "{username}", ldap.EscapeFilter(username))
	searchRequest := ldap.NewSearchRequest(
		c.config.UserBaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, // Size limit (0 = no limit)
		int(c.config.Timeout.Seconds()),
		false, // Types only
		userFilter,
		[]string{"dn", c.config.UserIDAttribute, c.config.UserEmailAttribute, c.config.UserNameAttribute},
		nil,
	)

	c.logger.WithFields(logrus.Fields{
		"base_dn": c.config.UserBaseDN,
		"filter":  userFilter,
	}).Debug("Searching for user")

	searchResult, err := conn.Search(searchRequest)
	if err != nil {
		c.logger.WithError(err).Error("Failed to search user")
		return nil, fmt.Errorf("LDAP search failed: %w", err)
	}

	if len(searchResult.Entries) == 0 {
		c.logger.WithField("username", username).Warn("User not found in LDAP")
		return nil, fmt.Errorf("user not found")
	}

	if len(searchResult.Entries) > 1 {
		c.logger.WithField("username", username).Error("Multiple users found")
		return nil, fmt.Errorf("multiple users found")
	}

	entry := searchResult.Entries[0]
	userDN := entry.DN

	c.logger.WithField("user_dn", userDN).Debug("User found, attempting bind")

	// Try to bind as user (authenticate)
	if err := conn.Bind(userDN, password); err != nil {
		c.logger.WithError(err).Warn("User authentication failed")
		return nil, fmt.Errorf("invalid credentials")
	}

	c.logger.WithField("user_dn", userDN).Info("User authenticated successfully")

	// Rebind as service account for group search
	if err := conn.Bind(c.config.BindDN, c.config.BindPassword); err != nil {
		c.logger.WithError(err).Error("Failed to rebind as service account")
		return nil, fmt.Errorf("failed to rebind: %w", err)
	}

	// Build user object
	user := &User{
		DN:       userDN,
		Username: entry.GetAttributeValue(c.config.UserIDAttribute),
		Email:    entry.GetAttributeValue(c.config.UserEmailAttribute),
		FullName: entry.GetAttributeValue(c.config.UserNameAttribute),
	}

	// Fetch groups (optional)
	if c.config.GroupBaseDN != "" {
		groups, err := c.getUserGroups(conn, userDN)
		if err != nil {
			c.logger.WithError(err).Warn("Failed to fetch user groups")
			// Don't fail authentication if group search fails
		} else {
			user.Groups = groups
			c.logger.WithFields(logrus.Fields{
				"user_dn":     userDN,
				"groups_count": len(groups),
			}).Debug("User groups retrieved")
		}
	}

	return user, nil
}

// connect устанавливает соединение с LDAP сервером
func (c *Client) connect() (*ldap.Conn, error) {
	// Parse URL
	u, err := url.Parse(c.config.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid LDAP URL: %w", err)
	}

	var conn *ldap.Conn

	if u.Scheme == "ldaps" {
		// LDAPS (LDAP over TLS)
		tlsConfig, err := c.buildTLSConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to build TLS config: %w", err)
		}

		c.logger.WithField("host", u.Host).Debug("Connecting to LDAPS")
		conn, err = ldap.DialTLS("tcp", u.Host, tlsConfig)
		if err != nil {
			return nil, fmt.Errorf("LDAPS connection failed: %w", err)
		}
	} else {
		// Plain LDAP
		c.logger.WithField("host", u.Host).Debug("Connecting to LDAP")
		conn, err = ldap.Dial("tcp", u.Host)
		if err != nil {
			return nil, fmt.Errorf("LDAP connection failed: %w", err)
		}

		// StartTLS if enabled
		if c.config.StartTLS {
			tlsConfig, err := c.buildTLSConfig()
			if err != nil {
				conn.Close()
				return nil, fmt.Errorf("failed to build TLS config: %w", err)
			}

			c.logger.Debug("Starting TLS")
			if err := conn.StartTLS(tlsConfig); err != nil {
				conn.Close()
				return nil, fmt.Errorf("StartTLS failed: %w", err)
			}
		}
	}

	return conn, nil
}

// buildTLSConfig создает TLS конфигурацию
func (c *Client) buildTLSConfig() (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: c.config.SkipVerify,
	}

	// Load CA certificate if provided
	if c.config.CACertFile != "" {
		caCert, err := os.ReadFile(c.config.CACertFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA cert file: %w", err)
		}

		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		tlsConfig.RootCAs = certPool
		c.logger.Debug("CA certificate loaded")
	}

	if c.config.SkipVerify {
		c.logger.Warn("TLS certificate verification is DISABLED - this is insecure!")
	}

	return tlsConfig, nil
}

// getUserGroups получает список групп пользователя
func (c *Client) getUserGroups(conn *ldap.Conn, userDN string) ([]string, error) {
	if c.config.GroupBaseDN == "" {
		return nil, nil
	}

	groupFilter := strings.ReplaceAll(c.config.GroupFilter, "{userdn}", ldap.EscapeFilter(userDN))
	searchRequest := ldap.NewSearchRequest(
		c.config.GroupBaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, // Size limit
		int(c.config.Timeout.Seconds()),
		false, // Types only
		groupFilter,
		[]string{c.config.GroupNameAttribute},
		nil,
	)

	c.logger.WithFields(logrus.Fields{
		"base_dn": c.config.GroupBaseDN,
		"filter":  groupFilter,
	}).Debug("Searching for user groups")

	searchResult, err := conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("group search failed: %w", err)
	}

	var groups []string
	for _, entry := range searchResult.Entries {
		groupName := entry.GetAttributeValue(c.config.GroupNameAttribute)
		if groupName != "" {
			groups = append(groups, groupName)
		}
	}

	return groups, nil
}

// User представляет LDAP пользователя
type User struct {
	DN       string   // Distinguished Name
	Username string   // Username (uid или sAMAccountName)
	Email    string   // Email address
	FullName string   // Full name
	Groups   []string // LDAP groups
}

// TestConnection проверяет соединение с LDAP сервером
func (c *Client) TestConnection() error {
	c.logger.Info("Testing LDAP connection...")

	conn, err := c.connect()
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	if err := conn.Bind(c.config.BindDN, c.config.BindPassword); err != nil {
		return fmt.Errorf("bind failed: %w", err)
	}

	c.logger.Info("LDAP connection test successful")
	return nil
}

