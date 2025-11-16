// Package handlers provides HTTP request handlers
// Version: v3.0.6+ - Auth helpers for JWT sessions
package handlers

import (
	"strings"
)

// detectDeviceType determines device type from User-Agent
func detectDeviceType(userAgent string) string {
	ua := strings.ToLower(userAgent)
	
	if strings.Contains(ua, "mobile") || strings.Contains(ua, "android") || strings.Contains(ua, "iphone") {
		return "mobile"
	}
	
	if strings.Contains(ua, "electron") || strings.Contains(ua, "desktop") {
		return "desktop"
	}
	
	return "web"
}

// parseDeviceName extracts browser/app name from User-Agent
func parseDeviceName(userAgent string) string {
	ua := userAgent
	
	// Mobile apps
	if strings.Contains(ua, "Electron") {
		return "Desktop App"
	}
	
	// Browsers
	if strings.Contains(ua, "Chrome") && !strings.Contains(ua, "Edg") {
		return "Chrome"
	}
	if strings.Contains(ua, "Firefox") {
		return "Firefox"
	}
	if strings.Contains(ua, "Safari") && !strings.Contains(ua, "Chrome") {
		return "Safari"
	}
	if strings.Contains(ua, "Edg") {
		return "Edge"
	}
	if strings.Contains(ua, "Opera") || strings.Contains(ua, "OPR") {
		return "Opera"
	}
	
	// Mobile browsers
	if strings.Contains(ua, "Mobile") {
		return "Mobile Browser"
	}
	
	return "Unknown"
}

