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

// Server representa um servidor WebSocket para streaming de dados para o frontend
type Server struct {
    port             string
    upgrader         websocket.Upgrader
    clientManager    *ClientManager
    kafkaConsumer    *kafka.Consumer
    server           *http.Server

    // Estatísticas
    mutex            sync.Mutex
    running          bool
    startTime        time.Time
    connectionsTotal int
    messagesTotal    int
}

// NewServer cria um novo servidor WebSocket
func NewServer(port string, kafkaConsumer *kafka.Consumer) *Server {
    upgrader := websocket.Upgrader{
        ReadBufferSize:  1024,
        WriteBufferSize: 1024,
        CheckOrigin: func(r *http.Request) bool {
            // Permite todas as origens por enquanto; ideal restringir em produção
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

// Start inicia o servidor WebSocket
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

    // Inicia o gerenciador de clientes
    go s.clientManager.Run()

    // Cria servidor HTTP
    mux := http.NewServeMux()
    mux.HandleFunc("/ws", s.handleWebSocket)
    mux.HandleFunc("/health", s.handleHealth)

    s.server = &http.Server{
        Addr:    fmt.Sprintf(":%s", s.port),
        Handler: mux,
    }

    log.Printf("Iniciando servidor WebSocket na porta %s", s.port)

    // Inicia consumidor Kafka para blocos, transações e whale alerts
    go s.consumeKafkaMessages()

    // Inicia o servidor em uma goroutine
    go func() {
        if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Printf("Erro no servidor WebSocket: %v", err)
        }
    }()

    return nil
}

// Stop interrompe o servidor WebSocket
func (s *Server) Stop(ctx context.Context) error {
    s.mutex.Lock()
    if !s.running {
        s.mutex.Unlock()
        return nil
    }
    s.running = false
    s.mutex.Unlock()

    // Fecha todas as conexões de clientes
    s.clientManager.CloseAll()

    // Desliga o servidor
    if err := s.server.Shutdown(ctx); err != nil {
        return fmt.Errorf("erro ao desligar servidor WebSocket: %w", err)
    }

    log.Printf("Servidor WebSocket parado")
    return nil
}

// handleWebSocket gerencia uma conexão WebSocket
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
    // Extrai parâmetros de inscrição da requisição
    topicParam := r.URL.Query().Get("topic")
    chainID := r.URL.Query().Get("chainId")

    // Atualiza a conexão HTTP para uma conexão WebSocket
    conn, err := s.upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Printf("Falha ao atualizar conexão: %v", err)
        return
    }

    // Cria um novo cliente
    client := NewClient(conn)
    client.Topic = topicParam
    client.ChainID = chainID

    // Registra o cliente no gerenciador de clientes
    s.clientManager.Register(client)

    // Incrementa contador de conexões
    s.mutex.Lock()
    s.connectionsTotal++
    s.mutex.Unlock()

    log.Printf("Nova conexão WebSocket: %s (tópico: %s, chainId: %s)",
        conn.RemoteAddr(), topicParam, chainID)

    // Gerencia mensagens do cliente (inscrições, etc.)
    go client.ReadPump(s.handleClientMessage)

    // Envia mensagens para o cliente
    go client.WritePump()
}

