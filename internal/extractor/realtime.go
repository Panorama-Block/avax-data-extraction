package extractor

import (
    "encoding/json"
    "log"
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

func StartBlockWebSocket(client *api.Client, producer *kafka.Producer) {
    log.Println("[WS] Conectando em wss://api.avax.network/ext/bc/C/ws")

    conn, _, err := websocket.DefaultDialer.Dial("wss://api.avax.network/ext/bc/C/ws", nil)
    if err != nil {
        log.Fatalf("[WS] Erro ao conectar: %v", err)
    }
    defer conn.Close()

    subReq := map[string]interface{}{
        "jsonrpc": "2.0",
        "id":      1,
        "method":  "eth_subscribe",
        "params":  []interface{}{"newHeads"},
    }
    if err := conn.WriteJSON(subReq); err != nil {
        log.Fatalf("[WS] Erro ao subscrever newHeads: %v", err)
    }

    for {
        _, message, err := conn.ReadMessage()
        if err != nil {
            log.Printf("[WS] Erro ao ler msg: %v", err)
            time.Sleep(3 * time.Second)
            continue
        }

        var wsMsg struct {
            JSONRPC string `json:"jsonrpc"`
            Method  string `json:"method"`
            Params  struct {
                Subscription string       `json:"subscription"`
                Result       newHeadsData `json:"result"`
            } `json:"params"`
        }
        if err := json.Unmarshal(message, &wsMsg); err != nil {
            continue
        }

        if wsMsg.Method == "eth_subscription" && wsMsg.Params.Result.Hash != "" {
            blockHash := wsMsg.Params.Result.Hash
            blockNum  := wsMsg.Params.Result.Number
            log.Printf("[WS] Cabeçalho: blockHash=%s number=%s parentHash=%s",
                blockHash, blockNum, wsMsg.Params.Result.ParentHash)

            go fetchAndPublishBlock(client, producer, "43114", blockHash)
        }
    }
}

func fetchAndPublishBlock(client *api.Client, producer *kafka.Producer, chainID, blockHash string) {
    time.Sleep(800 * time.Millisecond)

    log.Printf("[WS] fetchAndPublishBlock => chainID=%s blockHash=%s", chainID, blockHash)

    block, err := client.GetBlockByNumberOrHash(chainID, blockHash)
    if err != nil {
        log.Printf("[WS] ERRO => blockHash=%s erro=%v", blockHash, err)
        return
    }
    block.ChainID = chainID
    log.Printf("[WS] Bloco OK => blockNumber=%s, blockHash=%s (publicando no Kafka)",
        block.BlockNumber, block.BlockHash)
    producer.PublishBlock(*block)

    // if block.BlockNumber != "" {
    //     txs, _, err := client.GetTransactions(chainID, map[string]string{"blockNumber": block.BlockNumber})
    //     if err != nil {
    //         log.Printf("[WS] erro GetTransactions => blockNumber=%s, err=%v", block.BlockNumber, err)
    //         return
    //     }
    //     // ...
    //     log.Printf("[WS] Bloco %s => %d TXs", block.BlockNumber, len(txs))
    //     for _, t := range txs {
    //         enriched, err2 := client.GetTransactionByHash(chainID, t.TxHash)
    //         if err2 == nil && enriched != nil {
    //             t = *enriched
    //         }
    //         producer.PublishSingleTx(&t)
    //     }
    // }
}
