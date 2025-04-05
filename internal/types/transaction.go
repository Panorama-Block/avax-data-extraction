package types

// Transaction represents a blockchain transaction
type Transaction struct {
	BlockHash                string              `json:"blockHash"`
	BlockNumber              string              `json:"blockNumber"`
	From                     string              `json:"from"`
	Gas                      string              `json:"gas"`
	GasPrice                 string              `json:"gasPrice"`
	MaxFeePerGas             string              `json:"maxFeePerGas"`
	MaxPriorityFeePerGas     string              `json:"maxPriorityFeePerGas"`
	TxHash                   string              `json:"txHash"`
	TxStatus                 string              `json:"txStatus"`
	Input                    string              `json:"input"`
	Nonce                    string              `json:"nonce"`
	To                       string              `json:"to"`
	TransactionIndex         int                 `json:"transactionIndex"`
	Value                    string              `json:"value"`
	Type                     int                 `json:"type"`
	ChainID                  string              `json:"chainId"`
	ReceiptCumulativeGasUsed string              `json:"receiptCumulativeGasUsed"`
	ReceiptGasUsed           string              `json:"receiptGasUsed"`
	ReceiptEffectiveGasPrice string              `json:"receiptEffectiveGasPrice"`
	ReceiptRoot              string              `json:"receiptRoot"`
	BlockTimestamp           int64               `json:"blockTimestamp"`
	ERC20Transfers           []ERC20Transfer     `json:"erc20Transfers"`
	ERC721Transfers          []ERC721Transfer    `json:"erc721Transfers"`
	ERC1155Transfers         []ERC1155Transfer   `json:"erc1155Transfers"`
	InternalTransactions     []InternalTransaction `json:"internalTransactions"`
	Logs                     []Log               `json:"logs"`
}

// TransactionEvent is used for publishing transaction events
type TransactionEvent struct {
	Type        string      `json:"type"`
	Transaction Transaction `json:"transaction"`
}

// TransactionDetail represents detailed transaction information
type TransactionDetail struct {
	Transaction         Transaction          `json:"nativeTransaction"`
	ERC20Transfers      []ERC20Transfer      `json:"erc20Transfers,omitempty"`
	ERC721Transfers     []ERC721Transfer     `json:"erc721Transfers,omitempty"`
	ERC1155Transfers    []ERC1155Transfer    `json:"erc1155Transfers,omitempty"`
	InternalTransactions []InternalTransaction `json:"internalTransactions,omitempty"`
}

// Address represents an on-chain address with additional information
type Address struct {
	Address  string `json:"address"`
	Name     string `json:"name,omitempty"`
	Symbol   string `json:"symbol,omitempty"`
	LogoURI  string `json:"logoUri,omitempty"`
	Decimals int    `json:"decimals,omitempty"`
}

// MethodInfo represents method call information
type MethodInfo struct {
	CallType   string `json:"callType"`
	MethodHash string `json:"methodHash"`
	MethodName string `json:"methodName"`
} 