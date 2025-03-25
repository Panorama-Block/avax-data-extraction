package extractor

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "sync"
    "time"

    "github.com/Panorama-Block/avax/internal/api"
    "github.com/Panorama-Block/avax/internal/kafka"
    "github.com/gorilla/websocket"
)

type newHeadsData struct {
    Hash       string `json:"hash"`
    ParentHash string `json:"parentHash"`
    Number     string `json:"number"`
}

type RealTimeExtractor struct {
    dataAPI       *api.DataAPI
    kafkaProducer *kafka.Producer
    wsEndpoint    string
    chainID       string
    chainName     string
    
    // Canais de controle
    blocksCh      chan *newHeadsData
    errorCh       chan error
    stop          chan struct{}
    
    // Estado em execução
    mutex         sync.Mutex
    conn          *websocket.Conn
    subID         string
    running       bool
    lastProcessedBlock uint64
}

// NewRealTimeExtractor cria um novo extrator em tempo real
func NewRealTimeExtractor(
    dataAPI *api.DataAPI,
    kafkaProducer *kafka.Producer,
    wsEndpoint string,
    chainID string,
    chainName string,
) *RealTimeExtractor {
    return &RealTimeExtractor{
        dataAPI:       dataAPI,
        kafkaProducer: kafkaProducer,
        wsEndpoint:    wsEndpoint,
        chainID:       chainID,
        chainName:     chainName,
        
        blocksCh:      make(chan *newHeadsData, 100),
        errorCh:       make(chan error, 10),
        stop:          make(chan struct{}),
    }
}

// Start inicia o processo de extração em tempo real
func (r *RealTimeExtractor) Start(ctx context.Context) error {
    r.mutex.Lock()
    if r.running {
        r.mutex.Unlock()
        return fmt.Errorf("extrator já está em execução")
    }
    r.running = true
    r.mutex.Unlock()
    
    log.Printf("Iniciando extrator em tempo real para chain %s (%s)", r.chainName, r.chainID)
    
    // Conectar ao WebSocket
    var err error
    r.conn, _, err = websocket.DefaultDialer.Dial(r.wsEndpoint, nil)
    if err != nil {
        return fmt.Errorf("falha ao conectar ao WebSocket: %v", err)
    }
    
    // Enviar requisição de subscrição para newHeads
    subReq := map[string]interface{}{
        "jsonrpc": "2.0",
        "id":      1,
        "method":  "eth_subscribe",
        "params":  []interface{}{"newHeads"},
    }
    
    if err := r.conn.WriteJSON(subReq); err != nil {
        r.conn.Close()
        return fmt.Errorf("falha ao enviar requisição de subscrição: %v", err)
    }
    
    // Iniciar goroutines
    go r.readMessages(ctx)
    go r.processBlocks(ctx)
    
    return nil
}

// Stop interrompe o extrator em tempo real
func (r *RealTimeExtractor) Stop() {
    r.mutex.Lock()
    if !r.running {
        r.mutex.Unlock()
        return
    }
    r.running = false
    r.mutex.Unlock()
    
    // Fechar canal de parada para sinalizar às goroutines
    close(r.stop)
    
    // Fechar conexão WebSocket
    if r.conn != nil {
        r.conn.Close()
    }
    
    log.Printf("Extrator em tempo real para chain %s parado", r.chainName)
}

// readMessages lê mensagens da conexão WebSocket
func (r *RealTimeExtractor) readMessages(ctx context.Context) {
    defer func() {
        if r.conn != nil {
            r.conn.Close()
        }
    }()
    
    for {
        select {
        case <-r.stop:
            return
        default:
            // Lê a próxima mensagem
            _, message, err := r.conn.ReadMessage()
            if err != nil {
                select {
                case r.errorCh <- fmt.Errorf("erro ao ler mensagem WebSocket: %v", err):
                case <-r.stop:
                    return
                }
                
                // Tentar reconectar após erro
                time.Sleep(5 * time.Second)
                if err := r.reconnect(); err != nil {
                    log.Printf("Falha ao reconectar: %v", err)
                    return
                }
                continue
            }
            
            // Processa a mensagem
            var wsMsg struct {
                JSONRPC string `json:"jsonrpc"`
                ID      *int   `json:"id,omitempty"`
                Method  string `json:"method,omitempty"`
                Result  string `json:"result,omitempty"`
                Params  *struct {
                    Subscription string       `json:"subscription"`
                    Result       newHeadsData `json:"result"`
                } `json:"params,omitempty"`
            }
            
            if err := json.Unmarshal(message, &wsMsg); err != nil {
                log.Printf("Erro ao decodificar mensagem WebSocket: %v", err)
                continue
            }
            
            // Verifica se é resposta à subscrição
            if wsMsg.ID != nil && wsMsg.Result != "" {
                r.mutex.Lock()
                r.subID = wsMsg.Result
                r.mutex.Unlock()
                log.Printf("Subscrição newHeads confirmada: %s", wsMsg.Result)
                continue
            }
            
            // Verifica se é notificação de novo bloco
            if wsMsg.Method == "eth_subscription" && wsMsg.Params != nil && wsMsg.Params.Result.Hash != "" {
                blockHash := wsMsg.Params.Result.Hash
                log.Printf("Novo cabeçalho de bloco recebido: hash=%s número=%s", 
                    blockHash, wsMsg.Params.Result.Number)
                
                // Envia para o canal de blocos
                select {
                case r.blocksCh <- &wsMsg.Params.Result:
                case <-r.stop:
                    return
                }
            }
        }
    }
}

// processBlocks processa os blocos recebidos
func (r *RealTimeExtractor) processBlocks(ctx context.Context) {
    for {
        select {
        case <-r.stop:
            return
        case blockHeader := <-r.blocksCh:
            // Pequeno atraso para garantir que o bloco esteja disponível na API
            time.Sleep(800 * time.Millisecond)
            
            log.Printf("Processando bloco: hash=%s", blockHeader.Hash)
            
            // Busca detalhes do bloco na API
            block, err := r.dataAPI.GetBlockByNumberOrHash(r.chainID, blockHeader.Hash)
            if err != nil {
                log.Printf("Erro ao buscar detalhes do bloco %s: %v", blockHeader.Hash, err)
                continue
            }
            
            // Garante que o chainID está definido
            block.ChainID = r.chainID
            
            // Publica o bloco no Kafka
            r.kafkaProducer.PublishBlock(*block)
        }
    }
}

// reconnect tenta reconectar ao WebSocket e reinscrever-se
func (r *RealTimeExtractor) reconnect() error {
    // Fechar conexão existente
    if r.conn != nil {
        r.conn.Close()
    }
    
    // Reconectar
    var err error
    r.conn, _, err = websocket.DefaultDialer.Dial(r.wsEndpoint, nil)
    if err != nil {
        return fmt.Errorf("falha ao reconectar ao WebSocket: %v", err)
    }
    
    // Reenviar requisição de subscrição
    subReq := map[string]interface{}{
        "jsonrpc": "2.0",
        "id":      1,
        "method":  "eth_subscribe",
        "params":  []interface{}{"newHeads"},
    }
    
    if err := r.conn.WriteJSON(subReq); err != nil {
        r.conn.Close()
        return fmt.Errorf("falha ao enviar requisição de subscrição: %v", err)
    }
    
    log.Printf("Reconectado ao WebSocket para chain %s", r.chainName)
    return nil
}
