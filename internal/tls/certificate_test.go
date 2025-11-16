// Package tls - Certificate Manager Tests
// Version: v3.0.8
package tls

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCertificateGeneration tests self-signed certificate generation
func TestCertificateGeneration(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	cm := NewCertificateManager(tmpDir, logger)
	
	config := CertificateConfig{
		CommonName:   "test.local",
		Organization: "Test Org",
		ValidFor:     24 * time.Hour,
		Hosts:        []string{"test.local", "localhost", "127.0.0.1"},
	}
	
	// Generate certificate
	err := cm.EnsureCertificate("test.crt", "test.key", config)
	require.NoError(t, err)
	
	// Check files exist
	certPath, keyPath := cm.GetCertificatePaths("test.crt", "test.key")
	
	_, err = os.Stat(certPath)
	assert.NoError(t, err, "Certificate file should exist")
	
	_, err = os.Stat(keyPath)
	assert.NoError(t, err, "Key file should exist")
	
	// Validate certificate
	err = cm.ValidateCertificate(certPath)
	assert.NoError(t, err, "Certificate should be valid")
	
	t.Log("✅ Certificate generation works")
}

// TestCertificateReuse tests that existing certificate is reused
func TestCertificateReuse(t *testing.T) {
	tmpDir := t.TempDir()
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	cm := NewCertificateManager(tmpDir, logger)
	
	config := CertificateConfig{
		CommonName: "reuse.local",
		ValidFor:   24 * time.Hour,
	}
	
	// First generation
	err := cm.EnsureCertificate("reuse.crt", "reuse.key", config)
	require.NoError(t, err)
	
	certPath, _ := cm.GetCertificatePaths("reuse.crt", "reuse.key")
	
	// Get original modification time
	origInfo, err := os.Stat(certPath)
	require.NoError(t, err)
	origModTime := origInfo.ModTime()
	
	// Wait a bit
	time.Sleep(10 * time.Millisecond)
	
	// Second call (should reuse)
	err = cm.EnsureCertificate("reuse.crt", "reuse.key", config)
	require.NoError(t, err)
	
	// Check modification time (should be unchanged)
	newInfo, err := os.Stat(certPath)
	require.NoError(t, err)
	
	assert.Equal(t, origModTime, newInfo.ModTime(), "Certificate should be reused, not regenerated")
	
	t.Log("✅ Certificate reuse works")
}

// TestCertificateInfo tests certificate information extraction
func TestCertificateInfo(t *testing.T) {
	tmpDir := t.TempDir()
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	cm := NewCertificateManager(tmpDir, logger)
	
	config := CertificateConfig{
		CommonName:   "info.local",
		Organization: "Info Org",
		ValidFor:     48 * time.Hour,
		Hosts:        []string{"info.local", "*.info.local"},
	}
	
	err := cm.EnsureCertificate("info.crt", "info.key", config)
	require.NoError(t, err)
	
	certPath, _ := cm.GetCertificatePaths("info.crt", "info.key")
	
	// Get certificate info
	info, err := cm.GetCertificateInfo(certPath)
	require.NoError(t, err)
	
	assert.Equal(t, "info.local", info["subject"])
	assert.Contains(t, info["organization"], "Info Org")
	assert.True(t, info["is_self_signed"].(bool))
	assert.Greater(t, info["expires_in_days"].(int), 0)
	
	t.Logf("✅ Certificate info: %+v", info)
}

// TestCertificateDefaults tests default values
func TestCertificateDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	cm := NewCertificateManager(tmpDir, logger)
	
	// Empty config (should use defaults)
	config := CertificateConfig{}
	
	err := cm.EnsureCertificate("default.crt", "default.key", config)
	require.NoError(t, err)
	
	certPath, _ := cm.GetCertificatePaths("default.crt", "default.key")
	
	info, err := cm.GetCertificateInfo(certPath)
	require.NoError(t, err)
	
	// Check defaults
	assert.Equal(t, "localhost", info["subject"])
	assert.Contains(t, info["organization"], "AIGateway")
	assert.Greater(t, info["expires_in_days"].(int), 360, "Should be valid for ~1 year")
	
	t.Log("✅ Default certificate configuration works")
}

// TestCertificateValidation tests validation of expired/invalid certificates
func TestCertificateValidation(t *testing.T) {
	tmpDir := t.TempDir()
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	cm := NewCertificateManager(tmpDir, logger)
	
	// Generate short-lived certificate (for testing)
	config := CertificateConfig{
		CommonName: "short.local",
		ValidFor:   1 * time.Hour, // Short validity
	}
	
	err := cm.EnsureCertificate("short.crt", "short.key", config)
	require.NoError(t, err)
	
	certPath, _ := cm.GetCertificatePaths("short.crt", "short.key")
	
	// Should be valid now
	err = cm.ValidateCertificate(certPath)
	assert.NoError(t, err)
	
	t.Log("✅ Certificate validation works")
}

// TestInvalidCertificatePath tests handling of missing certificate
func TestInvalidCertificatePath(t *testing.T) {
	tmpDir := t.TempDir()
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	cm := NewCertificateManager(tmpDir, logger)
	
	// Try to validate non-existent certificate
	err := cm.ValidateCertificate(filepath.Join(tmpDir, "nonexistent.crt"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read certificate")
	
	t.Log("✅ Missing certificate handling works")
}

// BenchmarkCertificateGeneration benchmarks certificate generation
func BenchmarkCertificateGeneration(b *testing.B) {
	tmpDir := b.TempDir()
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	cm := NewCertificateManager(tmpDir, logger)
	
	config := CertificateConfig{
		CommonName: "bench.local",
		ValidFor:   24 * time.Hour,
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		certFile := "bench_" + string(rune(i)) + ".crt"
		keyFile := "bench_" + string(rune(i)) + ".key"
		
		_ = cm.generateSelfSignedCert(
			filepath.Join(tmpDir, certFile),
			filepath.Join(tmpDir, keyFile),
			config,
		)
	}
}

