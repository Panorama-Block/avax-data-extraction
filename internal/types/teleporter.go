package types

// TeleporterTx represents a cross-chain teleporter transaction
type TeleporterTx struct {
	ID          string `json:"id"`
	SourceChain string `json:"sourceChain"`
	DestChain   string `json:"destChain"`
	Asset       string `json:"asset"`
	Amount      string `json:"amount"`
	Sender      string `json:"sender"`
	Receiver    string `json:"receiver"`
	Status      string `json:"status"`
}

// TeleporterEvent is used for publishing teleporter events
type TeleporterEvent struct {
	Type      string       `json:"type"`
	Teleporter TeleporterTx `json:"teleporter"`
} 