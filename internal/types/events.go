package types

// Event constants for standardizing event types
const (
	// Block events
	EventBlockCreated = "block.created"
	EventBlockUpdated = "block.updated"
	
	// Chain events
	EventChainCreated = "chain.created"
	EventChainUpdated = "chain.updated"
	
	// Transaction events
	EventTransactionCreated = "transaction.created"
	EventTransactionUpdated = "transaction.updated"
	
	// Token transfer events
	EventERC20Transfer  = "erc20.transfer"
	EventERC721Transfer = "erc721.transfer"
	EventERC1155Transfer = "erc1155.transfer"
	
	// Log events
	EventLogCreated = "log.created"
	
	// Internal transaction events
	EventInternalTxCreated = "internal_tx.created"
	
	// Validator events
	EventValidatorCreated = "validator.created"
	EventValidatorUpdated = "validator.updated"
	
	// Delegator events
	EventDelegatorCreated = "delegator.created"
	EventDelegatorUpdated = "delegator.updated"
	
	// Subnet events
	EventSubnetCreated = "subnet.created"
	EventSubnetUpdated = "subnet.updated"
	
	// Blockchain events
	EventBlockchainCreated = "blockchain.created"
	EventBlockchainUpdated = "blockchain.updated"
	
	// Teleporter events
	EventTeleporterTxCreated = "teleporter.created"
	EventTeleporterTxUpdated = "teleporter.updated"
	
	// Token events
	EventTokenCreated = "token.created"
	EventTokenUpdated = "token.updated"
	
	// Metrics events - General
	EventMetricsUpdated = "metrics.updated"
	
	// Metrics events - Categories
	EventActivityMetricsUpdated   = "metrics.activity.updated"
	EventPerformanceMetricsUpdated = "metrics.performance.updated"
	EventGasMetricsUpdated        = "metrics.gas.updated"
	EventCumulativeMetricsUpdated = "metrics.cumulative.updated"
)

// Event represents a generic event in the system
type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
} 