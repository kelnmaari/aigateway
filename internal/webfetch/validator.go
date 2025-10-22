package webfetch

import (
	"fmt"
	"net"
	"net/url"
)

// URLValidator validates URLs for security and compliance with configuration
type URLValidator struct {
	config ValidationConfig
}

// NewURLValidator creates a new URL validator
func NewURLValidator(config ValidationConfig) *URLValidator {
	return &URLValidator{
		config: config,
	}
}

// Validate checks if the URL is safe to fetch according to the configuration
func (v *URLValidator) Validate(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Check scheme
	if !v.isAllowedScheme(u.Scheme) {
		return fmt.Errorf("scheme %s not allowed", u.Scheme)
	}

	// Check domain whitelist/blacklist
	if len(v.config.AllowedDomains) > 0 {
		if !v.isDomainAllowed(u.Host) {
			return fmt.Errorf("domain %s not in whitelist", u.Host)
		}
	}

	if v.isDomainBlocked(u.Host) {
		return fmt.Errorf("domain %s is blocked", u.Host)
	}

	// SSRF prevention
	if v.config.BlockPrivateIPs {
		if ip := net.ParseIP(u.Hostname()); ip != nil {
			if ip.IsPrivate() || ip.IsLoopback() {
				return fmt.Errorf("private IP addresses not allowed")
			}
		}
	}

	if v.config.BlockLocalhost {
		if u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" {
			return fmt.Errorf("localhost not allowed")
		}
	}

	return nil
}

// isAllowedScheme checks if the URL scheme is allowed
func (v *URLValidator) isAllowedScheme(scheme string) bool {
	for _, s := range v.config.AllowedSchemes {
		if s == scheme {
			return true
		}
	}
	return false
}

// isDomainAllowed checks if the domain is in the whitelist
func (v *URLValidator) isDomainAllowed(domain string) bool {
	for _, d := range v.config.AllowedDomains {
		if d == domain {
			return true
		}
	}
	return false
}

// isDomainBlocked checks if the domain is in the blacklist
func (v *URLValidator) isDomainBlocked(domain string) bool {
	for _, d := range v.config.BlockedDomains {
		if d == domain {
			return true
		}
	}
	return false
}
