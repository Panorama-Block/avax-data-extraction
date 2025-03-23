package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
	"strings"
	
	"github.com/gorilla/websocket"
	
	"github.com/Panorama-Block/avax/internal/kafka"
	
)

// Server represents a WebSocket server for streaming data to the frontend
type Server struct {
	port           int
	upgrader       websocket.Upgrader
	clientManager  *ClientManager
	kafkaConsumer  *kafka.Consumer
	server         *http.Server
	
	// Stats
	mutex          sync.Mutex
	running        bool
	startTime      time.Time
	connectionsTotal int
	messagesTotal  int
}

// NewServer creates a new WebSocket server
func NewServer(port int, kafkaConsumer *kafka.Consumer) *Server {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for now, restrict in production
		},
	}
	
	return &Server{
		port:          port,
		upgrader:      upgrader,
		clientManager: NewClientManager(),
		kafkaConsumer: kafkaConsumer,
	}
}

// Start begins the WebSocket server
func (s *Server) Start() error {
	s.mutex.Lock()
	if s.running {
		s.mutex.Unlock()
		return fmt.Errorf("WebSocket server already running")
	}
	s.running = true
	s.startTime = time.Now()
	s.connectionsTotal = 0
	s.messagesTotal = 0
	s.mutex.Unlock()
	
	// Start the client manager
	go s.clientManager.Run()
	
	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/health", s.handleHealth)
	
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}
	
	log.Printf("Starting WebSocket server on port %d", s.port)
	
	// Start Kafka consumer for blocks and transactions
	go s.consumeKafkaMessages()
	
	// Start the server in a goroutine
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("WebSocket server error: %v", err)
		}
	}()
	
	return nil
}

// Stop halts the WebSocket server
func (s *Server) Stop(ctx context.Context) error {
	s.mutex.Lock()
	if !s.running {
		s.mutex.Unlock()
		return nil
	}
	s.running = false
	s.mutex.Unlock()
	
	// Close all client connections
	s.clientManager.CloseAll()
	
	// Shutdown the server
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("error shutting down WebSocket server: %w", err)
	}
	
	log.Printf("WebSocket server stopped")
	
	return nil
}

// handleWebSocket handles a WebSocket connection
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract subscription parameters from the request
	topicParam := r.URL.Query().Get("topic")
	chainID := r.URL.Query().Get("chainId")
	
	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	
	// Create a new client
	client := NewClient(conn)
	client.Topic = topicParam
	client.ChainID = chainID
	
	// Register the client with the client manager
	s.clientManager.Register(client)
	
	// Increment connections count
	s.mutex.Lock()
	s.connectionsTotal++
	s.mutex.Unlock()
	
	log.Printf("New WebSocket connection: %s (topic: %s, chainId: %s)", conn.RemoteAddr(), topicParam, chainID)
	
	// Handle client messages (subscriptions, etc.)
	go client.ReadPump(s.handleClientMessage)
	
	// Send messages to the client
	go client.WritePump()
}

// handleClientMessage handles a message from a client
func (s *Server) handleClientMessage(client *Client, message []byte) {
	// Parse the message
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Error parsing client message: %v", err)
		return
	}
	
	// Check for subscription message
	if action, ok := msg["action"].(string); ok && action == "subscribe" {
		topic, ok1 := msg["topic"].(string)
		chainID, ok2 := msg["chainId"].(string)
		
		if ok1 && ok2 {
			// Update client subscription
			client.Topic = topic
			client.ChainID = chainID
			
			log.Printf("Client %s subscribed to topic: %s, chainId: %s", client.ID, topic, chainID)
			
			// Send acknowledgment
			ack := map[string]interface{}{
				"type": "subscription",
				"status": "success",
				"topic": topic,
				"chainId": chainID,
			}
			
			client.Send(ack)
		}
	}
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	uptime := time.Since(s.startTime).String()
	if !s.running {
		uptime = "0s"
	}
	
	// Return health status as JSON
	status := map[string]interface{}{
		"status": "UP",
		"connections": s.clientManager.Count(),
		"connectionsTotal": s.connectionsTotal,
		"messagesTotal": s.messagesTotal,
		"uptime": uptime,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// consumeKafkaMessages consumes messages from Kafka and broadcasts them to clients
func (s *Server) consumeKafkaMessages() {
	// Subscribe to block and transaction topics
	topics := []string{
		"avalanche.blocks.*",    // All block topics
		"avalanche.transactions.*", // All transaction topics
		"avalanche.whalealerts.*",  // All whale alert topics
	}
	
	if err := s.kafkaConsumer.Subscribe(topics); err != nil {
		log.Printf("Error subscribing to Kafka topics: %v", err)
		return
	}
	
	log.Printf("Subscribed to Kafka topics: %v", topics)
	
	for s.running {
		// Poll for messages
		msg, err := s.kafkaConsumer.Poll(100) // 100ms timeout
		if err != nil {
			if s.running { // Only log if still running
				log.Printf("Error polling Kafka: %v", err)
			}
			continue
		}
		
		// Skip if no message
		if msg == nil {
			continue
		}
		
		// Process the message
		s.processKafkaMessage(msg)
	}
}

// processKafkaMessage processes a message from Kafka
func (s *Server) processKafkaMessage(msg *kafka.Message) {
	// Parse the topic to extract chainId and data type
	topic := msg.Topic
	
	// Example topic format: avalanche.blocks.43114
	// We want to extract "blocks" and "43114"
	parts := strings.Split(topic, ".")
	if len(parts) < 3 {
		log.Printf("Invalid topic format: %s", topic)
		return
	}
	
	dataType := parts[1] // blocks, transactions, whalealerts, etc.
	chainID := parts[2]  // 43114, etc.
	
	// Parse the message value as JSON
	var data map[string]interface{}
	if err := json.Unmarshal(msg.Value, &data); err != nil {
		log.Printf("Error parsing Kafka message: %v", err)
		return
	}
	
	// Add metadata
	data["type"] = dataType
	data["chainId"] = chainID
	data["receivedAt"] = time.Now().Unix()
	
	// Broadcast to clients
	s.broadcastToClients(dataType, chainID, data)
	
	// Update stats
	s.mutex.Lock()
	s.messagesTotal++
	s.mutex.Unlock()
}

// broadcastToClients broadcasts a message to all clients subscribed to the topic
func (s *Server) broadcastToClients(dataType, chainID string, data map[string]interface{}) {
	// Create broadcast message
	message := map[string]interface{}{
		"type": dataType,
		"chainId": chainID,
		"data": data,
		"timestamp": time.Now().Unix(),
	}
	
	// Broadcast to all clients with matching subscriptions
	s.clientManager.Broadcast(dataType, chainID, message)
}

// Status returns the current status of the WebSocket server
func (s *Server) Status() map[string]interface{} {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	uptime := time.Since(s.startTime).String()
	if !s.running {
		uptime = "0s"
	}
	
	return map[string]interface{}{
		"running": s.running,
		"port": s.port,
		"connections": s.clientManager.Count(),
		"connectionsTotal": s.connectionsTotal,
		"messagesTotal": s.messagesTotal,
		"uptime": uptime,
		"startTime": s.startTime.Format(time.RFC3339),
	}
}

// BroadcastMessage represents a message to be broadcast to clients
type BroadcastMessage struct {
	Topic   string
	ChainID string
	Data    map[string]interface{}
}

// NewClientManager creates a new client manager
func NewClientManager() *ClientManager {
	return &ClientManager{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *BroadcastMessage),
	}
}

