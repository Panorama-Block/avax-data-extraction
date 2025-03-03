package types

// WebhookEvent representa o evento recebido pelo Webhook
type WebhookEvent struct {
	WebhookID string `json:"webhookId"`
	EventType string `json:"eventType"`
	MessageID string `json:"messageId"`
	Event     struct {
		Transaction Transaction `json:"transaction"`
		Logs        []Log       `json:"logs"`
	} `json:"event"`
}

// Transaction representa uma transação vinda do Webhook
type Transaction struct {
	BlockHash                 string             `json:"blockHash"`
	BlockNumber               string             `json:"blockNumber"`
	From                      string             `json:"from"`
	Gas                       string             `json:"gas"`
	GasPrice                  string             `json:"gasPrice"`
	MaxFeePerGas              string             `json:"maxFeePerGas"`
	MaxPriorityFeePerGas      string             `json:"maxPriorityFeePerGas"`
	TxHash                    string             `json:"txHash"`
	TxStatus                  string             `json:"txStatus"`
	Input                     string             `json:"input"`
	Nonce                     string             `json:"nonce"`
	To                        string             `json:"to"`
	TransactionIndex          int                `json:"transactionIndex"`
	Value                     string             `json:"value"`
	Type                      int                `json:"type"`
	ChainID                   string             `json:"chainId"`
	ReceiptCumulativeGasUsed  string             `json:"receiptCumulativeGasUsed"`
	ReceiptGasUsed            string             `json:"receiptGasUsed"`
	ReceiptEffectiveGasPrice  string             `json:"receiptEffectiveGasPrice"`
	ReceiptRoot               string             `json:"receiptRoot"`
	BlockTimestamp            int64              `json:"blockTimestamp"`
	ERC20Transfers            []ERC20Transfer    `json:"erc20Transfers"`
	ERC721Transfers           []ERC721Transfer   `json:"erc721Transfers"`
	ERC1155Transfers          []ERC1155Transfer  `json:"erc1155Transfers"`
	InternalTransactions      []InternalTransaction `json:"internalTransactions"`
}

// ERC20Transfer representa uma transferência ERC20
type ERC20Transfer struct {
	TransactionHash  string `json:"transactionHash"`
	Type            string `json:"type"`
	From            string `json:"from"`
	To              string `json:"to"`
	Value           string `json:"value"`
	BlockTimestamp  int64  `json:"blockTimestamp"`
	LogIndex        int    `json:"logIndex"`
	ERC20Token      struct {
		Address           string `json:"address"`
		Name              string `json:"name"`
		Symbol            string `json:"symbol"`
		Decimals         int    `json:"decimals"`
		ValueWithDecimals string `json:"valueWithDecimals"`
	} `json:"erc20Token"`
}

// ERC721Transfer representa uma transferência ERC721 (NFTs)
type ERC721Transfer struct {
	TransactionHash string `json:"transactionHash"`
	Type            string `json:"type"`
	From            string `json:"from"`
	To              string `json:"to"`
	TokenID         string `json:"tokenId"`
	BlockTimestamp  int64  `json:"blockTimestamp"`
	LogIndex        int    `json:"logIndex"`
	ERC721Token     struct {
		Address string `json:"address"`
		Name    string `json:"name"`
		Symbol  string `json:"symbol"`
	} `json:"erc721Token"`
}

// ERC1155Transfer representa uma transferência ERC1155 (NFTs multi-token)
type ERC1155Transfer struct {
	TransactionHash string `json:"transactionHash"`
	Type            string `json:"type"`
	From            string `json:"from"`
	To              string `json:"to"`
	TokenID         string `json:"tokenId"`
	Value           string `json:"value"`
	BlockTimestamp  int64  `json:"blockTimestamp"`
	LogIndex        int    `json:"logIndex"`
	ERC1155Token    struct {
		Address string `json:"address"`
		Name    string `json:"name"`
		Symbol  string `json:"symbol"`
	} `json:"erc1155Token"`
}

// InternalTransaction representa uma transação interna dentro da blockchain
type InternalTransaction struct {
	From            string `json:"from"`
	To              string `json:"to"`
	InternalTxType  string `json:"internalTxType"`
	Value           string `json:"value"`
	GasUsed         string `json:"gasUsed"`
	GasLimit        string `json:"gasLimit"`
	TransactionHash string `json:"transactionHash"`
}

// Log representa logs gerados por contratos inteligentes
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
