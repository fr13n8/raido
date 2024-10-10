package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// CertManager defines the interface for managing TLS configuration
type CertManager interface {
	GetTLSConfig() (*tls.Config, error)
}

// SelfSignedCertManager handles self-signed certificate generation
type SelfSignedCertManager struct {
	Host     string
	CertDir  string
	CertPath string
	KeyPath  string
	certDER  []byte // Store the generated cert in DER format
}

// NewSelfSignedCertManager creates a new manager for self-signed certificates
func NewSelfSignedCertManager(host, certDir string) *SelfSignedCertManager {
	certFileName := fmt.Sprintf("%s_cert.pem", host)
	keyFileName := fmt.Sprintf("%s_key.pem", host)

	return &SelfSignedCertManager{
		Host:     host,
		CertDir:  certDir,
		CertPath: filepath.Join(certDir, certFileName),
		KeyPath:  filepath.Join(certDir, keyFileName),
	}
}

func (cm *SelfSignedCertManager) GetCertHash() ([]byte, error) {
	// Check if cert exists in DER format; if not, generate the cert
	if cm.certDER == nil {
		if !certExists(cm.CertPath, cm.KeyPath) {
			_, err := cm.generateSelfSignedCert()
			if err != nil {
				return nil, err
			}
		} else {
			// If cert exists, read the cert file and extract its fingerprint
			certPEM, err := os.ReadFile(cm.CertPath)
			if err != nil {
				return nil, err
			}
			block, _ := pem.Decode(certPEM)
			if block == nil {
				return nil, fmt.Errorf("failed to decode PEM block containing certificate")
			}
			cm.certDER = block.Bytes
		}
	}

	// Compute and return the SHA-256 hash of the certificate
	fingerprint := sha256.Sum256(cm.certDER)
	return fingerprint[:], nil
}

// GetTLSConfig generates or loads a self-signed certificate and returns a tls.Config
func (cm *SelfSignedCertManager) GetTLSConfig() (*tls.Config, error) {
	cert, err := cm.GetCertificate()
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{*cert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}

// GetCertificate loads or generates a self-signed certificate
func (cm *SelfSignedCertManager) GetCertificate() (*tls.Certificate, error) {
	if certExists(cm.CertPath, cm.KeyPath) {
		return loadCertificate(cm.CertPath, cm.KeyPath)
	}
	return cm.generateSelfSignedCert()
}

// generateSelfSignedCert generates and saves a self-signed certificate
func (cm *SelfSignedCertManager) generateSelfSignedCert() (*tls.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour) // 1-year validity

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}

	var ips []net.IP
	for _, ip := range []string{"127.0.0.1", "::1"} {
		parsed := net.ParseIP(ip)
		if parsed == nil {
			return nil, err
		}
		ips = append(ips, parsed)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: cm.Host,
		},
		DNSNames:    []string{cm.Host},
		IPAddresses: ips,
		NotBefore:   notBefore,
		NotAfter:    notAfter,
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
		BasicConstraintsValid: true,
	}

	cm.certDER, err = x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, err
	}

	// Ensure the certificate directory exists
	os.MkdirAll(cm.CertDir, 0755)

	certOut, err := os.Create(cm.CertPath)
	if err != nil {
		return nil, err
	}
	defer certOut.Close()
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: cm.certDER})

	keyOut, err := os.Create(cm.KeyPath)
	if err != nil {
		return nil, err
	}
	defer keyOut.Close()
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	return loadCertificate(cm.CertPath, cm.KeyPath)
}

// Helper functions
func certExists(certPath, keyPath string) bool {
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return false
	}
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return false
	}
	return true
}

func loadCertificate(certPath, keyPath string) (*tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}
	return &cert, nil
}
