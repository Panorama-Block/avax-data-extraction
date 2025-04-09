package websocket

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "strings"
    "sync"
    "time"

    "github.com/gorilla/websocket"

    "github.com/Panorama-Block/avax/internal/kafka"
)

type Server struct {
    port             string
    upgrader         websocket.Upgrader
    clientManager    *ClientManager
    kafkaConsumer    *kafka.Consumer
    server           *http.Server

    mutex            sync.Mutex
    running          bool
    startTime        time.Time
    connectionsTotal int
    messagesTotal    int
}

func NewServer(port string, kafkaConsumer *kafka.Consumer) *Server {
    upgrader := websocket.Upgrader{
        ReadBufferSize:  1024,
        WriteBufferSize: 1024,
        CheckOrigin: func(r *http.Request) bool {
            return true
        },
    }

    return &Server{
        port:          port,
        upgrader:      upgrader,
        clientManager: NewClientManager(),
        kafkaConsumer: kafkaConsumer,
    }
}

func (s *Server) Start() error {
    s.mutex.Lock()
    if s.running {
        s.mutex.Unlock()
        return fmt.Errorf("Servidor WebSocket já está em execução")
    }
    s.running = true
    s.startTime = time.Now()
    s.connectionsTotal = 0
    s.messagesTotal = 0
    s.mutex.Unlock()

    go s.clientManager.Run()

    mux := http.NewServeMux()
    mux.HandleFunc("/ws", s.handleWebSocket)
    mux.HandleFunc("/health", s.handleHealth)

    s.server = &http.Server{
        Addr:    fmt.Sprintf(":%s", s.port),
        Handler: mux,
    }

    log.Printf("Iniciando servidor WebSocket na porta %s", s.port)

    go s.consumeKafkaMessages()

    go func() {
        if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Printf("Erro no servidor WebSocket: %v", err)
        }
    }()

    return nil
}

func (s *Server) Stop(ctx context.Context) error {
    s.mutex.Lock()
    if !s.running {
        s.mutex.Unlock()
        return nil
    }
    s.running = false
    s.mutex.Unlock()

    s.clientManager.CloseAll()

    if err := s.server.Shutdown(ctx); err != nil {
        return fmt.Errorf("erro ao desligar servidor WebSocket: %w", err)
    }

    log.Printf("Servidor WebSocket parado")
    return nil
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
    topicParam := r.URL.Query().Get("topic")
    chainID := r.URL.Query().Get("chainId")

    conn, err := s.upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Printf("Falha ao atualizar conexão: %v", err)
        return
    }

    client := NewClient(conn)
    client.Topic = topicParam
    client.ChainID = chainID

    s.clientManager.Register(client)

    s.mutex.Lock()
    s.connectionsTotal++
    s.mutex.Unlock()

    log.Printf("Nova conexão WebSocket: %s (tópico: %s, chainId: %s)",
        conn.RemoteAddr(), topicParam, chainID)

    go client.ReadPump(s.handleClientMessage)

    go client.WritePump()
}

func (s *Server) handleClientMessage(client *Client, message []byte) {
    var msg map[string]interface{}
    if err := json.Unmarshal(message, &msg); err != nil {
        log.Printf("Erro ao analisar mensagem do cliente: %v", err)
        return
    }

    if action, ok := msg["action"].(string); ok && action == "subscribe" {
        topic, ok1 := msg["topic"].(string)
        chainID, ok2 := msg["chainId"].(string)

        if ok1 && ok2 {
            client.Topic = topic
            client.ChainID = chainID

            log.Printf("Cliente %s inscrito no tópico: %s, chainId: %s",
                client.ID, topic, chainID)

            ack := map[string]interface{}{
                "type":    "subscription",
                "status":  "success",
                "topic":   topic,
                "chainId": chainID,
            }

            client.Send(ack)
        }
    }
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    s.mutex.Lock()
    defer s.mutex.Unlock()

    uptime := time.Since(s.startTime).String()
    if !s.running {
        uptime = "0s"
    }

    status := map[string]interface{}{
        "status":           "UP",
        "connections":      s.clientManager.Count(),
        "connectionsTotal": s.connectionsTotal,
        "messagesTotal":    s.messagesTotal,
        "uptime":           uptime,
    }

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(status)
}


func (s *Server) consumeKafkaMessages() {
    topics := []string{
        "avalanche.blocks.*",     
        "avalanche.transactions.*", 
        "avalanche.whalealerts.*", 
    }

    if err := s.kafkaConsumer.Subscribe(topics); err != nil {
        log.Printf("Erro ao inscrever-se em tópicos Kafka: %v", err)
        return
    }

    log.Printf("Inscrito em tópicos Kafka: %v", topics)

    for s.running {
        msg, err := s.kafkaConsumer.Poll(100)
        if err != nil {
            if s.running { 
                log.Printf("Erro ao sondar Kafka: %v", err)
            }
            continue
        }
        if msg == nil {
            continue
        }
        s.processKafkaMessage(msg)
    }
}

