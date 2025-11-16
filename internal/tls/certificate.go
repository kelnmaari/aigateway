// Package tls provides TLS certificate management and auto-generation
// Version: v3.0.8
package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

// CertificateManager manages TLS certificates
type CertificateManager struct {
	certDir string
	logger  *logrus.Logger
}

// CertificateConfig configuration for certificate generation
type CertificateConfig struct {
	CommonName       string        // CN (e.g., "localhost", "aigateway.local")
	Organization     string        // O
	OrganizationUnit string        // OU
	Country          string        // C
	Province         string        // ST
	Locality         string        // L
	ValidFor         time.Duration // Certificate validity period
	Hosts            []string      // DNS names and IP addresses
}

// NewCertificateManager creates a new certificate manager
func NewCertificateManager(certDir string, logger *logrus.Logger) *CertificateManager {
	if logger == nil {
		logger = logrus.New()
	}
	
	return &CertificateManager{
		certDir: certDir,
		logger:  logger,
	}
}

// EnsureCertificate ensures certificate exists, generates if missing
func (cm *CertificateManager) EnsureCertificate(certFile, keyFile string, config CertificateConfig) error {
	certPath := filepath.Join(cm.certDir, certFile)
	keyPath := filepath.Join(cm.certDir, keyFile)
	
	// Check if both files exist
	if _, err := os.Stat(certPath); err == nil {
		if _, err := os.Stat(keyPath); err == nil {
			cm.logger.WithFields(logrus.Fields{
				"cert": certPath,
				"key":  keyPath,
			}).Info("✅ TLS certificate found")
			return nil
		}
	}
	
	// Generate new self-signed certificate
	cm.logger.WithFields(logrus.Fields{
		"cert": certPath,
		"key":  keyPath,
		"cn":   config.CommonName,
	}).Info("🔐 Generating self-signed TLS certificate...")
	
	return cm.generateSelfSignedCert(certPath, keyPath, config)
}

// generateSelfSignedCert generates a self-signed certificate
func (cm *CertificateManager) generateSelfSignedCert(certPath, keyPath string, config CertificateConfig) error {
	// Ensure cert directory exists
	if err := os.MkdirAll(cm.certDir, 0755); err != nil {
		return fmt.Errorf("failed to create cert directory: %w", err)
	}
	
	// Set defaults
	if config.ValidFor == 0 {
		config.ValidFor = 365 * 24 * time.Hour // 1 year
	}
	if config.CommonName == "" {
		config.CommonName = "localhost"
	}
	if config.Organization == "" {
		config.Organization = "AIGateway"
	}
	if len(config.Hosts) == 0 {
		config.Hosts = []string{"localhost", "127.0.0.1", "::1"}
	}
	
	// Generate ECDSA private key (P-256)
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}
	
	// Generate serial number
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}
	
	// Create certificate template
	notBefore := time.Now()
	notAfter := notBefore.Add(config.ValidFor)
	
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:         config.CommonName,
			Organization:       []string{config.Organization},
			OrganizationalUnit: []string{config.OrganizationUnit},
			Country:            []string{config.Country},
			Province:           []string{config.Province},
			Locality:           []string{config.Locality},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              config.Hosts,
	}
	
	// Create self-signed certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}
	
	// Write certificate to file
	certFile, err := os.Create(certPath)
	if err != nil {
		return fmt.Errorf("failed to create cert file: %w", err)
	}
	defer certFile.Close()
	
	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return fmt.Errorf("failed to write cert: %w", err)
	}
	
	// Write private key to file
	keyFile, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create key file: %w", err)
	}
	defer keyFile.Close()
	
	privBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}
	
	if err := pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes}); err != nil {
		return fmt.Errorf("failed to write key: %w", err)
	}
	
	cm.logger.WithFields(logrus.Fields{
		"cert":      certPath,
		"key":       keyPath,
		"valid_for": config.ValidFor,
		"cn":        config.CommonName,
		"hosts":     config.Hosts,
	}).Info("✅ Self-signed TLS certificate generated successfully")
	
	return nil
}

// GetCertificatePaths returns full paths to certificate and key files
func (cm *CertificateManager) GetCertificatePaths(certFile, keyFile string) (string, string) {
	return filepath.Join(cm.certDir, certFile), filepath.Join(cm.certDir, keyFile)
}

// ValidateCertificate checks if certificate is valid and not expired
func (cm *CertificateManager) ValidateCertificate(certPath string) error {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return fmt.Errorf("failed to read certificate: %w", err)
	}
	
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return fmt.Errorf("failed to parse certificate PEM")
	}
	
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}
	
	// Check expiration
	now := time.Now()
	if now.Before(cert.NotBefore) {
		return fmt.Errorf("certificate not yet valid (valid from %v)", cert.NotBefore)
	}
	if now.After(cert.NotAfter) {
		return fmt.Errorf("certificate expired (valid until %v)", cert.NotAfter)
	}
	
	cm.logger.WithFields(logrus.Fields{
		"subject":     cert.Subject.CommonName,
		"issuer":      cert.Issuer.CommonName,
		"not_before":  cert.NotBefore,
		"not_after":   cert.NotAfter,
		"dns_names":   cert.DNSNames,
	}).Info("Certificate is valid")
	
	return nil
}

// GetCertificateInfo returns information about the certificate
func (cm *CertificateManager) GetCertificateInfo(certPath string) (map[string]interface{}, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate: %w", err)
	}
	
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to parse certificate PEM")
	}
	
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}
	
	info := map[string]interface{}{
		"subject":         cert.Subject.CommonName,
		"organization":    cert.Subject.Organization,
		"issuer":          cert.Issuer.CommonName,
		"not_before":      cert.NotBefore,
		"not_after":       cert.NotAfter,
		"dns_names":       cert.DNSNames,
		"serial_number":   cert.SerialNumber.String(),
		"signature_algo":  cert.SignatureAlgorithm.String(),
		"is_self_signed":  cert.Subject.CommonName == cert.Issuer.CommonName,
		"expires_in_days": int(time.Until(cert.NotAfter).Hours() / 24),
	}
	
	return info, nil
}

