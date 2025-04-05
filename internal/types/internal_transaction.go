package types

// InternalTransaction represents an internal transaction
type InternalTransaction struct {
	From            string `json:"from"`
	To              string `json:"to"`
	InternalTxType  string `json:"internalTxType"`
	Value           string `json:"value"`
	GasUsed         string `json:"gasUsed"`
	GasLimit        string `json:"gasLimit"`
	TransactionHash string `json:"transactionHash"`
}

// InternalTransactionEvent is used for publishing internal transaction events
type InternalTransactionEvent struct {
	Type                string             `json:"type"`
	InternalTransaction InternalTransaction `json:"internalTransaction"`
} 