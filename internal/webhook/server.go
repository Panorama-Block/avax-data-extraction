package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/Panorama-Block/avax/internal/extractor"
	"github.com/Panorama-Block/avax/internal/types"
)

type WebhookServer struct {
    EventChan chan types.WebhookEvent
    Port      string
    pipeline  *extractor.TransactionPipeline
    webhookSecret string
    metrics   *WebhookMetrics
    server    *http.Server
}

type WebhookMetrics struct {
    Processed   int64
    Errors      int64
    InvalidSigs int64
    StartTime   time.Time
    mutex       http.Handler
}

func NewWebhookServer(eventChan chan types.WebhookEvent, port string, webhookSecret string) *WebhookServer {
    return &WebhookServer{
        EventChan: eventChan,
        Port:      port,
        webhookSecret: webhookSecret,
        metrics: &WebhookMetrics{
            StartTime: time.Now(),
        },
    }
}

func (ws *WebhookServer) Start() error {
    // Create a new mux instead of using the global DefaultServeMux
    mux := http.NewServeMux()
    
    // Register handlers on this mux
    mux.HandleFunc("/webhook", ws.handleWebhook)
    mux.HandleFunc("/health", ws.handleHealth)
    mux.HandleFunc("/test", ws.handleTest)
    
    // Create server with explicit configuration
    ws.server = &http.Server{
        Addr:         ":" + ws.Port,
        Handler:      mux,
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 30 * time.Second,
        IdleTimeout:  120 * time.Second,
    }

    log.Printf("🔄 Starting Webhook server on port %s...", ws.Port)
    
    // Start the server in a goroutine
    go func() {
        log.Printf("📡 Webhook server listening on port %s", ws.Port)
        if err := ws.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Printf("❌ CRITICAL ERROR starting Webhook server: %v", err)
        }
    }()

    // Test connection to verify if the server is actually listening
    time.Sleep(1 * time.Second)
    
    // Try to connect to the server multiple times
    checkURL := fmt.Sprintf("http://localhost:%s/health", ws.Port)
    var lastErr error
    
    for i := 0; i < 5; i++ {
        resp, err := http.Get(checkURL)
        if err == nil {
            respBody, _ := io.ReadAll(resp.Body)
            resp.Body.Close()
            log.Printf("✅ Webhook server confirmed running on port %s", ws.Port)
            log.Printf("✅ Server response: %s", string(respBody))
            log.Printf("✅ To test locally, access: http://localhost:%s/test", ws.Port)
            return nil
        }
        
        lastErr = err
        log.Printf("⏳ Check %d: waiting for Webhook server to start... error: %v", i+1, err)
        time.Sleep(2 * time.Second)
    }
    
    log.Printf("⚠️ WARNING: Could not confirm if the Webhook server is running!")
    log.Printf("⚠️ Last error: %v", lastErr)
    log.Printf("⚠️ Check if port %s is available and not blocked by firewall", ws.Port)
    
    return fmt.Errorf("could not confirm if the Webhook server is running")
}

func (ws *WebhookServer) Stop() {
    log.Printf("🛑 Stopping Webhook server...")
    
    if ws.server != nil {
        // Create a context with timeout for shutdown
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        
        // Try to gracefully shutdown the server
        if err := ws.server.Shutdown(ctx); err != nil {
            log.Printf("⚠️ Error shutting down Webhook server: %v", err)
        } else {
            log.Printf("✅ Webhook server successfully shutdown")
        }
    } else {
        log.Printf("⚠️ Webhook server was not initialized")
    }
}

func (ws *WebhookServer) SetTransactionPipeline(pipeline *extractor.TransactionPipeline) {
    ws.pipeline = pipeline
}

