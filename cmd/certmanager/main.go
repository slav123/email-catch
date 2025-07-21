package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/slav123/email-catch/internal/config"
	tlsmanager "github.com/slav123/email-catch/internal/tls"
)

func main() {
	var (
		configFile = flag.String("config", "config/config.yaml", "Path to configuration file")
		action     = flag.String("action", "info", "Action to perform: info, renew, generate-self-signed")
		domains    = flag.String("domains", "", "Comma-separated domains for self-signed cert")
		certPath   = flag.String("cert", "certs/server.crt", "Certificate path for self-signed")
		keyPath    = flag.String("key", "certs/server.key", "Key path for self-signed")
	)
	flag.Parse()

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	switch *action {
	case "info":
		showCertificateInfo(cfg)
	case "renew":
		renewCertificates(cfg)
	case "generate-self-signed":
		generateSelfSigned(*domains, *certPath, *keyPath)
	default:
		fmt.Printf("Unknown action: %s\n", *action)
		fmt.Println("Available actions: info, renew, generate-self-signed")
		os.Exit(1)
	}
}

func showCertificateInfo(cfg *config.Config) {
	if !cfg.Server.TLS.LetsEncrypt.Enabled {
		fmt.Println("Let's Encrypt is not enabled in configuration")
		return
	}

	leMgr, err := tlsmanager.NewLetsEncryptManager(tlsmanager.LetsEncryptConfig{
		Enabled:         cfg.Server.TLS.LetsEncrypt.Enabled,
		Domains:         cfg.Server.TLS.LetsEncrypt.Domains,
		Email:           cfg.Server.TLS.LetsEncrypt.Email,
		CacheDir:        cfg.Server.TLS.LetsEncrypt.CacheDir,
		Staging:         cfg.Server.TLS.LetsEncrypt.Staging,
		HTTPPort:        cfg.Server.TLS.LetsEncrypt.HTTPPort,
		RenewBefore:     cfg.Server.TLS.LetsEncrypt.RenewBeforeDays,
	})
	if err != nil {
		log.Fatalf("Failed to create Let's Encrypt manager: %v", err)
	}

	info, err := leMgr.GetCertificateInfo()
	if err != nil {
		log.Fatalf("Failed to get certificate info: %v", err)
	}

	fmt.Println("=== Certificate Information ===")
	for domain, certInfo := range info {
		fmt.Printf("\nDomain: %s\n", domain)
		
		data, _ := json.MarshalIndent(certInfo, "  ", "  ")
		fmt.Printf("  %s\n", string(data))
	}
}

func renewCertificates(cfg *config.Config) {
	if !cfg.Server.TLS.LetsEncrypt.Enabled {
		fmt.Println("Let's Encrypt is not enabled in configuration")
		return
	}

	leMgr, err := tlsmanager.NewLetsEncryptManager(tlsmanager.LetsEncryptConfig{
		Enabled:         cfg.Server.TLS.LetsEncrypt.Enabled,
		Domains:         cfg.Server.TLS.LetsEncrypt.Domains,
		Email:           cfg.Server.TLS.LetsEncrypt.Email,
		CacheDir:        cfg.Server.TLS.LetsEncrypt.CacheDir,
		Staging:         cfg.Server.TLS.LetsEncrypt.Staging,
		HTTPPort:        cfg.Server.TLS.LetsEncrypt.HTTPPort,
		RenewBefore:     cfg.Server.TLS.LetsEncrypt.RenewBeforeDays,
	})
	if err != nil {
		log.Fatalf("Failed to create Let's Encrypt manager: %v", err)
	}

	fmt.Println("Starting HTTP challenge server...")
	if err := leMgr.StartHTTPChallengeServer(); err != nil {
		log.Printf("Failed to start HTTP challenge server: %v", err)
	}

	fmt.Println("Requesting certificates...")
	certPath, keyPath, err := leMgr.GetCertificatePaths()
	if err != nil {
		log.Fatalf("Failed to get certificates: %v", err)
	}

	fmt.Printf("Certificates saved to:\n")
	fmt.Printf("  Certificate: %s\n", certPath)
	fmt.Printf("  Private Key: %s\n", keyPath)
	fmt.Println("Certificate renewal completed successfully!")
}

func generateSelfSigned(domainsStr, certPath, keyPath string) {
	if domainsStr == "" {
		log.Fatal("Domains must be specified for self-signed certificate generation")
	}

	domains := strings.Split(domainsStr, ",")
	for i, domain := range domains {
		domains[i] = strings.TrimSpace(domain)
	}

	fmt.Printf("Generating self-signed certificate for domains: %v\n", domains)

	if err := os.MkdirAll(filepath.Dir(certPath), 0755); err != nil {
		log.Fatalf("Failed to create certificate directory: %v", err)
	}

	if err := tlsmanager.GenerateSelfSignedCert(domains, certPath, keyPath); err != nil {
		log.Fatalf("Failed to generate self-signed certificate: %v", err)
	}

	fmt.Printf("Self-signed certificate generated:\n")
	fmt.Printf("  Certificate: %s\n", certPath)
	fmt.Printf("  Private Key: %s\n", keyPath)
}