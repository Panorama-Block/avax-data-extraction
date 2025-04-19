package extractor

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Panorama-Block/avax/internal/api"
	"github.com/Panorama-Block/avax/internal/kafka"
)

// CronJobManager manages periodic data extraction jobs
type CronJobManager struct {
    client    *api.Client
    producer  kafka.KafkaProducer
    network   string
    
    ticker    *time.Ticker
    stop      chan struct{}
    mutex     sync.Mutex
    running   bool
}

// NewCronJobManager creates a new cron job manager
func NewCronJobManager(client *api.Client, producer kafka.KafkaProducer, network string) *CronJobManager {
    return &CronJobManager{
        client:   client,
        producer: producer,
        network:  network,
        stop:     make(chan struct{}),
        running:  false,
    }
}

// Start starts the cron job manager
func (c *CronJobManager) Start() error {
    c.mutex.Lock()
    if c.running {
        c.mutex.Unlock()
        return fmt.Errorf("cron job manager já está em execução")
    }
    c.running = true
    c.mutex.Unlock()
    
    log.Printf("Iniciando gerenciador de tarefas cron para rede %s", c.network)
    
    // Execute jobs immediately
    err := c.executeJobs()
    if err != nil {
        log.Printf("Erro na execução inicial de tarefas cron: %v", err)
    }
    
    // Schedule periodic execution
    c.ticker = time.NewTicker(10 * time.Minute)
    go func() {
        for {
            select {
            case <-c.ticker.C:
                if err := c.executeJobs(); err != nil {
                    log.Printf("Erro na execução de tarefas cron: %v", err)
                }
            case <-c.stop:
                return
            }
        }
    }()
    
    return nil
}

// Stop stops the cron job manager
func (c *CronJobManager) Stop() {
    c.mutex.Lock()
    defer c.mutex.Unlock()
    
    if !c.running {
        return
    }
    
    if c.ticker != nil {
        c.ticker.Stop()
    }
    
    close(c.stop)
    c.running = false
    log.Println("Gerenciador de tarefas cron parado")
}

// IsRunning returns whether the cron job manager is running
func (c *CronJobManager) IsRunning() bool {
    c.mutex.Lock()
    defer c.mutex.Unlock()
    return c.running
}

// GetName returns the name of the service
func (c *CronJobManager) GetName() string {
    return "CronJobManager"
}

// executeJobs executes all scheduled jobs
func (c *CronJobManager) executeJobs() error {
    dataAPI := api.NewDataAPI(c.client)
    
    err := fetchSubnetsAndBlockchains(dataAPI, c.producer, c.network)
    if err != nil {
        log.Printf("[CRON] Erro subnets: %v", err)
    }

    err = fetchValidators(dataAPI, c.producer, c.network)
    if err != nil {
        log.Printf("[CRON] Erro validators: %v", err)
    }

    err = fetchDelegators(dataAPI, c.producer, c.network)
    if err != nil {
        log.Printf("[CRON] Erro delegators: %v", err)
    }

    err = fetchTeleporterTxs(dataAPI, c.producer)
    if err != nil {
        log.Printf("[CRON] Erro teleporter: %v", err)
    }
    
    return nil
}

// StartCronJobs starts a new cron job manager
func StartCronJobs(client *api.Client, producer kafka.KafkaProducer, network string) *CronJobManager {
    manager := NewCronJobManager(client, producer, network)
    if err := manager.Start(); err != nil {
        log.Printf("Erro ao iniciar gerenciador de tarefas cron: %v", err)
    }
    return manager
}

func fetchSubnetsAndBlockchains(dataAPI *api.DataAPI, producer kafka.KafkaProducer, network string) error {
    subnets, err := dataAPI.GetSubnets(network)
    if err != nil {
        return err
    }
    for _, s := range subnets {
        if s.SubnetID != "" {
            producer.PublishSubnet(s)
            det, err2 := dataAPI.GetSubnetByID(network, s.SubnetID)
            if err2 == nil && det != nil {
                if det.SubnetID != "" {
                    producer.PublishSubnet(*det)
                    for _, bc := range det.Blockchains {
                        producer.PublishBlockchain(bc)
                    }
                }
            }
        }
    }
    bcs, err := dataAPI.GetBlockchains(network)
    if err == nil {
        for _, bc := range bcs {
            producer.PublishBlockchain(bc)
        }
    }
    return nil
}

func fetchValidators(dataAPI *api.DataAPI, producer kafka.KafkaProducer, network string) error {
    validators, err := dataAPI.GetValidators(network)
    if err != nil {
        return err
    }
    for _, v := range validators {
        producer.PublishValidator(v)
    }
    return nil
}

func fetchDelegators(dataAPI *api.DataAPI, producer kafka.KafkaProducer, network string) error {
    delegators, err := dataAPI.GetDelegators(network)
    if err != nil {
        return err
    }
    for _, d := range delegators {
        producer.PublishDelegator(d)
    }
    return nil
}

func fetchTeleporterTxs(dataAPI *api.DataAPI, producer kafka.KafkaProducer) error {
    // Looking at C-Chain to P-Chain bridge transactions
    txs, err := dataAPI.GetTeleporterTxs("C", "P")
    if err != nil {
        return err
    }
    for _, tx := range txs {
        producer.PublishBridgeTx(tx)
    }
    return nil
}
