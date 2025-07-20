package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/slav123/email-catch/internal/config"
	"github.com/slav123/email-catch/internal/smtp"
	"github.com/slav123/email-catch/internal/storage"
	"github.com/slav123/email-catch/internal/webhook"
	"github.com/slav123/email-catch/pkg/email"
)

func main() {
	var configFile = flag.String("config", "config/config.yaml", "Path to configuration file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	storageBackend, err := storage.NewStorageBackend(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize storage backend: %v", err)
	}

	webhookClient := webhook.NewClient()

	processor := email.NewProcessor(cfg, storageBackend, webhookClient)

	server := smtp.NewServer(cfg, processor)

	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start SMTP server: %v", err)
	}

	log.Println("Email catch server started successfully")
	log.Printf("Listening on ports: %v", cfg.Server.Ports)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("Shutting down server...")

	server.Stop()
	log.Println("Server stopped")
}