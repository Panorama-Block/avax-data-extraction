package config
import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	APIUrl string
	APIKey string
	KafkaBroker string
	KafkaTopic string
}

func LoadConfig() *Config {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file", err)
	}

	return &Config{
		APIUrl: os.Getenv("API_URL"),
		APIKey: os.Getenv("API_KEY"),
		KafkaBroker: os.Getenv("KAFKA_BROKER"),
		KafkaTopic: os.Getenv("KAFKA_TOPIC"),
	}
}