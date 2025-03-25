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
    KafkaBroker            string
    KafkaTopicChains       string
    KafkaTopicBlocks       string
    KafkaTopicTransactions string
    KafkaTopicLogs         string
    KafkaTopicERC20        string
    KafkaTopicERC721       string
    KafkaTopicERC1155      string
    KafkaTopicMetrics      string

    KafkaTopicSubnets     string
    KafkaTopicBlockchains string
    KafkaTopicValidators  string
    KafkaTopicDelegators  string
    KafkaTopicBridges     string

    WebhookPort    string
    WebSocketPort  string
    Workers        int
    TokenAddresses []string
}

func LoadConfig() *Config {
    err := godotenv.Load(".env")
    if err != nil {
        log.Println("Aviso: .env não encontrado, usando variáveis de ambiente do sistema")
    }

    workers, err := strconv.Atoi(os.Getenv("WORKERS"))
    if err != nil || workers <= 0 {
        workers = 5
    }

    tokenAddressesEnv := os.Getenv("TOKEN_ADDRESSES")
    var tokenAddresses []string
    if tokenAddressesEnv != "" {
        tokenAddresses = append(tokenAddresses, tokenAddressesEnv)
    }

    return &Config{
        APIUrl:                 os.Getenv("API_URL"),
        APIKey:                 os.Getenv("API_KEY"),
        ChainWsUrl:             os.Getenv("CHAIN_WS_URL"),
        KafkaBroker:            os.Getenv("KAFKA_BROKER"),
        KafkaTopicChains:       os.Getenv("KAFKA_TOPIC_CHAINS"),
        KafkaTopicBlocks:       os.Getenv("KAFKA_TOPIC_BLOCKS"),
        KafkaTopicTransactions: os.Getenv("KAFKA_TOPIC_TRANSACTIONS"),
        KafkaTopicLogs:         os.Getenv("KAFKA_TOPIC_LOGS"),
        KafkaTopicERC20:        os.Getenv("KAFKA_TOPIC_ERC20"),
        KafkaTopicERC721:       os.Getenv("KAFKA_TOPIC_ERC721"),
        KafkaTopicERC1155:      os.Getenv("KAFKA_TOPIC_ERC1155"),
        KafkaTopicMetrics:      os.Getenv("KAFKA_TOPIC_METRICS"),

        KafkaTopicSubnets:     os.Getenv("KAFKA_TOPIC_SUBNETS"),
        KafkaTopicBlockchains: os.Getenv("KAFKA_TOPIC_BLOCKCHAINS"),
        KafkaTopicValidators:  os.Getenv("KAFKA_TOPIC_VALIDATORS"),
        KafkaTopicDelegators:  os.Getenv("KAFKA_TOPIC_DELEGATORS"),
        KafkaTopicBridges:     os.Getenv("KAFKA_TOPIC_BRIDGES"),

        WebhookPort:    os.Getenv("WEBHOOK_PORT"),
        WebSocketPort:  os.Getenv("WEBSOCKET_PORT"),
        Workers:        workers,
        TokenAddresses: tokenAddresses,
    }
}
