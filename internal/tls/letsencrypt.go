package tls

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

type LetsEncryptManager struct {
	domains    []string
	email      string
	cacheDir   string
	staging    bool
	manager    *autocert.Manager
	httpPort   int
	client     *acme.Client
}

type LetsEncryptConfig struct {
	Enabled      bool     `yaml:"enabled"`
	Domains      []string `yaml:"domains"`
	Email        string   `yaml:"email"`
	CacheDir     string   `yaml:"cache_dir"`
	Staging      bool     `yaml:"staging"`
	HTTPPort     int      `yaml:"http_port"`
	RenewBefore  int      `yaml:"renew_before_days"`
}

func NewLetsEncryptManager(config LetsEncryptConfig) (*LetsEncryptManager, error) {
	if !config.Enabled {
		return nil, fmt.Errorf("Let's Encrypt is not enabled")
	}

	if len(config.Domains) == 0 {
		return nil, fmt.Errorf("no domains specified for Let's Encrypt")
	}

	if config.Email == "" {
		return nil, fmt.Errorf("email is required for Let's Encrypt")
	}

	if config.CacheDir == "" {
		config.CacheDir = "./certs/letsencrypt"
	}

	if config.HTTPPort == 0 {
		config.HTTPPort = 80
	}

	if config.RenewBefore == 0 {
		config.RenewBefore = 30
	}

	if err := os.MkdirAll(config.CacheDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	manager := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		Cache:      autocert.DirCache(config.CacheDir),
		HostPolicy: autocert.HostWhitelist(config.Domains...),
		Email:      config.Email,
	}

	if config.Staging {
		manager.Client = &acme.Client{
			DirectoryURL: "https://acme-staging-v02.api.letsencrypt.org/directory",
		}
	}

	lm := &LetsEncryptManager{
		domains:  config.Domains,
		email:    config.Email,
		cacheDir: config.CacheDir,
		staging:  config.Staging,
		manager:  manager,
		httpPort: config.HTTPPort,
	}

	return lm, nil
}

func (lm *LetsEncryptManager) GetCertificate() (*tls.Certificate, error) {
	cert, err := lm.manager.GetCertificate(&tls.ClientHelloInfo{
		ServerName: lm.domains[0],
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate: %w", err)
	}

	return cert, nil
}

func (lm *LetsEncryptManager) GetTLSConfig() *tls.Config {
	return &tls.Config{
		GetCertificate: lm.manager.GetCertificate,
		NextProtos:     []string{"h2", "http/1.1"},
	}
}

func (lm *LetsEncryptManager) StartHTTPChallengeServer() error {
	log.Printf("Starting HTTP challenge server on port %d", lm.httpPort)
	
	server := lm.manager.HTTPHandler(nil)
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", lm.httpPort), server); err != nil {
			log.Printf("HTTP challenge server error: %v", err)
		}
	}()

	return nil
}

func (lm *LetsEncryptManager) GetCertificatePaths() (string, string, error) {
	certPath := filepath.Join(lm.cacheDir, lm.domains[0]+".crt")
	keyPath := filepath.Join(lm.cacheDir, lm.domains[0]+".key")

	cert, err := lm.manager.GetCertificate(&tls.ClientHelloInfo{
		ServerName: lm.domains[0],
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to get certificate: %w", err)
	}

	certPEM := ""
	for _, certDER := range cert.Certificate {
		block := &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certDER,
		}
		certPEM += string(pem.EncodeToMemory(block))
	}

	if err := os.WriteFile(certPath, []byte(certPEM), 0644); err != nil {
		return "", "", fmt.Errorf("failed to write certificate file: %w", err)
	}

	keyDER, err := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal private key: %w", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyDER,
	})

	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return "", "", fmt.Errorf("failed to write private key file: %w", err)
	}

	return certPath, keyPath, nil
}

func (lm *LetsEncryptManager) StartAutoRenewal(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	log.Println("Starting certificate auto-renewal service")

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping certificate auto-renewal service")
			return
		case <-ticker.C:
			lm.checkAndRenewCertificates()
		}
	}
}

func (lm *LetsEncryptManager) checkAndRenewCertificates() {
	log.Println("Checking certificate expiration...")

	for _, domain := range lm.domains {
		if err := lm.checkCertificateExpiration(domain); err != nil {
			log.Printf("Certificate check/renewal failed for %s: %v", domain, err)
			continue
		}
	}
}

func (lm *LetsEncryptManager) checkCertificateExpiration(domain string) error {
	cert, err := lm.manager.GetCertificate(&tls.ClientHelloInfo{
		ServerName: domain,
	})
	if err != nil {
		return fmt.Errorf("failed to get certificate for %s: %w", domain, err)
	}

	if len(cert.Certificate) == 0 {
		return fmt.Errorf("no certificates found for %s", domain)
	}

	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return fmt.Errorf("failed to parse certificate for %s: %w", domain, err)
	}

	daysUntilExpiry := time.Until(x509Cert.NotAfter).Hours() / 24
	log.Printf("Certificate for %s expires in %.1f days", domain, daysUntilExpiry)

	if daysUntilExpiry <= 30 {
		log.Printf("Certificate for %s needs renewal (expires in %.1f days)", domain, daysUntilExpiry)
		
		_, err := lm.manager.GetCertificate(&tls.ClientHelloInfo{
			ServerName: domain,
		})
		if err != nil {
			return fmt.Errorf("failed to renew certificate for %s: %w", domain, err)
		}

		log.Printf("Successfully renewed certificate for %s", domain)
	}

	return nil
}

func (lm *LetsEncryptManager) ValidateDomains() error {
	for _, domain := range lm.domains {
		if domain == "" {
			return fmt.Errorf("empty domain name")
		}
		
		if domain == "localhost" || domain == "127.0.0.1" {
			return fmt.Errorf("cannot use Let's Encrypt with localhost or IP addresses")
		}
	}
	return nil
}

func (lm *LetsEncryptManager) GetCertificateInfo() (map[string]interface{}, error) {
	info := make(map[string]interface{})
	
	for _, domain := range lm.domains {
		cert, err := lm.manager.GetCertificate(&tls.ClientHelloInfo{
			ServerName: domain,
		})
		if err != nil {
			info[domain] = map[string]interface{}{
				"error": err.Error(),
			}
			continue
		}

		if len(cert.Certificate) == 0 {
			info[domain] = map[string]interface{}{
				"error": "no certificate data",
			}
			continue
		}

		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			info[domain] = map[string]interface{}{
				"error": fmt.Sprintf("failed to parse certificate: %v", err),
			}
			continue
		}

		info[domain] = map[string]interface{}{
			"subject":     x509Cert.Subject.CommonName,
			"issuer":      x509Cert.Issuer.CommonName,
			"not_before":  x509Cert.NotBefore,
			"not_after":   x509Cert.NotAfter,
			"dns_names":   x509Cert.DNSNames,
			"days_until_expiry": time.Until(x509Cert.NotAfter).Hours() / 24,
		}
	}

	return info, nil
}

func GenerateSelfSignedCert(domains []string, certPath, keyPath string) error {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Email Catch"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	for _, domain := range domains {
		if ip := net.ParseIP(domain); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, domain)
		}
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	certOut, err := os.Create(certPath)
	if err != nil {
		return fmt.Errorf("failed to open cert.pem for writing: %w", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}

	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open key.pem for writing: %w", err)
	}
	defer keyOut.Close()

	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	return nil
}