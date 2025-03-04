package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	APIUrl      string
	APIKey      string
	KafkaBroker string
	KafkaTopic  string
	WebhookPort string
	Workers     int
}

func LoadConfig() *Config {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println(".env not found")
	}

	workers, err := strconv.Atoi(os.Getenv("WORKERS"))
	if err != nil || workers <= 0 {
		workers = 5
	}

	return &Config{
		APIUrl:      os.Getenv("API_URL"),
		APIKey:      os.Getenv("API_KEY"),
		KafkaBroker: os.Getenv("KAFKA_BROKER"),
		KafkaTopic:  os.Getenv("KAFKA_TOPIC"),
		WebhookPort: os.Getenv("WEBHOOK_PORT"),
		Workers:     workers,
	}
}
