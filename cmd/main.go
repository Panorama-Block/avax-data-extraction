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
	// Load application configuration
	cfg := config.LoadConfig()

	// Create and configure the application
	application := app.NewApp(cfg)
	if err := application.SetupServices(); err != nil {
		log.Fatalf("Error setting up services: %v", err)
	}

	// Start the application
	if err := application.Start(); err != nil {
		log.Fatalf("Error starting application: %v", err)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Interrupt signal received, shutting down...")

	// Stop the application
	application.Stop()

	// Give time for graceful shutdown
	time.Sleep(2 * time.Second)
	log.Println("Application shut down successfully")
}
