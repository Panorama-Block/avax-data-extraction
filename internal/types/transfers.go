package types

// ERC20Transfer represents an ERC20 token transfer
type ERC20Transfer struct {
	TransactionHash string `json:"transactionHash"`
	Type            string `json:"type"`
	From            string `json:"from"`
	To              string `json:"to"`
	Value           string `json:"value"`
	BlockTimestamp  int64  `json:"blockTimestamp"`
	LogIndex        int    `json:"logIndex"`
	ERC20Token struct {
		Address           string `json:"address"`
		Name              string `json:"name"`
		Symbol            string `json:"symbol"`
		Decimals          int    `json:"decimals"`
		ValueWithDecimals string `json:"valueWithDecimals"`
	} `json:"erc20Token"`
}

// ERC20Event is used for publishing ERC20 transfer events
type ERC20Event struct {
	Type    string       `json:"type"`
	Transfer ERC20Transfer `json:"transfer"`
}

// ERC721Transfer represents an ERC721 token transfer
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

// ERC721Event is used for publishing ERC721 transfer events
type ERC721Event struct {
	Type    string        `json:"type"`
	Transfer ERC721Transfer `json:"transfer"`
}

// ERC1155Transfer represents an ERC1155 token transfer
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

// ERC1155Event is used for publishing ERC1155 transfer events
type ERC1155Event struct {
	Type    string         `json:"type"`
	Transfer ERC1155Transfer `json:"transfer"`
} 