// Register registers a client
func (m *ClientManager) Register(client *Client) {
	m.register <- client
}

// Unregister unregisters a client
func (m *ClientManager) Unregister(client *Client) {
	m.unregister <- client
}

// Broadcast broadcasts a message to all clients with matching subscriptions
func (m *ClientManager) Broadcast(topic, chainID string, data map[string]interface{}) {
	m.broadcast <- &BroadcastMessage{
		Topic:   topic,
		ChainID: chainID,
		Data:    data,
	}
}

// Count returns the number of connected clients
func (m *ClientManager) Count() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return len(m.clients)
}

// CloseAll closes all client connections
func (m *ClientManager) CloseAll() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	for client := range m.clients {
		client.Conn.Close()
		delete(m.clients, client)
	}
}

// Client represents a WebSocket client
type Client struct {
	ID        string
	Conn      *websocket.Conn
	SendChan  chan map[string]interface{}
	Topic     string // Subscription topic (blocks, transactions, etc.)
	ChainID   string // Chain ID filter (43114, etc.)
}

// NewClient creates a new WebSocket client
func NewClient(conn *websocket.Conn) *Client {
	return &Client{
		ID:       fmt.Sprintf("%d", time.Now().UnixNano()),
		Conn:     conn,
		SendChan: make(chan map[string]interface{}, 256),
		Topic:    "",
		ChainID:  "",
	}
}

// ReadPump reads messages from the WebSocket connection
func (c *Client) ReadPump(messageHandler func(*Client, []byte)) {
	defer func() {
		c.Conn.Close()
	}()
	
	c.Conn.SetReadLimit(4096) // Maximum message size allowed
	
	// Configure WebSocket parameters
	_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}
		
		messageHandler(c, message)
	}
}

// WritePump writes messages to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	
	for {
		select {
		case message, ok := <-c.SendChan:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Channel was closed
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			
			// Write JSON message
			if err := c.Conn.WriteJSON(message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}
		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Send sends a message to the client
func (c *Client) Send(message map[string]interface{}) {
	select {
	case c.SendChan <- message:
	default:
		// Channel is full, client might be slow
		log.Printf("Client %s send channel full, dropping message", c.ID)
	}
}

// ClientManager manages WebSocket clients
type ClientManager struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan *BroadcastMessage
	mutex      sync.Mutex
}

// Run starts the client manager
func (m *ClientManager) Run() {
	for {
		select {
		case client := <-m.register:
			m.mutex.Lock()
			m.clients[client] = true
			m.mutex.Unlock()
		case client := <-m.unregister:
			m.mutex.Lock()
			if _, ok := m.clients[client]; ok {
				delete(m.clients, client)
				close(client.SendChan)
			}
			m.mutex.Unlock()
		case message := <-m.broadcast:
			m.mutex.Lock()
			for client := range m.clients {
				// Send to clients that match the topic and chainID or have wildcards
				if (client.Topic == message.Topic || client.Topic == "*" || client.Topic == "") &&
					(client.ChainID == message.ChainID || client.ChainID == "*" || client.ChainID == "") {
					select {
					case client.SendChan <- message.Data:
					default:
						close(client.SendChan)
						delete(m.clients, client)
					}
				}
			}
			m.mutex.Unlock()
		}
	}
}

