package api

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// DataAPI handles data-related API calls to AvaCloud
type DataAPI struct {
	client *Client
}

// NewDataAPI creates a new Data API client
func NewDataAPI(client *Client) *DataAPI {
	return &DataAPI{client: client}
}

// ========== Chain Endpoints ==========

// Chain represents an EVM-compatible chain
type Chain struct {
	ChainID           string  `json:"chainId"`
	Name              string  `json:"name"`
	IsTestnet         bool    `json:"isTestnet"`
	Network           string  `json:"network"`
	NativeCurrency    Token   `json:"nativeCurrency"`
	RpcURL            string  `json:"rpcUrl,omitempty"`
	ExplorerURL       string  `json:"explorerUrl,omitempty"`
	SubnetID          string  `json:"subnetId,omitempty"`
	IsSubnet          bool    `json:"isSubnet"`
	IsL1              bool    `json:"isL1,omitempty"`
}

// ChainsResponse is the response from the chains endpoint
type ChainsResponse struct {
	Chains []Chain `json:"chains"`
	PaginationResponse
}

// GetChains retrieves all EVM-compatible chains
func (d *DataAPI) GetChains(network string, pagination *PaginationParams) (*ChainsResponse, error) {
	params := url.Values{}
	if network != "" {
		params.Add("network", network)
	}
	
	if pagination != nil {
		params = MergeQueryParams(params, pagination.ToQueryParams())
	}
	
	respBody, err := d.client.SendRequest("GET", "/v1/chains", params, nil)
	if err != nil {
		return nil, err
	}
	
	var response ChainsResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// GetChain retrieves details for a specific chain
func (d *DataAPI) GetChain(chainID string) (*Chain, error) {
	respBody, err := d.client.SendRequest("GET", fmt.Sprintf("/v1/chains/%s", chainID), nil, nil)
	if err != nil {
		return nil, err
	}
	
	var response Chain
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// ========== Block Endpoints ==========

// Block represents a block on an EVM-compatible chain
type Block struct {
	BlockNumber           string `json:"blockNumber"`
	BlockHash             string `json:"blockHash"`
	BlockTimestamp        int64  `json:"blockTimestamp"`
	TxCount               int    `json:"txCount"`
	BaseFee               string `json:"baseFee,omitempty"`
	GasUsed               string `json:"gasUsed,omitempty"`
	GasLimit              string `json:"gasLimit,omitempty"`
	FeesSpent             string `json:"feesSpent,omitempty"`
	CumulativeTransactions string `json:"cumulativeTransactions,omitempty"`
	ParentHash            string `json:"parentHash,omitempty"`
	Miner                 string `json:"miner,omitempty"`
}

// BlocksResponse is the response from the blocks endpoint
type BlocksResponse struct {
	Blocks []Block `json:"blocks"`
	PaginationResponse
}

// GetBlocks retrieves blocks for a specific chain
func (d *DataAPI) GetBlocks(chainID string, pagination *PaginationParams) (*BlocksResponse, error) {
	params := url.Values{}
	if pagination != nil {
		params = pagination.ToQueryParams()
	}
	
	respBody, err := d.client.SendRequest("GET", fmt.Sprintf("/v1/chains/%s/blocks", chainID), params, nil)
	if err != nil {
		return nil, err
	}
	
	var response BlocksResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// GetBlock retrieves a specific block by number or hash
func (d *DataAPI) GetBlock(chainID, blockID string) (*Block, error) {
	respBody, err := d.client.SendRequest("GET", fmt.Sprintf("/v1/chains/%s/blocks/%s", chainID, blockID), nil, nil)
	if err != nil {
		return nil, err
	}
	
	var response Block
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// ========== Transaction Endpoints ==========

// Transaction represents a transaction on an EVM-compatible chain
type Transaction struct {
	BlockHash       string `json:"blockHash"`
	BlockNumber     string `json:"blockNumber"`
	BlockTimestamp  int64  `json:"blockTimestamp"`
	From            Address `json:"from"`
	To              Address `json:"to"`
	Gas             string `json:"gas"`
	GasPrice        string `json:"gasPrice"`
	GasUsed         string `json:"gasUsed,omitempty"`
	Hash            string `json:"hash"`
	Input           string `json:"input,omitempty"`
	Nonce           string `json:"nonce"`
	Value           string `json:"value"`
	TxStatus        string `json:"txStatus"`
	TxIndex         string `json:"txIndex,omitempty"`
	TxType          string `json:"txType,omitempty"`
	MaxFeePerGas    string `json:"maxFeePerGas,omitempty"`
	MaxPriorityFee  string `json:"maxPriorityFee,omitempty"`
}

// Address represents an address with additional metadata
type Address struct {
	Address  string `json:"address"`
	Name     string `json:"name,omitempty"`
	Symbol   string `json:"symbol,omitempty"`
	LogoURI  string `json:"logoUri,omitempty"`
	Decimals int    `json:"decimals,omitempty"`
}

// Token represents a token with its metadata
type Token struct {
	Address  string      `json:"address,omitempty"`
	Name     string      `json:"name"`
	Symbol   string      `json:"symbol"`
	Decimals int         `json:"decimals"`
	LogoURI  string      `json:"logoUri,omitempty"`
	Price    *TokenPrice `json:"price,omitempty"`
}

// TokenPrice represents a token's price information
type TokenPrice struct {
	Value        float64 `json:"value"`
	CurrencyCode string  `json:"currencyCode"`
}

// ERC20Transfer represents an ERC-20 token transfer
type ERC20Transfer struct {
	BlockHash      string `json:"blockHash"`
	BlockNumber    string `json:"blockNumber"`
	BlockTimestamp int64  `json:"blockTimestamp"`
	From           Address `json:"from"`
	To             Address `json:"to"`
	Value          string `json:"value"`
	ERC20Token     Token  `json:"erc20Token"`
	LogIndex       string `json:"logIndex,omitempty"`
}

// ERC721Transfer represents an ERC-721 token transfer (NFT)
type ERC721Transfer struct {
	BlockHash       string `json:"blockHash"`
	BlockNumber     string `json:"blockNumber"`
	BlockTimestamp  int64  `json:"blockTimestamp"`
	From            Address `json:"from"`
	To              Address `json:"to"`
	TokenID         string `json:"tokenId"`
	ERC721Token     Token   `json:"erc721Token"`
	LogIndex        string  `json:"logIndex,omitempty"`
}

// ERC1155Transfer represents an ERC-1155 token transfer
type ERC1155Transfer struct {
	BlockHash       string  `json:"blockHash"`
	BlockNumber     string  `json:"blockNumber"`
	BlockTimestamp  int64   `json:"blockTimestamp"`
	From            Address `json:"from"`
	To              Address `json:"to"`
	TokenID         string  `json:"tokenId"`
	Value           string  `json:"value"`
	ERC1155Token    Token   `json:"erc1155Token"`
	LogIndex        string  `json:"logIndex,omitempty"`
}

// InternalTransaction represents an internal transaction
type InternalTransaction struct {
	BlockHash       string  `json:"blockHash,omitempty"`
	BlockNumber     string  `json:"blockNumber,omitempty"`
	BlockTimestamp  int64   `json:"blockTimestamp,omitempty"`
	From            string  `json:"from"`
	To              string  `json:"to"`
	Value           string  `json:"value"`
	Gas             string  `json:"gas,omitempty"`
	GasUsed         string  `json:"gasUsed,omitempty"`
	Input           string  `json:"input,omitempty"`
	Output          string  `json:"output,omitempty"`
	Type            string  `json:"internalTxType,omitempty"`
	TraceAddress    []int   `json:"traceAddress,omitempty"`
	CallType        string  `json:"callType,omitempty"`
}

// TransactionDetail includes the transaction and its associated transfers
type TransactionDetail struct {
	Transaction         Transaction          `json:"nativeTransaction"`
	ERC20Transfers      []ERC20Transfer      `json:"erc20Transfers,omitempty"`
	ERC721Transfers     []ERC721Transfer     `json:"erc721Transfers,omitempty"`
	ERC1155Transfers    []ERC1155Transfer    `json:"erc1155Transfers,omitempty"`
	InternalTransactions []InternalTransaction `json:"internalTransactions,omitempty"`
}

// TransactionsResponse is the response from the transactions endpoint
type TransactionsResponse struct {
	Transactions []TransactionDetail `json:"transactions"`
	PaginationResponse
}

// GetTransactions retrieves transactions for a specific chain
func (d *DataAPI) GetTransactions(chainID string, status string, pagination *PaginationParams) (*TransactionsResponse, error) {
	params := url.Values{}
	if status != "" {
		params.Add("status", status)
	}
	
	if pagination != nil {
		params = MergeQueryParams(params, pagination.ToQueryParams())
	}
	
	respBody, err := d.client.SendRequest("GET", fmt.Sprintf("/v1/chains/%s/transactions", chainID), params, nil)
	if err != nil {
		return nil, err
	}
	
	var response TransactionsResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// GetTransaction retrieves a specific transaction by hash
func (d *DataAPI) GetTransaction(chainID, txHash string) (*TransactionDetail, error) {
	respBody, err := d.client.SendRequest("GET", fmt.Sprintf("/v1/chains/%s/transactions/%s", chainID, txHash), nil, nil)
	if err != nil {
		return nil, err
	}
	
	var response TransactionDetail
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// GetAddressTransactions retrieves transactions for a specific address
func (d *DataAPI) GetAddressTransactions(chainID, address string, blockRange *BlockRange, pagination *PaginationParams) (*TransactionsResponse, error) {
	var params url.Values
	if blockRange != nil {
		params = blockRange.ToQueryParams()
	} else {
		params = url.Values{}
	}
	
	if pagination != nil {
		params = MergeQueryParams(params, pagination.ToQueryParams())
	}
	
	respBody, err := d.client.SendRequest("GET", fmt.Sprintf("/v1/chains/%s/addresses/%s/transactions", chainID, address), params, nil)
	if err != nil {
		return nil, err
	}
	
	var response TransactionsResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// ========== Token Transfer Endpoints ==========

// ERC20TransfersResponse is the response for ERC-20 transfers
type ERC20TransfersResponse struct {
	Transfers []ERC20Transfer `json:"transfers"`
	PaginationResponse
}

// GetERC20Transfers retrieves ERC-20 transfers for an address
func (d *DataAPI) GetERC20Transfers(chainID, address string, blockRange *BlockRange, pagination *PaginationParams) (*ERC20TransfersResponse, error) {
	var params url.Values
	if blockRange != nil {
		params = blockRange.ToQueryParams()
	} else {
		params = url.Values{}
	}
	
	if pagination != nil {
		params = MergeQueryParams(params, pagination.ToQueryParams())
	}
	
	respBody, err := d.client.SendRequest("GET", fmt.Sprintf("/v1/chains/%s/addresses/%s/transactions:listErc20", chainID, address), params, nil)
	if err != nil {
		return nil, err
	}
	
	var response ERC20TransfersResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// ERC721TransfersResponse is the response for ERC-721 transfers
type ERC721TransfersResponse struct {
	Transfers []ERC721Transfer `json:"transfers"`
	PaginationResponse
}

// GetERC721Transfers retrieves ERC-721 transfers for an address
func (d *DataAPI) GetERC721Transfers(chainID, address string, blockRange *BlockRange, pagination *PaginationParams) (*ERC721TransfersResponse, error) {
	var params url.Values
	if blockRange != nil {
		params = blockRange.ToQueryParams()
	} else {
		params = url.Values{}
	}
	
	if pagination != nil {
		params = MergeQueryParams(params, pagination.ToQueryParams())
	}
	
	respBody, err := d.client.SendRequest("GET", fmt.Sprintf("/v1/chains/%s/addresses/%s/transactions:listErc721", chainID, address), params, nil)
	if err != nil {
		return nil, err
	}
	
	var response ERC721TransfersResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// ========== Balance Endpoints ==========

// NativeBalanceResponse is the response for native token balance
type NativeBalanceResponse struct {
	Balance string `json:"balance"`
}

// GetNativeBalance retrieves the native token balance for an address
func (d *DataAPI) GetNativeBalance(chainID, address string, blockNumber string) (*NativeBalanceResponse, error) {
	params := url.Values{}
	if blockNumber != "" {
		params.Add("blockNumber", blockNumber)
	}
	
	respBody, err := d.client.SendRequest("GET", fmt.Sprintf("/v1/chains/%s/addresses/%s/balances:getNative", chainID, address), params, nil)
	if err != nil {
		return nil, err
	}
	
	var response NativeBalanceResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// ERC20Balance represents an ERC-20 token balance
type ERC20Balance struct {
	Token      Token  `json:"token"`
	Balance    string `json:"balance"`
	BalanceUSD string `json:"balanceUSD,omitempty"`
}

// ERC20BalancesResponse is the response for ERC-20 balances
type ERC20BalancesResponse struct {
	Balances []ERC20Balance `json:"erc20Balances"`
	PaginationResponse
}

// GetERC20Balances retrieves ERC-20 token balances for an address
func (d *DataAPI) GetERC20Balances(chainID, address string, blockNumber string, contractAddresses []string, pagination *PaginationParams) (*ERC20BalancesResponse, error) {
	params := url.Values{}
	if blockNumber != "" {
		params.Add("blockNumber", blockNumber)
	}
	
	if len(contractAddresses) > 0 {
		for _, contractAddress := range contractAddresses {
			params.Add("contractAddresses", contractAddress)
		}
	}
	
	if pagination != nil {
		params = MergeQueryParams(params, pagination.ToQueryParams())
	}
	
	respBody, err := d.client.SendRequest("GET", fmt.Sprintf("/v1/chains/%s/addresses/%s/balances:listErc20", chainID, address), params, nil)
	if err != nil {
		return nil, err
	}
	
	var response ERC20BalancesResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// ========== Avalanche Primary Network Endpoints ==========

// Network represents an Avalanche network (mainnet, fuji, etc.)
type Network struct {
	Name string `json:"name"`
}

// Validator represents a validator on the P-Chain
type Validator struct {
	NodeID            string   `json:"nodeId"`
	StartTime         int64    `json:"startTime"`
	EndTime           int64    `json:"endTime"`
	StakeAmount       string   `json:"stakeAmount"`
	PotentialReward   string   `json:"potentialReward"`
	DelegationFee     float64  `json:"delegationFee"`
	Uptime            float64  `json:"uptime"`
	Connected         bool     `json:"connected"`
	StakeOwnerAddress string   `json:"stakeOwnerAddress,omitempty"`
	SubnetIDs         []string `json:"subnetIds,omitempty"`
}

// ValidatorsResponse is the response for the validators endpoint
type ValidatorsResponse struct {
	Validators []Validator `json:"validators"`
	PaginationResponse
}

// GetValidators retrieves validators for a network
func (d *DataAPI) GetValidators(network string, nodeIDs []string, status string, pagination *PaginationParams) (*ValidatorsResponse, error) {
	params := url.Values{}
	if len(nodeIDs) > 0 {
		for _, nodeID := range nodeIDs {
			params.Add("nodeIds", nodeID)
		}
	}
	
	if status != "" {
		params.Add("status", status)
	}
	
	if pagination != nil {
		params = MergeQueryParams(params, pagination.ToQueryParams())
	}
	
	respBody, err := d.client.SendRequest("GET", fmt.Sprintf("/v1/networks/%s/validators", network), params, nil)
	if err != nil {
		return nil, err
	}
	
	var response ValidatorsResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// Delegator represents a delegator on the P-Chain
type Delegator struct {
	NodeID            string `json:"nodeId"`
	StartTime         int64  `json:"startTime"`
	EndTime           int64  `json:"endTime"`
	StakeAmount       string `json:"stakeAmount"`
	PotentialReward   string `json:"potentialReward"`
	RewardOwnerAddress string `json:"rewardOwnerAddress,omitempty"`
	StakeOwnerAddress string `json:"stakeOwnerAddress,omitempty"`
}

// DelegatorsResponse is the response for the delegators endpoint
type DelegatorsResponse struct {
	Delegators []Delegator `json:"delegators"`
	PaginationResponse
}

// GetDelegators retrieves delegators for a network
func (d *DataAPI) GetDelegators(network string, pagination *PaginationParams) (*DelegatorsResponse, error) {
	params := url.Values{}
	
	if pagination != nil {
		params = MergeQueryParams(params, pagination.ToQueryParams())
	}
	
	respBody, err := d.client.SendRequest("GET", fmt.Sprintf("/v1/networks/%s/delegators", network), params, nil)
	if err != nil {
		return nil, err
	}
	
	var response DelegatorsResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// Subnet represents a subnet on the P-Chain
type Subnet struct {
	ID          string   `json:"id"`
	ControlKeys []string `json:"controlKeys"`
	Threshold   int      `json:"threshold"`
	IsL1        bool     `json:"isL1"`
}

// SubnetsResponse is the response for the subnets endpoint
type SubnetsResponse struct {
	Subnets []Subnet `json:"subnets"`
	PaginationResponse
}

// GetSubnets retrieves subnets for a network
func (d *DataAPI) GetSubnets(network string, pagination *PaginationParams) (*SubnetsResponse, error) {
	params := url.Values{}
	
	if pagination != nil {
		params = MergeQueryParams(params, pagination.ToQueryParams())
	}
	
	respBody, err := d.client.SendRequest("GET", fmt.Sprintf("/v1/networks/%s/subnets", network), params, nil)
	if err != nil {
		return nil, err
	}
	
	var response SubnetsResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// Blockchain represents a blockchain on the P-Chain
type Blockchain struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	SubnetID string `json:"subnetId"`
	VMName   string `json:"vmName"`
}

// BlockchainsResponse is the response for the blockchains endpoint
type BlockchainsResponse struct {
	Blockchains []Blockchain `json:"blockchains"`
	PaginationResponse
}

// GetBlockchains retrieves blockchains for a network
func (d *DataAPI) GetBlockchains(network string, pagination *PaginationParams) (*BlockchainsResponse, error) {
	params := url.Values{}
	
	if pagination != nil {
		params = MergeQueryParams(params, pagination.ToQueryParams())
	}
	
	respBody, err := d.client.SendRequest("GET", fmt.Sprintf("/v1/networks/%s/blockchains", network), params, nil)
	if err != nil {
		return nil, err
	}
	
	var response BlockchainsResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// Reward represents a staking reward on the P-Chain
type Reward struct {
	NodeID        string `json:"nodeId"`
	RewardAmount  string `json:"rewardAmount"`
	Timestamp     int64  `json:"timestamp"`
	RewardAddress string `json:"rewardAddress"`
}

// RewardsResponse is the response for the rewards endpoint
type RewardsResponse struct {
	Rewards []Reward `json:"rewards"`
	PaginationResponse
}

// GetRewards retrieves staking rewards for a network
func (d *DataAPI) GetRewards(network string, addresses []string, pagination *PaginationParams) (*RewardsResponse, error) {
	params := url.Values{}
	
	if len(addresses) > 0 {
		for _, address := range addresses {
			params.Add("addresses", address)
		}
	}
	
	if pagination != nil {
		params = MergeQueryParams(params, pagination.ToQueryParams())
	}
	
	respBody, err := d.client.SendRequest("GET", fmt.Sprintf("/v1/networks/%s/rewards", network), params, nil)
	if err != nil {
		return nil, err
	}
	
	var response RewardsResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// PendingRewardsResponse is the response for the pending rewards endpoint
type PendingRewardsResponse struct {
	PendingRewards []struct {
		NodeID       string `json:"nodeId"`
		Amount       string `json:"amount"`
		EndTime      int64  `json:"endTime"`
		StakeOwner   string `json:"stakeOwner,omitempty"`
		RewardOwner  string `json:"rewardOwner,omitempty"`
	} `json:"pendingRewards"`
	PaginationResponse
}

// GetPendingRewards retrieves pending staking rewards for a network
func (d *DataAPI) GetPendingRewards(network string, addresses []string, pagination *PaginationParams) (*PendingRewardsResponse, error) {
	params := url.Values{}
	
	if len(addresses) > 0 {
		for _, address := range addresses {
			params.Add("addresses", address)
		}
	}
	
	if pagination != nil {
		params = MergeQueryParams(params, pagination.ToQueryParams())
	}
	
	respBody, err := d.client.SendRequest("GET", fmt.Sprintf("/v1/networks/%s/rewards:listPending", network), params, nil)
	if err != nil {
		return nil, err
	}
	
	var response PendingRewardsResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// UTXOsResponse is the response for the UTXOs endpoint
type UTXOsResponse struct {
	UTXOs []struct {
		UTXOID      string `json:"utxoId"`
		Amount      string `json:"amount"`
		AssetID     string `json:"assetId"`
		OutputType  int    `json:"outputType"`
		Addresses   []string `json:"addresses"`
		Locktime    int64  `json:"locktime,omitempty"`
		Threshold   int    `json:"threshold,omitempty"`
	} `json:"utxos"`
	PaginationResponse
}

// GetUTXOs retrieves UTXOs for addresses on a blockchain
func (d *DataAPI) GetUTXOs(network, blockchainID string, addresses []string, pagination *PaginationParams) (*UTXOsResponse, error) {
	params := url.Values{}
	
	if len(addresses) > 0 {
		for _, address := range addresses {
			params.Add("addresses", address)
		}
	}
	
	if pagination != nil {
		params = MergeQueryParams(params, pagination.ToQueryParams())
	}
	
	respBody, err := d.client.SendRequest("GET", fmt.Sprintf("/v1/networks/%s/blockchains/%s/utxos", network, blockchainID), params, nil)
	if err != nil {
		return nil, err
	}
	
	var response UTXOsResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// PChainTransaction represents a transaction on the P-Chain
type PChainTransaction struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	BlockID         string `json:"blockId"`
	Timestamp       int64  `json:"timestamp"`
	Inputs          []interface{} `json:"inputs"`  // Structure varies by transaction type
	Outputs         []interface{} `json:"outputs"` // Structure varies by transaction type
	Memo            string `json:"memo,omitempty"`
	NetworkID       int    `json:"networkId"`
	BlockchainID    string `json:"blockchainId"`
}

// PChainTransactionsResponse is the response for P-Chain transactions
type PChainTransactionsResponse struct {
	Transactions []PChainTransaction `json:"transactions"`
	PaginationResponse
}

// GetPChainTransactions retrieves transactions on the P-Chain
func (d *DataAPI) GetPChainTransactions(network, blockchainID string, timeRange *TimeRange, address string, txTypes []string, pagination *PaginationParams) (*PChainTransactionsResponse, error) {
	params := url.Values{}
	
	if timeRange != nil {
		params = MergeQueryParams(params, timeRange.ToQueryParams())
	}
	
	if address != "" {
		params.Add("address", address)
	}
	
	if len(txTypes) > 0 {
		for _, txType := range txTypes {
			params.Add("txTypes", txType)
		}
	}
	
	if pagination != nil {
		params = MergeQueryParams(params, pagination.ToQueryParams())
	}
	
	respBody, err := d.client.SendRequest("GET", fmt.Sprintf("/v1/networks/%s/blockchains/%s/transactions", network, blockchainID), params, nil)
	if err != nil {
		return nil, err
	}
	
	var response PChainTransactionsResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// Asset represents an asset on the X-Chain
type Asset struct {
	ID            string `json:"id"`
	Name          string `json:"name,omitempty"`
	Symbol        string `json:"symbol,omitempty"`
	Denomination  int    `json:"denomination"`
	InitialStates map[string][]interface{} `json:"initialStates,omitempty"`
}

// GetAsset retrieves information about an asset on the X-Chain
func (d *DataAPI) GetAsset(network, blockchainID, assetID string) (*Asset, error) {
	respBody, err := d.client.SendRequest("GET", fmt.Sprintf("/v1/networks/%s/blockchains/%s/assets/%s", network, blockchainID, assetID), nil, nil)
	if err != nil {
		return nil, err
	}
	
	var response Asset
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}