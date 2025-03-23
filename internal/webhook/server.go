package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
	"bytes"
	"github.com/Panorama-Block/avax/internal/extractor"
	"github.com/Panorama-Block/avax/internal/api"
)

// Server represents a webhook server for receiving AvaCloud events
type Server struct {
	transactionPipeline *extractor.TransactionPipeline
	sharedSecret        string
	port                int
	server              *http.Server
	
	// Stats
	mutex               sync.Mutex
	eventsReceived      int64
	eventsProcessed     int64
	eventsRejected      int64
	startTime           time.Time
	running             bool
}

// NewServer creates a new webhook server
func NewServer(
	transactionPipeline *extractor.TransactionPipeline,
	sharedSecret string,
	port int,
) *Server {
	return &Server{
		transactionPipeline: transactionPipeline,
		sharedSecret:        sharedSecret,
		port:                port,
	}
}

// Start begins the webhook server
func (s *Server) Start() error {
	s.mutex.Lock()
	if s.running {
		s.mutex.Unlock()
		return fmt.Errorf("webhook server already running")
	}
	s.running = true
	s.startTime = time.Now()
	s.eventsReceived = 0
	s.eventsProcessed = 0
	s.eventsRejected = 0
	s.mutex.Unlock()
	
	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", s.handleWebhook)
	mux.HandleFunc("/health", s.handleHealth)
	
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}
	
	log.Printf("Starting webhook server on port %d", s.port)
	
	// Start the server in a goroutine
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Webhook server error: %v", err)
		}
	}()
	
	return nil
}

// Stop halts the webhook server
func (s *Server) Stop(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if !s.running {
		return nil
	}
	
	// Shutdown the server
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("error shutting down webhook server: %w", err)
	}
	
	s.running = false
	log.Printf("Webhook server stopped")
	
	return nil
}

// handleWebhook processes incoming webhook requests
func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	// Update received count
	s.mutex.Lock()
	s.eventsReceived++
	s.mutex.Unlock()
	
	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		s.incrementRejected()
		return
	}
	
	// Verify HMAC signature if shared secret is configured
	if s.sharedSecret != "" {
		signature := r.Header.Get("X-Signature")
		if signature == "" {
			http.Error(w, "Missing signature", http.StatusBadRequest)
			s.incrementRejected()
			return
		}
		
		// Read the request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			s.incrementRejected()
			return
		}
		
		// Verify the signature
		valid, err := api.VerifyWebhookSignature(body, signature, s.sharedSecret)
		if err != nil {
			log.Printf("Error verifying signature: %v", err)
			http.Error(w, "Error verifying signature", http.StatusInternalServerError)
			s.incrementRejected()
			return
		}
		
		if !valid {
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			s.incrementRejected()
			return
		}
		
		// Replace the body for further reading
		r.Body = io.NopCloser(bytes.NewReader(body))
	}
	
	// Parse the webhook event
	var event extractor.WebhookEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "Error parsing webhook event", http.StatusBadRequest)
		s.incrementRejected()
		return
	}
	
	// Log receipt of the event
	log.Printf("Received webhook event: %s (type: %s)", event.MessageID, event.EventType)
	
	// Submit the event to the transaction pipeline
	if err := s.transactionPipeline.Submit(event); err != nil {
		log.Printf("Error submitting event to transaction pipeline: %v", err)
		http.Error(w, "Error processing webhook event", http.StatusInternalServerError)
		s.incrementRejected()
		return
	}
	
	// Update processed count
	s.mutex.Lock()
	s.eventsProcessed++
	s.mutex.Unlock()
	
	// Respond with 200 OK
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleHealth provides a health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Return simple health status
	status := map[string]interface{}{
		"status":           "UP",
		"eventsReceived":   s.eventsReceived,
		"eventsProcessed":  s.eventsProcessed,
		"eventsRejected":   s.eventsRejected,
		"uptime":           time.Since(s.startTime).String(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// incrementRejected increments the rejected events counter
func (s *Server) incrementRejected() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.eventsRejected++
}

// Status returns the current status of the webhook server
func (s *Server) Status() map[string]interface{} {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	uptime := time.Since(s.startTime).String()
	if !s.running {
		uptime = "0s"
	}
	
	return map[string]interface{}{
		"running":          s.running,
		"port":             s.port,
		"eventsReceived":   s.eventsReceived,
		"eventsProcessed":  s.eventsProcessed,
		"eventsRejected":   s.eventsRejected,
		"uptime":           uptime,
		"startTime":        s.startTime.Format(time.RFC3339),
	}
}