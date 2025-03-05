package extractor

import (
    "log"
    "sync"

    "github.com/Panorama-Block/avax/internal/api"
    "github.com/Panorama-Block/avax/internal/kafka"
    "github.com/Panorama-Block/avax/internal/types"
)

func StartTxPipeline(client *api.Client, producer *kafka.Producer, eventChan chan types.WebhookEvent, numWorkers int) {
    txChan := make(chan *types.Transaction, numWorkers)
    erc20Chan := make(chan types.ERC20Transfer, numWorkers)
    erc721Chan := make(chan types.ERC721Transfer, numWorkers)
    erc1155Chan := make(chan types.ERC1155Transfer, numWorkers)
    logChan := make(chan types.Log, numWorkers)

    var wg sync.WaitGroup
    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        go processWebhookEvents(client, eventChan, txChan, erc20Chan, erc721Chan, erc1155Chan, logChan, &wg)
    }

    go producer.PublishTransactions(txChan)
    go producer.PublishERC20Transfers(erc20Chan)
    go producer.PublishERC721Transfers(erc721Chan)
    go producer.PublishERC1155Transfers(erc1155Chan)
    go producer.PublishLogs(logChan)

    wg.Wait()
    close(txChan)
    close(erc20Chan)
    close(erc721Chan)
    close(erc1155Chan)
    close(logChan)
}

func processWebhookEvents(
    client *api.Client,
    eventChan <-chan types.WebhookEvent,
    txChan chan<- *types.Transaction,
    erc20Chan chan<- types.ERC20Transfer,
    erc721Chan chan<- types.ERC721Transfer,
    erc1155Chan chan<- types.ERC1155Transfer,
    logChan chan<- types.Log,
    wg *sync.WaitGroup,
) {
    defer wg.Done()

    for evt := range eventChan {
        log.Printf("[Webhook] Recebido eventType=%s, msgID=%s", evt.EventType, evt.MessageID)

        rawTx := evt.Event.Transaction
    
        txChan <- &rawTx

        for _, e20 := range rawTx.ERC20Transfers {
            erc20Chan <- e20
        }
        for _, e721 := range rawTx.ERC721Transfers {
            erc721Chan <- e721
        }
        for _, e1155 := range rawTx.ERC1155Transfers {
            erc1155Chan <- e1155
        }
        for _, lg := range rawTx.Logs {
            logChan <- lg
        }
    }
}
