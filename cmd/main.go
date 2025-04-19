package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Panorama-Block/avax/internal/app"
	"github.com/Panorama-Block/avax/internal/config"
)

func main() {
	// Configure log format
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Initializing application...")
	
	// Load application configuration
	log.Println("Loading configuration...")
	cfg := config.LoadConfig()
	log.Printf("Configuration loaded: Webhook Port=%s, URL=%s", cfg.WebhookPort, cfg.WebhookURL)

	// Create and configure the application
	log.Println("Creating application...")
	application := app.NewApp(cfg)
	
	log.Println("Setting up services...")
	if err := application.SetupServices(); err != nil {
		log.Fatalf("Fatal error setting up services: %v", err)
	}

	// Start the application
	log.Println("Starting application...")
	if err := application.Start(); err != nil {
		log.Fatalf("Fatal error starting application: %v", err)
	}
	
	log.Println("===========================================")
	log.Println("Application running. Press Ctrl+C to exit.")
	log.Printf("Webhook listening on port: %s", cfg.WebhookPort)
	log.Printf("Webhook URL: %s", cfg.WebhookURL)
	log.Println("===========================================")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Interrupt signal received, shutting down application...")

	// Stop the application
	application.Stop()

	// Give time for graceful shutdown
	time.Sleep(2 * time.Second)
	log.Println("Application successfully shut down")
}
