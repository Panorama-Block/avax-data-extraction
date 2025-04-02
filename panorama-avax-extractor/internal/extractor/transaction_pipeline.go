package extractor

import (
    "context"
    "fmt"
    "log"
    "sync"
    "time"

    "github.com/Panorama-Block/avax/internal/api"
    "github.com/Panorama-Block/avax/internal/kafka"
    "github.com/Panorama-Block/avax/internal/types"
)

type TransactionPipeline struct {
    dataAPI       *api.DataAPI
    kafkaProducer *kafka.Producer
    
    eventCh       chan types.WebhookEvent
    stop          chan struct{}
    
    workerCount   int
    bufferSize    int

    mutex         sync.Mutex
    processed     int64
    errors        int64
    running       bool
    startTime     time.Time
}

func NewTransactionPipeline(
    dataAPI *api.DataAPI,
    kafkaProducer *kafka.Producer,
    workerCount int,
    bufferSize int,
) *TransactionPipeline {
    return &TransactionPipeline{
        dataAPI:       dataAPI,
        kafkaProducer: kafkaProducer,
        eventCh:       make(chan types.WebhookEvent, bufferSize),
        stop:          make(chan struct{}),
        workerCount:   workerCount,
        bufferSize:    bufferSize,
    }
}

func (t *TransactionPipeline) Start(ctx context.Context) error {
    t.mutex.Lock()
    if t.running {
        t.mutex.Unlock()
        return fmt.Errorf("pipeline de transações já está em execução")
    }
    t.running = true
    t.startTime = time.Now()
    t.processed = 0
    t.errors = 0
    t.mutex.Unlock()
    
    log.Printf("Iniciando pipeline de processamento de transações com %d workers", t.workerCount)
    
    for i := 0; i < t.workerCount; i++ {
        go t.worker(ctx, i)
    }
    
    return nil
}

func (t *TransactionPipeline) Stop() {
    t.mutex.Lock()
    defer t.mutex.Unlock()
    
    if !t.running {
        return
    }
    
    close(t.stop)
    t.running = false
    log.Printf("Interrompendo pipeline de processamento de transações")
}

func (t *TransactionPipeline) Submit(event types.WebhookEvent) error {
    if !t.running {
        return fmt.Errorf("pipeline de transações não está em execução")
    }
    
    select {
    case t.eventCh <- event:
        return nil
    default:
        return fmt.Errorf("canal de eventos está cheio, considere aumentar o tamanho do buffer")
    }
}

func (t *TransactionPipeline) worker(ctx context.Context, id int) {
    log.Printf("Worker de transação %d iniciado", id)
    
    for {
        select {
        case <-t.stop:
            log.Printf("Worker de transação %d parando", id)
            return
        case event := <-t.eventCh:
            txChan := make(chan *types.Transaction, 1)
            erc20Chan := make(chan types.ERC20Transfer, len(event.Event.Transaction.ERC20Transfers))
            erc721Chan := make(chan types.ERC721Transfer, len(event.Event.Transaction.ERC721Transfers))
            erc1155Chan := make(chan types.ERC1155Transfer, len(event.Event.Transaction.ERC1155Transfers))
            logChan := make(chan types.Log, len(event.Event.Transaction.Logs))
            
            txChan <- &event.Event.Transaction
            
            for _, e20 := range event.Event.Transaction.ERC20Transfers {
                erc20Chan <- e20
            }
            for _, e721 := range event.Event.Transaction.ERC721Transfers {
                erc721Chan <- e721
            }
            for _, e1155 := range event.Event.Transaction.ERC1155Transfers {
                erc1155Chan <- e1155
            }
            for _, lg := range event.Event.Transaction.Logs {
                logChan <- lg
            }
            
            close(txChan)
            close(erc20Chan)
            close(erc721Chan)
            close(erc1155Chan)
            close(logChan)
            
            go t.kafkaProducer.PublishTransactions(txChan)
            go t.kafkaProducer.PublishERC20Transfers(erc20Chan)
            go t.kafkaProducer.PublishERC721Transfers(erc721Chan)
            go t.kafkaProducer.PublishERC1155Transfers(erc1155Chan)
            go t.kafkaProducer.PublishLogs(logChan)
            
            t.mutex.Lock()
            t.processed++
            t.mutex.Unlock()
        }
    }
}

func (t *TransactionPipeline) Status() map[string]interface{} {
    t.mutex.Lock()
    defer t.mutex.Unlock()
    
    uptime := time.Since(t.startTime).String()
    if !t.running {
        uptime = "0s"
    }
    
    return map[string]interface{}{
        "running":        t.running,
        "workerCount":    t.workerCount,
        "bufferSize":     t.bufferSize,
        "currentBuffer":  len(t.eventCh),
        "processed":      t.processed,
        "errors":         t.errors,
        "uptime":         uptime,
        "startTime":      t.startTime.Format(time.RFC3339),
    }
}
