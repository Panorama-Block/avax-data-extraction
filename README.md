# Avalanche Data Extractor

This application extracts data from the Avalanche blockchain and publishes it to Kafka topics.

## Requirements

- Docker and Docker Compose

## Running with Docker

1. Start all services using Docker Compose:

```bash
docker-compose up -d
```

2. Check if all services are running:

```bash
docker-compose ps
```

3. Access Kafka UI to monitor topics:
   - Open http://localhost:8080 in your browser

4. To check logs from the application:

```bash
docker-compose logs -f avax-extractor
```

5. To stop all services:

```bash
docker-compose down
```

## Running locally

If you want to run the application locally without Docker:

1. Make sure you have a Kafka broker running at `localhost:9092`

2. Run the application:

```bash
go run cmd/main.go
```

## Troubleshooting

### Kafka Connection Issues

If you see errors like:

```
FAIL|rdkafka#producer-1| [thrd:localhost:9092/bootstrap]: localhost:9092/bootstrap: Connect to ipv4#127.0.0.1:9092 failed: Connection refused
```

It means the application can't connect to Kafka. Make sure:

1. Kafka is running at the address specified in your `.env` file
2. No firewall is blocking the connection
3. If using Docker, check that the network configuration is correct

### Other Issues

Check the logs for specific errors:

```bash
docker-compose logs -f
``` 