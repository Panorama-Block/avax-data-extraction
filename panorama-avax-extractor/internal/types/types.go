package types

type WebhookEvent struct {
    WebhookID string `json:"webhookId"`
    EventType string `json:"eventType"`
    MessageID string `json:"messageId"`
    Event struct {
        Transaction Transaction `json:"transaction"`
        Logs        []Log       `json:"logs"`
    } `json:"event"`
}

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

type AddressInfo struct {
    Name     string `json:"name"`
    Symbol   string `json:"symbol"`
    Decimals int    `json:"decimals"`
    LogoUri  string `json:"logoUri"`
    Address  string `json:"address"`
}

type MethodInfo struct {
    CallType   string `json:"callType"`
    MethodHash string `json:"methodHash"`
    MethodName string `json:"methodName"`
}

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

type InternalTransaction struct {
    From            string `json:"from"`
    To              string `json:"to"`
    InternalTxType  string `json:"internalTxType"`
    Value           string `json:"value"`
    GasUsed         string `json:"gasUsed"`
    GasLimit        string `json:"gasLimit"`
    TransactionHash string `json:"transactionHash"`
}

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

type Chain struct {
    ChainID         string   `json:"chainId"`
    Status          string   `json:"status"`
    ChainName       string   `json:"chainName"`
    Description     string   `json:"description"`
    PlatformChainId string   `json:"platformChainId"`
    SubnetId        string   `json:"subnetId"`
    VmId            string   `json:"vmId"`
    VmName          string   `json:"vmName"`
    ExplorerURL     string   `json:"explorerUrl"`
    RpcURL          string   `json:"rpcUrl"`
    WsURL           string   `json:"wsUrl"`
    IsTestnet       bool     `json:"isTestnet"`
    Private         bool     `json:"private"`
    ChainLogoUri    string   `json:"chainLogoUri"`
    EnabledFeatures []string `json:"enabledFeatures"`
    NetworkToken    struct {
        Name        string `json:"name"`
        Symbol      string `json:"symbol"`
        Decimals    int    `json:"decimals"`
        LogoUri     string `json:"logoUri"`
        Description string `json:"description"`
    } `json:"networkToken"`
}

type Validator struct {
    NodeID      string `json:"nodeId"`
    StartTime   int64  `json:"startTime"`
    EndTime     int64  `json:"endTime"`
    StakeAmount string `json:"stakeAmount"`
    Uptime      int64  `json:"uptime,omitempty"`
}

type Delegator struct {
    Delegator     string `json:"delegator"`
    NodeID        string `json:"nodeId"`
    StakeAmount   string `json:"stakeAmount"`
    StartTime     int64  `json:"startTime"`
    EndTime       int64  `json:"endTime"`
    RewardAddress string `json:"rewardAddress"`
}

type Subnet struct {
    SubnetID    string       `json:"subnetId"`
    Blockchains []Blockchain `json:"blockchains"`
}

type Blockchain struct {
    BlockchainID string `json:"blockchainId"`
    VMID         string `json:"vmId"`
    SubnetID     string `json:"subnetId"`
}

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

type StakingMetrics struct {
    ValidatorCount int     `json:"validatorCount"`
    DelegatorCount int     `json:"delegatorCount"`
    TotalStaked    float64 `json:"totalStaked"`
}

type TokenInfo struct {
    Address           string `json:"address"`
    Name              string `json:"name"`
    Symbol            string `json:"symbol"`
    Decimals          int    `json:"decimals"`
    ValueWithDecimals string `json:"valueWithDecimals"`
}
