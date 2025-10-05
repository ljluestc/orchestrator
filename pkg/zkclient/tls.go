package zkclient

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"time"
)

// tlsDialer implements a TLS dialer for Zookeeper connections
type tlsDialer struct {
	config *tls.Config
}

// Dial establishes a TLS connection
func (d *tlsDialer) Dial(network, address string, timeout time.Duration) (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout: timeout,
	}

	conn, err := tls.DialWithDialer(dialer, network, address, d.config)
	if err != nil {
		return nil, fmt.Errorf("TLS dial failed: %w", err)
	}

	return conn, nil
}

// DialContext establishes a TLS connection with context
func (d *tlsDialer) DialContext(network, address string) (net.Conn, error) {
	conn, err := tls.Dial(network, address, d.config)
	if err != nil {
		return nil, fmt.Errorf("TLS dial failed: %w", err)
	}

	return conn, nil
}

// buildTLSConfigFromConfig creates a TLS configuration from TLSConfig
func buildTLSConfigFromConfig(tlsCfg *TLSConfig) (*tls.Config, error) {
	if tlsCfg == nil {
		return nil, fmt.Errorf("TLS config is nil")
	}

	config := &tls.Config{
		InsecureSkipVerify: tlsCfg.InsecureSkipVerify,
		ServerName:         tlsCfg.ServerName,
		MinVersion:         tls.VersionTLS12,
	}

	// Load client certificate if provided
	if tlsCfg.CertFile != "" && tlsCfg.KeyFile != "" {
		cert, err := loadCertificate(tlsCfg.CertFile, tlsCfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		config.Certificates = []tls.Certificate{cert}
	}

	// Load CA certificate if provided
	if tlsCfg.CAFile != "" {
		certPool, err := loadCACertPool(tlsCfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load CA certificate: %w", err)
		}
		config.RootCAs = certPool
	}

	return config, nil
}

// loadCertificate loads a client certificate and key from files
func loadCertificate(certFile, keyFile string) (tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to load X509 key pair: %w", err)
	}
	return cert, nil
}

// loadCACertPool loads CA certificates from a file into a cert pool
func loadCACertPool(caFile string) (*x509.CertPool, error) {
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA file: %w", err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	return certPool, nil
}

// ValidateCertificate validates a TLS certificate
func ValidateCertificate(cert *x509.Certificate) error {
	if cert == nil {
		return fmt.Errorf("certificate is nil")
	}

	now := time.Now()
	if now.Before(cert.NotBefore) {
		return fmt.Errorf("certificate is not yet valid (valid from: %v)", cert.NotBefore)
	}

	if now.After(cert.NotAfter) {
		return fmt.Errorf("certificate has expired (expired on: %v)", cert.NotAfter)
	}

	return nil
}

// VerifyPeerCertificate is a custom verification function for peer certificates
func VerifyPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	if len(rawCerts) == 0 {
		return fmt.Errorf("no certificates provided")
	}

	// Parse the certificate
	cert, err := x509.ParseCertificate(rawCerts[0])
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Validate certificate timing
	if err := ValidateCertificate(cert); err != nil {
		return err
	}

	// Additional custom validation can be added here
	// For example, checking specific certificate fields or extensions

	return nil
}

// NewTLSDialer creates a new TLS dialer with the given configuration
func NewTLSDialer(tlsCfg *TLSConfig) (*tlsDialer, error) {
	config, err := buildTLSConfigFromConfig(tlsCfg)
	if err != nil {
		return nil, err
	}

	return &tlsDialer{
		config: config,
	}, nil
}

// GetTLSConfig returns the TLS configuration for a client
func (c *Client) GetTLSConfig() (*tls.Config, error) {
	if !c.config.TLSEnabled {
		return nil, fmt.Errorf("TLS is not enabled")
	}

	if c.config.TLSConfig == nil {
		return nil, fmt.Errorf("TLS config is not set")
	}

	return buildTLSConfigFromConfig(c.config.TLSConfig)
}

// IsTLSEnabled returns whether TLS is enabled for this client
func (c *Client) IsTLSEnabled() bool {
	return c.config.TLSEnabled
}
