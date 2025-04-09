package types

// Log represents a blockchain transaction log
type Log struct {
	Address          string  `json:"address"`
	Topic0           string  `json:"topic0"`
	Topic1           *string `json:"topic1,omitempty"`
	Topic2           *string `json:"topic2,omitempty"`
	Topic3           *string `json:"topic3,omitempty"`
	Data             string  `json:"data"`
	TransactionIndex int     `json:"transactionIndex"`
	LogIndex         int     `json:"logIndex"`
	Removed          bool    `json:"removed"`
}

// LogEvent is used for publishing log events
type LogEvent struct {
	Type string `json:"type"`
	Log  Log    `json:"log"`
}

// WebhookEvent represents an event received from a webhook
type WebhookEvent struct {
	WebhookID string `json:"webhookId"`
	EventType string `json:"eventType"`
	MessageID string `json:"messageId"`
	Event struct {
		Transaction Transaction `json:"transaction"`
		Logs        []Log       `json:"logs"`
	} `json:"event"`
} 