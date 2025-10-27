package webfetch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestURLValidator_Validate(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		config    SecurityConfig
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid https URL",
			url:  "https://example.com/page",
			config: SecurityConfig{
				AllowedSchemes:  []string{"http", "https"},
				BlockPrivateIPs: false,
			},
			wantError: false,
		},
		{
			name: "invalid scheme",
			url:  "ftp://example.com",
			config: SecurityConfig{
				AllowedSchemes: []string{"http", "https"},
			},
			wantError: true,
			errorMsg:  "scheme ftp not allowed",
		},
		{
			name: "localhost blocked",
			url:  "http://localhost:8080",
			config: SecurityConfig{
				AllowedSchemes: []string{"http", "https"},
				BlockLocalhost: true,
			},
			wantError: true,
			errorMsg:  "localhost not allowed",
		},
		{
			name: "127.0.0.1 blocked",
			url:  "http://127.0.0.1",
			config: SecurityConfig{
				AllowedSchemes: []string{"http", "https"},
				BlockLocalhost: true,
			},
			wantError: true,
			errorMsg:  "localhost not allowed",
		},
		{
			name: "private IP blocked (10.x.x.x)",
			url:  "http://10.0.0.1",
			config: SecurityConfig{
				AllowedSchemes:  []string{"http", "https"},
				BlockPrivateIPs: true,
			},
			wantError: true,
			errorMsg:  "private IP",
		},
		{
			name: "private IP blocked (192.168.x.x)",
			url:  "http://192.168.1.1",
			config: SecurityConfig{
				AllowedSchemes:  []string{"http", "https"},
				BlockPrivateIPs: true,
			},
			wantError: true,
			errorMsg:  "private IP",
		},
		{
			name: "domain whitelist - allowed",
			url:  "https://example.com",
			config: SecurityConfig{
				AllowedSchemes: []string{"http", "https"},
				AllowedDomains: []string{"example.com", "test.com"},
			},
			wantError: false,
		},
		{
			name: "domain whitelist - not allowed",
			url:  "https://blocked.com",
			config: SecurityConfig{
				AllowedSchemes: []string{"http", "https"},
				AllowedDomains: []string{"example.com"},
			},
			wantError: true,
			errorMsg:  "not in whitelist",
		},
		{
			name: "domain blacklist",
			url:  "https://malicious.com",
			config: SecurityConfig{
				AllowedSchemes: []string{"http", "https"},
				BlockedDomains: []string{"malicious.com"},
			},
			wantError: true,
			errorMsg:  "is blocked",
		},
		{
			name: "wildcard domain allowed",
			url:  "https://api.example.com",
			config: SecurityConfig{
				AllowedSchemes: []string{"http", "https"},
				AllowedDomains: []string{"*.example.com"},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewURLValidator(tt.config)
			err := validator.Validate(tt.url)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func BenchmarkURLValidator_Validate(b *testing.B) {
	validator := NewURLValidator(SecurityConfig{
		AllowedSchemes:  []string{"http", "https"},
		BlockPrivateIPs: true,
		BlockLocalhost:  true,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate("https://example.com/page")
	}
}

