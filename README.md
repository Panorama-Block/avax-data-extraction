# AVAX Data Extraction

## Project Overview
AVAX Data Extraction is a robust data extraction and processing system designed for the Avalanche blockchain. This service provides real-time data extraction capabilities for various blockchain components including blocks, transactions, logs, and token events (ERC20, ERC721, ERC1155). The system is built with scalability in mind, utilizing Kafka for message streaming and providing both WebSocket and Webhook interfaces for real-time data access.

### Architecture Overview
The system follows an event-driven microservices architecture with the following key components:

1. **Data Extraction Layer**
   - Real-time block processing through WebSocket connections
   - Historical data extraction through REST API endpoints
   - Rate-limited API integration to prevent overload
   - Parallel processing through configurable worker pools

2. **Event Processing Pipeline**
   - Kafka-based event streaming architecture
   - Separate topics for different event types (blocks, transactions, logs, etc.)
   - Event deduplication and ordering guarantees
   - Configurable consumer groups for parallel processing

3. **Integration Layer**
   - WebSocket server for real-time data streaming
   - Webhook endpoints for event notifications
   - AvaCloud integration for enhanced blockchain data access
   - REST API endpoints for data querying

4. **Data Types and Processing**
   - Block data: Header information, transactions, and block metadata
   - Transaction data: Input/output, gas usage, and status
   - Event logs: Smart contract events and their parameters
   - Token events: ERC20 transfers, ERC721 mints/transfers, ERC1155 batch operations
   - Internal transactions: Contract creation and self-destruct operations
   - Chain metrics: Network health, performance, and activity metrics
   - Subnet information: Validator sets, delegations, and subnet parameters

### Event Flow
1. **Data Ingestion**
   - WebSocket connection to Avalanche nodes for real-time updates
   - REST API polling for historical data and gap filling
   - Rate limiting and backoff strategies for API stability

2. **Event Processing**
   - Raw data transformation into standardized event formats
   - Event validation and enrichment
   - Topic-based routing to appropriate Kafka topics
   - Event persistence and indexing

3. **Event Distribution**
   - Real-time event streaming through WebSocket connections
   - Webhook notifications for subscribed events
   - Kafka consumer groups for downstream processing
   - Event replay capabilities for historical analysis

### Scalability Features
- Horizontal scaling through Kafka partitioning
- Configurable worker pools for parallel processing
- Rate limiting and backoff strategies
- Load balancing across multiple Avalanche nodes
- Caching mechanisms for frequently accessed data

### Monitoring and Observability
- Real-time metrics collection
- Performance monitoring
- Error tracking and alerting
- Health check endpoints
- Detailed logging for debugging and auditing

## Features
- Real-time blockchain data extraction
- Support for multiple data types:
  - Blocks
  - Transactions
  - Event Logs
  - Token Events (ERC20, ERC721, ERC1155)
  - Internal Transactions
  - Chain Metrics
  - Subnet Information
  - Validator Data
  - Delegator Information
  - Bridge Events
- WebSocket interface for real-time data streaming
- Webhook support for event notifications
- Rate-limited API integration
- Configurable worker pool for parallel processing
- Kafka integration for message streaming
- Docker support for easy deployment

## Prerequisites
- Go 1.19 or higher
- Docker and Docker Compose
- Kafka broker
- Access to Avalanche API endpoints

## Configuration
The application is configured through environment variables. Create a `.env` file in the root directory with the following variables:

```env
# API Configuration
API_URL=
API_KEY=
CHAIN_WS_URL=
API_RATE_LIMIT=5.0
API_RATE_BURST=10

# Kafka Configuration
KAFKA_BROKER=
KAFKA_TOPIC_CHAINS=
KAFKA_TOPIC_BLOCKS=
KAFKA_TOPIC_TRANSACTIONS=
KAFKA_TOPIC_LOGS=
KAFKA_TOPIC_ERC20=
KAFKA_TOPIC_ERC721=
KAFKA_TOPIC_ERC1155=
KAFKA_TOPIC_INTERNAL_TX=
KAFKA_TOPIC_METRICS=

# Webhook Configuration
WEBHOOK_PORT=
WEBHOOK_URL=
WEBHOOK_SECRET=

# WebSocket Configuration
WEBSOCKET_PORT=

# Worker Configuration
WORKERS=10

# AvaCloud Configuration
AVACLOUD_API_KEY=
AVACLOUD_API_BASE_URL=
CREATE_C_CHAIN_WEBHOOK=false

# Feature Flags
ENABLE_REALTIME=false
```

## Building and Running

### Using Docker
```bash
# Build and run using Docker Compose
docker-compose up --build
```

### Running Locally
```bash
# Build the application
go build -o avaxtractor cmd/main.go

# Run the application
./avaxtractor
```

## Project Structure
```
.
├── cmd/
│   └── main.go           # Application entry point
├── internal/
│   ├── api/             # API client implementations
│   ├── app/             # Application core
│   ├── avacloud/        # AvaCloud integration
│   ├── config/          # Configuration management
│   ├── event/           # Event handling
│   ├── extractor/       # Data extraction logic
│   ├── kafka/           # Kafka integration
│   ├── service/         # Service implementations
│   ├── types/           # Type definitions
│   ├── webhook/         # Webhook handling
│   └── websocket/       # WebSocket implementation
├── Dockerfile
├── docker-compose.yml
└── go.mod
```

## Development
1. Clone the repository
2. Install dependencies: `go mod download`
3. Set up your environment variables
4. Run the application: `go run cmd/main.go`