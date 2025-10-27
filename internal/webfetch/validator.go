package webfetch

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// URLValidator проверяет безопасность URLs
type URLValidator struct {
	config SecurityConfig
}

// NewURLValidator создает новый валидатор
func NewURLValidator(config SecurityConfig) *URLValidator {
	return &URLValidator{
		config: config,
	}
}

// Validate проверяет URL на безопасность
func (v *URLValidator) Validate(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Check scheme
	if !v.isAllowedScheme(u.Scheme) {
		return fmt.Errorf("scheme %s not allowed", u.Scheme)
	}

	// Check domain whitelist
	if len(v.config.AllowedDomains) > 0 {
		if !v.isDomainInList(u.Host, v.config.AllowedDomains) {
			return fmt.Errorf("domain %s not in whitelist", u.Host)
		}
	}

	// Check domain blacklist
	if v.isDomainInList(u.Host, v.config.BlockedDomains) {
		return fmt.Errorf("domain %s is blocked", u.Host)
	}

	// SSRF prevention: block private IPs
	if v.config.BlockPrivateIPs {
		if err := v.checkPrivateIP(u); err != nil {
			return err
		}
	}

	// Block localhost
	if v.config.BlockLocalhost {
		if v.isLocalhost(u.Hostname()) {
			return fmt.Errorf("localhost not allowed")
		}
	}

	return nil
}

// isAllowedScheme проверяет разрешенность схемы
func (v *URLValidator) isAllowedScheme(scheme string) bool {
	scheme = strings.ToLower(scheme)
	for _, s := range v.config.AllowedSchemes {
		if strings.ToLower(s) == scheme {
			return true
		}
	}
	return false
}

// isDomainInList проверяет наличие домена в списке
func (v *URLValidator) isDomainInList(host string, list []string) bool {
	host = strings.ToLower(host)

	// Remove port if present
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}

	for _, domain := range list {
		domain = strings.ToLower(domain)

		// Exact match
		if host == domain {
			return true
		}

		// Wildcard match (*.example.com)
		if strings.HasPrefix(domain, "*.") {
			suffix := domain[2:]
			if strings.HasSuffix(host, suffix) {
				return true
			}
		}
	}

	return false
}

// checkPrivateIP проверяет IP на приватность (SSRF protection)
func (v *URLValidator) checkPrivateIP(u *url.URL) error {
	hostname := u.Hostname()

	// Try to parse as IP
	ip := net.ParseIP(hostname)
	if ip != nil {
		if isPrivateIP(ip) {
			return fmt.Errorf("private IP addresses not allowed: %s", hostname)
		}
		return nil
	}

	// Resolve hostname to IP
	ips, err := net.LookupIP(hostname)
	if err != nil {
		// DNS resolution failed - allow (will fail at HTTP level)
		return nil
	}

	// Check all resolved IPs
	for _, resolvedIP := range ips {
		if isPrivateIP(resolvedIP) {
			return fmt.Errorf("hostname %s resolves to private IP: %s", hostname, resolvedIP)
		}
	}

	return nil
}

// isPrivateIP проверяет является ли IP приватным
func isPrivateIP(ip net.IP) bool {
	// Loopback
	if ip.IsLoopback() {
		return true
	}

	// Private ranges
	if ip.IsPrivate() {
		return true
	}

	// Link-local
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	// Multicast
	if ip.IsMulticast() {
		return true
	}

	// Unspecified (0.0.0.0 or ::)
	if ip.IsUnspecified() {
		return true
	}

	// Check specific ranges not covered by IsPrivate()
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16", // Link-local
		"127.0.0.0/8",    // Loopback
		"::1/128",        // IPv6 loopback
		"fc00::/7",       // IPv6 unique local
		"fe80::/10",      // IPv6 link-local
	}

	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

// isLocalhost проверяет является ли hostname localhost
func (v *URLValidator) isLocalhost(hostname string) bool {
	hostname = strings.ToLower(hostname)

	localhostVariants := []string{
		"localhost",
		"127.0.0.1",
		"::1",
		"0.0.0.0",
		"::",
	}

	for _, variant := range localhostVariants {
		if hostname == variant {
			return true
		}
	}

	return false
}

