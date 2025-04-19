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
    kafkaProducer kafka.KafkaProducer
    wsEndpoint    string
    chainID       string
    chainName     string
    
    blocksCh      chan *newHeadsData
    errorCh       chan error
    stop          chan struct{}

    mutex         sync.Mutex
    conn          *websocket.Conn
    subID         string
    running       bool
    lastProcessedBlock uint64
}

func NewRealTimeExtractor(
    dataAPI *api.DataAPI,
    kafkaProducer kafka.KafkaProducer,
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
        
        blocksCh:      make(chan *newHeadsData, 500),
        errorCh:       make(chan error, 10),
        stop:          make(chan struct{}),
    }
}

func (r *RealTimeExtractor) Start(ctx context.Context) error {
    r.mutex.Lock()
    if r.running {
        r.mutex.Unlock()
        return fmt.Errorf("extrator já está em execução")
    }
    r.running = true
    r.mutex.Unlock()
    
    log.Printf("Iniciando extrator em tempo real para chain %s (%s)", r.chainName, r.chainID)
    
    var err error
    r.conn, _, err = websocket.DefaultDialer.Dial(r.wsEndpoint, nil)
    if err != nil {
        return fmt.Errorf("falha ao conectar ao WebSocket: %v", err)
    }
    
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
    
    go r.readMessages(ctx)
    go r.processBlocks(ctx)
    
    return nil
}

func (r *RealTimeExtractor) Stop() {
    r.mutex.Lock()
    if !r.running {
        r.mutex.Unlock()
        return
    }
    r.running = false
    r.mutex.Unlock()
    
    close(r.stop)
    
    if r.conn != nil {
        r.conn.Close()
    }
    
    log.Printf("Extrator em tempo real para chain %s parado", r.chainName)
}

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
            _, message, err := r.conn.ReadMessage()
            if err != nil {
                select {
                case r.errorCh <- fmt.Errorf("erro ao ler mensagem WebSocket: %v", err):
                case <-r.stop:
                    return
                }
                
                time.Sleep(5 * time.Second)
                if err := r.reconnect(); err != nil {
                    log.Printf("Falha ao reconectar: %v", err)
                    return
                }
                continue
            }
            
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
            
            if wsMsg.ID != nil && wsMsg.Result != "" {
                r.mutex.Lock()
                r.subID = wsMsg.Result
                r.mutex.Unlock()
                log.Printf("Subscrição newHeads confirmada: %s", wsMsg.Result)
                continue
            }
            if wsMsg.Method == "eth_subscription" && wsMsg.Params != nil && wsMsg.Params.Result.Hash != "" {
                blockHash := wsMsg.Params.Result.Hash
                log.Printf("Novo cabeçalho de bloco recebido: hash=%s número=%s", 
                    blockHash, wsMsg.Params.Result.Number)
                
                select {
                case r.blocksCh <- &wsMsg.Params.Result:
                case <-r.stop:
                    return
                }
            }
        }
    }
}

func (r *RealTimeExtractor) processBlocks(ctx context.Context) {
    for {
        select {
        case <-r.stop:
            return
        case blockHeader := <-r.blocksCh:
            // Add a sleep before fetching block details to prevent API rate limiting
            time.Sleep(500 * time.Millisecond)
            log.Printf("Processando bloco: hash=%s", blockHeader.Hash)
            
            block, err := r.dataAPI.GetBlockByNumberOrHash(r.chainID, blockHeader.Hash)
            if err != nil {
                log.Printf("Erro ao buscar detalhes do bloco %s: %v", blockHeader.Hash, err)
                continue
            }
            
            block.ChainID = r.chainID
            r.kafkaProducer.PublishBlock(*block)
        }
    }
}

func (r *RealTimeExtractor) reconnect() error {
    if r.conn != nil {
        r.conn.Close()
    }
    var err error
    r.conn, _, err = websocket.DefaultDialer.Dial(r.wsEndpoint, nil)
    if err != nil {
        return fmt.Errorf("falha ao reconectar ao WebSocket: %v", err)
    }
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

// GetName returns the service name
func (r *RealTimeExtractor) GetName() string {
    return "RealTimeExtractor-" + r.chainName
}

// IsRunning returns the running status
func (r *RealTimeExtractor) IsRunning() bool {
    r.mutex.Lock()
    defer r.mutex.Unlock()
    return r.running
}
