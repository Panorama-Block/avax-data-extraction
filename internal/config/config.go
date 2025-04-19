package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
    APIUrl                 string
    APIKey                 string
    ChainWsUrl             string
    APIRateLimit           float64  // Request rate limit (requests per second)
    APIRateBurst           int      // Maximum burst size for the rate limiter
    KafkaBroker            string
    KafkaTopicChains       string
    KafkaTopicBlocks       string
    KafkaTopicTransactions string
    KafkaTopicLogs         string
    KafkaTopicERC20        string
    KafkaTopicERC721       string
    KafkaTopicERC1155      string
    KafkaTopicInternalTx   string
    KafkaTopicMetrics      string
    
    // New specific metrics topics
    KafkaTopicActivityMetrics    string
    KafkaTopicPerformanceMetrics string
    KafkaTopicGasMetrics         string
    KafkaTopicCumulativeMetrics  string

    KafkaTopicSubnets     string
    KafkaTopicBlockchains string
    KafkaTopicValidators  string
    KafkaTopicDelegators  string
    KafkaTopicBridges     string

    WebhookPort    string
    WebSocketPort  string
    Workers        int
    TokenAddresses []string

    // Enable realtime extraction of blocks via WebSocket
    EnableRealtime bool
    
    // Enable creation of C-Chain webhook on AvaCloud
    CreateCChainWebhook bool
    
    // AvaCloud Webhook Configuration
    AvaCloudAPIKey      string
    AvaCloudAPIBaseURL  string
    WebhookURL          string
    WebhookSecret       string
}

func LoadConfig() *Config {
    err := godotenv.Load(".env")
    if err != nil {
        log.Println("WARN: .env não encontrado")
    }

    workers, err := strconv.Atoi(os.Getenv("WORKERS"))
    if err != nil || workers <= 0 {
        workers = 10
    }

    // Parse API rate limit settings with defaults
    apiRateLimit, err := strconv.ParseFloat(os.Getenv("API_RATE_LIMIT"), 64)
    if err != nil || apiRateLimit <= 0 {
        apiRateLimit = 5.0 // Default: 5 requests per second
    }

    apiRateBurst, err := strconv.Atoi(os.Getenv("API_RATE_BURST"))
    if err != nil || apiRateBurst <= 0 {
        apiRateBurst = 10 // Default: burst of 10 requests
    }

    tokenAddressesEnv := os.Getenv("TOKEN_ADDRESSES")
    var tokenAddresses []string
    if tokenAddressesEnv != "" {
        tokenAddresses = append(tokenAddresses, tokenAddressesEnv)
    }

    // Get default metrics topic as fallback for specific metrics topics
    defaultMetricsTopic := os.Getenv("KAFKA_TOPIC_METRICS")
    if defaultMetricsTopic == "" {
        log.Fatalf("ERRO: KAFKA_TOPIC_METRICS não pode estar vazio!")
    }
    
    // Try to get specific metrics topics or fall back to the default metrics topic
    activityMetricsTopic := os.Getenv("KAFKA_TOPIC_ACTIVITY_METRICS")
    if activityMetricsTopic == "" {
        activityMetricsTopic = defaultMetricsTopic
    }
    
    performanceMetricsTopic := os.Getenv("KAFKA_TOPIC_PERFORMANCE_METRICS")
    if performanceMetricsTopic == "" {
        performanceMetricsTopic = defaultMetricsTopic
    }
    
    gasMetricsTopic := os.Getenv("KAFKA_TOPIC_GAS_METRICS")
    if gasMetricsTopic == "" {
        gasMetricsTopic = defaultMetricsTopic
    }
    
	cumulativeMetricsTopic := os.Getenv("KAFKA_TOPIC_CUMULATIVE_METRICS")
    if cumulativeMetricsTopic == "" {
        cumulativeMetricsTopic = defaultMetricsTopic
    }

    // Parse the EnableRealtime setting
    enableRealtimeStr := os.Getenv("ENABLE_REALTIME")
    enableRealtime := false
    if enableRealtimeStr == "true" || enableRealtimeStr == "1" || enableRealtimeStr == "yes" {
        enableRealtime = true
    }

    // Parse the CreateCChainWebhook setting
    createCChainWebhookStr := os.Getenv("CREATE_C_CHAIN_WEBHOOK")
    createCChainWebhook := false
    if createCChainWebhookStr == "true" || createCChainWebhookStr == "1" || createCChainWebhookStr == "yes" {
        createCChainWebhook = true
    }

    return &Config{
        APIUrl:                 os.Getenv("API_URL"),
        APIKey:                 os.Getenv("API_KEY"),
        ChainWsUrl:             os.Getenv("CHAIN_WS_URL"),
        APIRateLimit:           apiRateLimit,
        APIRateBurst:           apiRateBurst,

        KafkaBroker:            os.Getenv("KAFKA_BROKER"),
        KafkaTopicChains:       os.Getenv("KAFKA_TOPIC_CHAINS"),
        KafkaTopicBlocks:       os.Getenv("KAFKA_TOPIC_BLOCKS"),
        KafkaTopicTransactions: os.Getenv("KAFKA_TOPIC_TRANSACTIONS"),
        KafkaTopicLogs:         os.Getenv("KAFKA_TOPIC_LOGS"),
        KafkaTopicERC20:        os.Getenv("KAFKA_TOPIC_ERC20"),
        KafkaTopicERC721:       os.Getenv("KAFKA_TOPIC_ERC721"),
        KafkaTopicERC1155:      os.Getenv("KAFKA_TOPIC_ERC1155"),
        KafkaTopicInternalTx:   os.Getenv("KAFKA_TOPIC_INTERNAL_TX"),
        KafkaTopicMetrics:      defaultMetricsTopic,
        
        // Assign specific metrics topics
        KafkaTopicActivityMetrics:    activityMetricsTopic,
        KafkaTopicPerformanceMetrics: performanceMetricsTopic,
        KafkaTopicGasMetrics:         gasMetricsTopic,
        KafkaTopicCumulativeMetrics:  cumulativeMetricsTopic,
        
        KafkaTopicSubnets:     os.Getenv("KAFKA_TOPIC_SUBNETS"),
        KafkaTopicBlockchains: os.Getenv("KAFKA_TOPIC_BLOCKCHAINS"),
        KafkaTopicValidators:  os.Getenv("KAFKA_TOPIC_VALIDATORS"),
        KafkaTopicDelegators:  os.Getenv("KAFKA_TOPIC_DELEGATORS"),
        KafkaTopicBridges:     os.Getenv("KAFKA_TOPIC_BRIDGES"),

        WebhookPort:    os.Getenv("WEBHOOK_PORT"),
        WebSocketPort:  os.Getenv("WEBSOCKET_PORT"),
        Workers:        workers,
        TokenAddresses: tokenAddresses,
        EnableRealtime: enableRealtime,
        CreateCChainWebhook: createCChainWebhook,
        
        // AvaCloud Webhook Configuration
        AvaCloudAPIKey:     os.Getenv("AVACLOUD_API_KEY"),
        AvaCloudAPIBaseURL: os.Getenv("AVACLOUD_API_BASE_URL"),
        WebhookURL:         os.Getenv("WEBHOOK_URL"),
        WebhookSecret:      os.Getenv("WEBHOOK_SECRET"),
    }
}