func (s *Server) processKafkaMessage(msg *kafka.Message) {
    topic := msg.Topic
    parts := strings.Split(topic, ".")
    if len(parts) < 3 {
        log.Printf("Formato de tópico inválido: %s", topic)
        return
    }
    dataType := parts[1]
    chainID := parts[2]

    var data map[string]interface{}
    if err := json.Unmarshal(msg.Value, &data); err != nil {
        log.Printf("Erro ao analisar mensagem Kafka: %v", err)
        return
    }

    data["type"] = dataType
    data["chainId"] = chainID
    data["receivedAt"] = time.Now().Unix()

    s.broadcastToClients(dataType, chainID, data)

    s.mutex.Lock()
    s.messagesTotal++
    s.mutex.Unlock()
}

func (s *Server) broadcastToClients(dataType, chainID string, data map[string]interface{}) {
    message := map[string]interface{}{
        "type":      dataType,
        "chainId":   chainID,
        "data":      data,
        "timestamp": time.Now().Unix(),
    }
    s.clientManager.Broadcast(dataType, chainID, message)
}

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
        "connections":      s.clientManager.Count(),
        "connectionsTotal": s.connectionsTotal,
        "messagesTotal":    s.messagesTotal,
        "uptime":           uptime,
        "startTime":        s.startTime.Format(time.RFC3339),
    }
}

type ClientManager struct {
    clients    map[*Client]bool
    register   chan *Client
    unregister chan *Client
    broadcast  chan *BroadcastMessage
    mutex      sync.Mutex
}

type BroadcastMessage struct {
    Topic   string
    ChainID string
    Data    map[string]interface{}
}

func NewClientManager() *ClientManager {
    return &ClientManager{
        clients:    make(map[*Client]bool),
        register:   make(chan *Client),
        unregister: make(chan *Client),
        broadcast:  make(chan *BroadcastMessage),
    }
}

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
                close(client.SendCh) 
            }
            m.mutex.Unlock()

        case message := <-m.broadcast:
            m.mutex.Lock()
            for client := range m.clients {
                if (client.Topic == message.Topic || client.Topic == "*" || client.Topic == "") &&
                    (client.ChainID == message.ChainID || client.ChainID == "*" || client.ChainID == "") {
                    select {
                    case client.SendCh <- message.Data:
                    default:
                        close(client.SendCh)
                        delete(m.clients, client)
                    }
                }
            }
            m.mutex.Unlock()
        }
    }
}

func (m *ClientManager) Register(client *Client) {
    m.register <- client
}

func (m *ClientManager) Unregister(client *Client) {
    m.unregister <- client
}

func (m *ClientManager) Broadcast(topic, chainID string, data map[string]interface{}) {
    m.broadcast <- &BroadcastMessage{
        Topic:   topic,
        ChainID: chainID,
        Data:    data,
    }
}

func (m *ClientManager) Count() int {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    return len(m.clients)
}

func (m *ClientManager) CloseAll() {
    m.mutex.Lock()
    defer m.mutex.Unlock()

    for client := range m.clients {
        client.Conn.Close()
        delete(m.clients, client)
    }
}

type Client struct {
    ID      string
    Conn    *websocket.Conn
    SendCh  chan map[string]interface{} 
    Topic   string 
    ChainID string 
}


func NewClient(conn *websocket.Conn) *Client {
    return &Client{
        ID:     fmt.Sprintf("%d", time.Now().UnixNano()),
        Conn:   conn,
        SendCh: make(chan map[string]interface{}, 256),
        Topic:  "",
        ChainID: "",
    }
}

func (c *Client) ReadPump(messageHandler func(*Client, []byte)) {
    defer func() {
        c.Conn.Close()
    }()

    c.Conn.SetReadLimit(4096) 

    _ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    c.Conn.SetPongHandler(func(string) error {
        _ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
        return nil
    })

    for {
        _, message, err := c.Conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err,
                websocket.CloseGoingAway,
                websocket.CloseAbnormalClosure) {
                log.Printf("Erro de leitura WebSocket: %v", err)
            }
            break
        }

        messageHandler(c, message)
    }
}

func (c *Client) WritePump() {
    ticker := time.NewTicker(30 * time.Second)
    defer func() {
        ticker.Stop()
        c.Conn.Close()
    }()

    for {
        select {
        case message, ok := <-c.SendCh:
            _ = c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if !ok {
                _ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }
            if err := c.Conn.WriteJSON(message); err != nil {
                log.Printf("Erro de escrita WebSocket: %v", err)
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

func (c *Client) Send(message map[string]interface{}) {
    select {
    case c.SendCh <- message:
    default:
        log.Printf("Canal de envio do cliente %s cheio, descartando mensagem", c.ID)
    }
}
