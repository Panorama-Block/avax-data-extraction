package types

// Block represents a blockchain block
type Block struct {
	ChainID                string `json:"chainId"`
	BlockNumber            string `json:"blockNumber"`
	BlockTimestamp         int64  `json:"blockTimestamp"`
	BlockHash              string `json:"blockHash"`
	TxCount                int    `json:"txCount"`
	BaseFee                string `json:"baseFee"`
	GasUsed                string `json:"gasUsed"`
	GasLimit               string `json:"gasLimit"`
	GasCost                string `json:"gasCost"`
	ParentHash             string `json:"parentHash"`
	FeesSpent              string `json:"feesSpent"`
	CumulativeTransactions string `json:"cumulativeTransactions"`
}

// BlockEvent is used for publishing block events
type BlockEvent struct {
	Type  string `json:"type"`
	Block Block  `json:"block"`
} 