// handleClientMessage gerencia uma mensagem de um cliente
func (s *Server) handleClientMessage(client *Client, message []byte) {
    // Analisa a mensagem
    var msg map[string]interface{}
    if err := json.Unmarshal(message, &msg); err != nil {
        log.Printf("Erro ao analisar mensagem do cliente: %v", err)
        return
    }

    // Verifica mensagem de inscrição
    if action, ok := msg["action"].(string); ok && action == "subscribe" {
        topic, ok1 := msg["topic"].(string)
        chainID, ok2 := msg["chainId"].(string)

        if ok1 && ok2 {
            // Atualiza inscrição do cliente
            client.Topic = topic
            client.ChainID = chainID

            log.Printf("Cliente %s inscrito no tópico: %s, chainId: %s",
                client.ID, topic, chainID)

            // Envia reconhecimento de inscrição
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

// handleHealth gerencia requisições de verificação de saúde
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    s.mutex.Lock()
    defer s.mutex.Unlock()

    uptime := time.Since(s.startTime).String()
    if !s.running {
        uptime = "0s"
    }

    // Retorna status de saúde como JSON
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

// consumeKafkaMessages consome mensagens do Kafka e as transmite para os clientes
func (s *Server) consumeKafkaMessages() {
    // Inscreve-se em tópicos de blocos, transações e whale alerts
    topics := []string{
        "avalanche.blocks.*",       // Todos os tópicos de blocos
        "avalanche.transactions.*", // Todos os tópicos de transações
        "avalanche.whalealerts.*",  // Todos os tópicos de alertas de baleias
    }

    if err := s.kafkaConsumer.Subscribe(topics); err != nil {
        log.Printf("Erro ao inscrever-se em tópicos Kafka: %v", err)
        return
    }

    log.Printf("Inscrito em tópicos Kafka: %v", topics)

    for s.running {
        // Poll para mensagens (100ms)
        msg, err := s.kafkaConsumer.Poll(100)
        if err != nil {
            if s.running { // Só registra erro se ainda estivermos rodando
                log.Printf("Erro ao sondar Kafka: %v", err)
            }
            continue
        }

        // Se não chegou mensagem, pula
        if msg == nil {
            continue
        }

        // Processa a mensagem
        s.processKafkaMessage(msg)
    }
}

// processKafkaMessage processa uma mensagem do Kafka
func (s *Server) processKafkaMessage(msg *kafka.Message) {
    topic := msg.Topic
    parts := strings.Split(topic, ".")
    if len(parts) < 3 {
        log.Printf("Formato de tópico inválido: %s", topic)
        return
    }

    // Exemplo: avalanche.blocks.43114 => dataType = "blocks", chainID = "43114"
    dataType := parts[1]
    chainID := parts[2]

    // Análise do payload da mensagem como JSON
    var data map[string]interface{}
    if err := json.Unmarshal(msg.Value, &data); err != nil {
        log.Printf("Erro ao analisar mensagem Kafka: %v", err)
        return
    }

    // Adiciona metadados
    data["type"] = dataType
    data["chainId"] = chainID
    data["receivedAt"] = time.Now().Unix()

    // Transmite para clientes
    s.broadcastToClients(dataType, chainID, data)

    // Atualiza estatísticas
    s.mutex.Lock()
    s.messagesTotal++
    s.mutex.Unlock()
}

// broadcastToClients transmite uma mensagem para todos os clientes inscritos no tópico
func (s *Server) broadcastToClients(dataType, chainID string, data map[string]interface{}) {
    message := map[string]interface{}{
        "type":      dataType,
        "chainId":   chainID,
        "data":      data,
        "timestamp": time.Now().Unix(),
    }
    s.clientManager.Broadcast(dataType, chainID, message)
}

// Status retorna o status atual do servidor WebSocket
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

// ClientManager gerencia clientes WebSocket
type ClientManager struct {
    clients    map[*Client]bool
    register   chan *Client
    unregister chan *Client
    broadcast  chan *BroadcastMessage
    mutex      sync.Mutex
}

// BroadcastMessage representa uma mensagem a ser transmitida para clientes
type BroadcastMessage struct {
    Topic   string
    ChainID string
    Data    map[string]interface{}
}

// NewClientManager cria um novo gerenciador de clientes
func NewClientManager() *ClientManager {
    return &ClientManager{
        clients:    make(map[*Client]bool),
        register:   make(chan *Client),
        unregister: make(chan *Client),
        broadcast:  make(chan *BroadcastMessage),
    }
}

// Run inicia o gerenciador de clientes
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
                close(client.SendCh) // agora fechamos o canal SendCh
            }
            m.mutex.Unlock()

        case message := <-m.broadcast:
            m.mutex.Lock()
            for client := range m.clients {
                // Envia a mensagem para clientes que correspondem ao tópico e chainID ou usam curingas
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

// Register registra um cliente
func (m *ClientManager) Register(client *Client) {
    m.register <- client
}

// Unregister cancela o registro de um cliente
func (m *ClientManager) Unregister(client *Client) {
    m.unregister <- client
}

// Broadcast transmite uma mensagem para todos os clientes com inscrições correspondentes
func (m *ClientManager) Broadcast(topic, chainID string, data map[string]interface{}) {
    m.broadcast <- &BroadcastMessage{
        Topic:   topic,
        ChainID: chainID,
        Data:    data,
    }
}

// Count retorna o número de clientes conectados
func (m *ClientManager) Count() int {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    return len(m.clients)
}

// CloseAll fecha todas as conexões de clientes
func (m *ClientManager) CloseAll() {
    m.mutex.Lock()
    defer m.mutex.Unlock()

    for client := range m.clients {
        client.Conn.Close()
        delete(m.clients, client)
    }
}

// Client representa um cliente WebSocket
type Client struct {
    ID      string
    Conn    *websocket.Conn
    SendCh  chan map[string]interface{} // Renomeamos de "Send" para "SendCh"
    Topic   string // Tópico de inscrição (blocks, transactions, etc.)
    ChainID string // Filtro de Chain ID (43114, etc.)
}

// NewClient cria um novo cliente WebSocket
func NewClient(conn *websocket.Conn) *Client {
    return &Client{
        ID:     fmt.Sprintf("%d", time.Now().UnixNano()),
        Conn:   conn,
        SendCh: make(chan map[string]interface{}, 256),
        Topic:  "",
        ChainID: "",
    }
}

// ReadPump lê mensagens da conexão WebSocket
func (c *Client) ReadPump(messageHandler func(*Client, []byte)) {
    defer func() {
        c.Conn.Close()
    }()

    c.Conn.SetReadLimit(4096) // Tamanho máximo de mensagem permitido

    // Configura parâmetros WebSocket
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

// WritePump escreve mensagens na conexão WebSocket
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
                // Canal foi fechado
                _ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }

            // Envia mensagem como JSON
            if err := c.Conn.WriteJSON(message); err != nil {
                log.Printf("Erro de escrita WebSocket: %v", err)
                return
            }

        case <-ticker.C:
            // Mantém a conexão viva com um ping periódico
            _ = c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}

// Send envia uma mensagem para o cliente (usando o canal SendCh)
func (c *Client) Send(message map[string]interface{}) {
    select {
    case c.SendCh <- message:
    default:
        // Canal está cheio, cliente pode estar lento
        log.Printf("Canal de envio do cliente %s cheio, descartando mensagem", c.ID)
    }
}