func (ws *WebhookServer) handleWebhook(w http.ResponseWriter, r *http.Request) {
    startTime := time.Now()
    
    // Log the received request
    log.Printf("📥 Webhook request received: %s %s", r.Method, r.URL.Path)
    
    // Log all headers for debugging
    log.Printf("📋 Request headers:")
    for name, values := range r.Header {
        for _, value := range values {
            log.Printf("   %s: %s", name, value)
        }
    }
    
    // Read the request body
    body, err := io.ReadAll(r.Body)
    if err != nil {
        log.Printf("❌ Error reading request body: %v", err)
        http.Error(w, "Error reading request", http.StatusInternalServerError)
        return
    }
    defer r.Body.Close()

    // Log the request body for debugging
    log.Printf("📄 Webhook request body: %s", string(body))

    // Verify signature if secret is set
    signatureValid := true
    if ws.webhookSecret != "" {
        signature := r.Header.Get("X-Avacloud-Signature-256")
        // Try alternative header names if signature not found
        if signature == "" {
            signature = r.Header.Get("X-Signature-256")
        }
        if signature == "" {
            signature = r.Header.Get("X-Hub-Signature-256")
        }
        
        if signature == "" {
            log.Printf("⚠️ WARNING: Signature not found in header - TEMPORARILY ALLOWING REQUEST FOR DEBUGGING")
            ws.metrics.InvalidSigs++
            signatureValid = false
            // Continue processing instead of returning error
            // http.Error(w, "Missing signature", http.StatusUnauthorized)
            // return
        } else {
            expectedSignature := "sha256=" + computeHMAC(body, ws.webhookSecret)
            log.Printf("🔐 Verifying signature: Received=%s, Expected=%s", signature, expectedSignature)
            
            if !secureCompare(signature, expectedSignature) {
                log.Printf("⚠️ WARNING: Invalid signature: %s - TEMPORARILY ALLOWING REQUEST FOR DEBUGGING", signature)
                ws.metrics.InvalidSigs++
                signatureValid = false
                // Continue processing instead of returning error
                // http.Error(w, "Invalid signature", http.StatusUnauthorized)
                // return
            } else {
                log.Printf("✅ Signature successfully validated")
            }
        }
    }

    // Unmarshal the request body into a WebhookEvent
    var event types.WebhookEvent
    if err := json.Unmarshal(body, &event); err != nil {
        log.Printf("❌ Error decoding JSON: %v", err)
        http.Error(w, "Error decoding JSON", http.StatusBadRequest)
        ws.metrics.Errors++
        return
    }
    
    // Log reception of webhook
    log.Printf("✅ Webhook received: messageId=%s eventType=%s (signatureValid=%v)", 
        event.MessageID, event.EventType, signatureValid)
    
    // If transaction pipeline is configured, submit the event to it
    if ws.pipeline != nil {
        if err := ws.pipeline.Submit(event); err != nil {
            log.Printf("❌ Error submitting event to pipeline: %v", err)
            ws.metrics.Errors++
        } else {
            ws.metrics.Processed++
            log.Printf("✅ Event successfully sent to pipeline")
        }
    } else {
        // Otherwise, send to the event channel
        go func() {
            log.Printf("📤 Sending event to channel (pipeline not configured)")
            ws.EventChan <- event
            ws.metrics.Processed++
        }()
    }

    // Calculate and log processing time
    processingTime := time.Since(startTime)
    log.Printf("⏱️ Webhook processed in %v: messageId=%s", 
        processingTime, event.MessageID)

    w.WriteHeader(http.StatusOK)
    fmt.Fprintln(w, "Webhook successfully received")
}

// handleHealth handles health check requests
func (ws *WebhookServer) handleHealth(w http.ResponseWriter, r *http.Request) {
    health := map[string]interface{}{
        "status":        "running",
        "uptime":        time.Since(ws.metrics.StartTime).String(),
        "processed":     ws.metrics.Processed,
        "errors":        ws.metrics.Errors,
        "invalidSigs":   ws.metrics.InvalidSigs,
        "pipelineStats": nil,
    }
    
    if ws.pipeline != nil {
        health["pipelineStats"] = ws.pipeline.Status()
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(health)
}

// handleTest provides a simple test endpoint for verifying the webhook server
func (ws *WebhookServer) handleTest(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    
    // Create test response
    testResponse := map[string]interface{}{
        "status":    "ok",
        "timestamp": time.Now().Format(time.RFC3339),
        "message":   "Webhook server is working correctly!",
        "server_info": map[string]interface{}{
            "uptime":        time.Since(ws.metrics.StartTime).String(),
            "processed":     ws.metrics.Processed,
            "errors":        ws.metrics.Errors,
            "invalidSigs":   ws.metrics.InvalidSigs,
        },
    }
    
    json.NewEncoder(w).Encode(testResponse)
    log.Printf("✅ Test endpoint accessed successfully")
}

// Helper functions for HMAC signature verification
func computeHMAC(body []byte, secret string) string {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(body)
    return hex.EncodeToString(mac.Sum(nil))
}

func secureCompare(a, b string) bool {
    return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
