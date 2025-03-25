package api

import (
    "encoding/json"
    "fmt"
    "log"
    "net/url"
    "strconv"

    "github.com/Panorama-Block/avax/internal/types"
)

// DataAPI gerencia as chamadas relacionadas a dados para a API AvaCloud
type DataAPI struct {
    client *Client
}

// NewDataAPI cria um novo cliente DataAPI
func NewDataAPI(client *Client) *DataAPI {
    return &DataAPI{client: client}
}

func (c *DataAPI) GetChains() ([]types.Chain, error) {
    body, err := c.client.makeRequest("/chains")
    if err != nil {
        log.Println("Erro ao buscar chains:", err)
        return nil, err
    }
    var response struct {
        Chains []types.Chain `json:"chains"`
    }
    if err = json.Unmarshal(body, &response); err != nil {
        log.Println("Erro decodificando /chains:", err)
        return nil, err
    }
    return response.Chains, nil
}

func (c *DataAPI) GetChainByID(chainID string) (*types.Chain, error) {
    body, err := c.client.makeRequest("/chains/" + chainID)
    if err != nil {
        log.Printf("Erro ao buscar chain %s: %v", chainID, err)
        return nil, err
    }
    var chain types.Chain
    if err = json.Unmarshal(body, &chain); err != nil {
        log.Printf("Erro decodificando chain %s: %v", chainID, err)
        return nil, err
    }
    return &chain, nil
}

func (c *DataAPI) GetBlocks(chainID string, pageSize int, pageToken string) ([]types.Block, string, error) {
    params := url.Values{}
    params.Set("chainId", chainID)
    if pageSize > 0 {
        params.Set("pageSize", strconv.Itoa(pageSize))
    }
    if pageToken != "" {
        params.Set("pageToken", pageToken)
    }
    respBody, err := c.client.makeRequest("/blocks?" + params.Encode())
    if err != nil {
        log.Printf("Erro ao listar blocos chain %s: %v", chainID, err)
        return nil, "", err
    }
    var result struct {
        Blocks        []types.Block `json:"blocks"`
        NextPageToken string        `json:"nextPageToken"`
    }
    if err := json.Unmarshal(respBody, &result); err != nil {
        log.Printf("Erro decodificando blocos chain %s: %v", chainID, err)
        return nil, "", err
    }
    return result.Blocks, result.NextPageToken, nil
}

func (c *DataAPI) GetBlockByNumberOrHash(chainID, blockIdentifier string) (*types.Block, error) {
    endpoint := fmt.Sprintf("/chains/%s/blocks/%s", chainID, blockIdentifier)
    respBody, err := c.client.makeRequest(endpoint)
    if err != nil {
        log.Printf("Erro ao buscar bloco %s (chain %s): %v", blockIdentifier, chainID, err)
        return nil, err
    }
    var block types.Block
    if err := json.Unmarshal(respBody, &block); err != nil {
        return nil, err
    }
    return &block, nil
}

func (c *DataAPI) GetValidators(network string) ([]types.Validator, error) {
    endpoint := fmt.Sprintf("/networks/%s/validators", network)
    body, err := c.client.makeRequest(endpoint)
    if err != nil {
        log.Printf("Erro ao buscar validadores da rede %s: %v", network, err)
        return nil, err
    }
    var result struct {
        Validators []types.Validator `json:"validators"`
    }
    if err := json.Unmarshal(body, &result); err != nil {
        log.Printf("Erro decodificando validadores: %v", err)
        return nil, err
    }
    return result.Validators, nil
}

func (c *DataAPI) GetValidatorByNodeID(network, nodeID string) (*types.Validator, error) {
    endpoint := fmt.Sprintf("/networks/%s/validators/%s", network, nodeID)
    body, err := c.client.makeRequest(endpoint)
    if err != nil {
        log.Printf("Erro ao buscar validador %s: %v", nodeID, err)
        return nil, err
    }
    var wrapper struct {
        Validator types.Validator `json:"validator"`
    }
    if err := json.Unmarshal(body, &wrapper); err != nil {
        log.Printf("Erro decodificando validador %s: %v", nodeID, err)
        return nil, err
    }
    return &wrapper.Validator, nil
}

func (c *DataAPI) GetDelegators(network string) ([]types.Delegator, error) {
    endpoint := fmt.Sprintf("/networks/%s/delegators", network)
    body, err := c.client.makeRequest(endpoint)
    if err != nil {
        log.Printf("Erro ao buscar delegators da rede %s: %v", network, err)
        return nil, err
    }
    var result struct {
        Delegations []types.Delegator `json:"delegations"`
    }
    if err := json.Unmarshal(body, &result); err != nil {
        log.Printf("Erro decodificando delegators: %v", err)
        return nil, err
    }
    return result.Delegations, nil
}

func (c *DataAPI) GetSubnets(network string) ([]types.Subnet, error) {
    endpoint := fmt.Sprintf("/networks/%s/subnets", network)
    body, err := c.client.makeRequest(endpoint)
    if err != nil {
        log.Printf("Erro ao buscar subnets da rede %s: %v", network, err)
        return nil, err
    }
    var result struct {
        Subnets []types.Subnet `json:"subnets"`
    }
    if err := json.Unmarshal(body, &result); err != nil {
        log.Printf("Erro decodificando subnets: %v", err)
        return nil, err
    }
    return result.Subnets, nil
}

func (c *DataAPI) GetSubnetByID(network, subnetID string) (*types.Subnet, error) {
    endpoint := fmt.Sprintf("/networks/%s/subnets/%s", network, subnetID)
    body, err := c.client.makeRequest(endpoint)
    if err != nil {
        log.Printf("Erro ao buscar subnet %s: %v", subnetID, err)
        return nil, err
    }
    var wrapper struct {
        Subnet types.Subnet `json:"subnet"`
    }
    if err := json.Unmarshal(body, &wrapper); err != nil {
        log.Printf("Erro decodificando subnet %s: %v", subnetID, err)
        return nil, err
    }
    return &wrapper.Subnet, nil
}

func (c *DataAPI) GetBlockchains(network string) ([]types.Blockchain, error) {
    endpoint := fmt.Sprintf("/networks/%s/blockchains", network)
    body, err := c.client.makeRequest(endpoint)
    if err != nil {
        log.Printf("Erro ao buscar blockchains da rede %s: %v", network, err)
        return nil, err
    }
    var result struct {
        Blockchains []types.Blockchain `json:"blockchains"`
    }
    if err := json.Unmarshal(body, &result); err != nil {
        log.Printf("Erro decodificando blockchains: %v", err)
        return nil, err
    }
    return result.Blockchains, nil
}

func (c *DataAPI) GetTeleporterTxs(sourceChain, destChain string) ([]types.TeleporterTx, error) {
    endpoint := fmt.Sprintf("/teleporter/transactions?sourceChain=%s&destChain=%s", sourceChain, destChain)
    body, err := c.client.makeRequest(endpoint)
    if err != nil {
        log.Printf("Erro ao buscar teleporter TXs: %v", err)
        return nil, err
    }
    var result struct {
        Transactions []types.TeleporterTx `json:"transactions"`
    }
    if err := json.Unmarshal(body, &result); err != nil {
        log.Printf("Erro decodificando teleporter txs: %v", err)
        return nil, err
    }
    return result.Transactions, nil
}

func (c *DataAPI) GetTokenDetails(tokenAddress string) (*types.TokenInfo, error) {
    endpoint := fmt.Sprintf("/assets/%s", tokenAddress)
    body, err := c.client.makeRequest(endpoint)
    if err != nil {
        log.Printf("Erro ao buscar detalhes do token %s: %v", tokenAddress, err)
        return nil, err
    }
    var tokenDetails types.TokenInfo
    if err := json.Unmarshal(body, &tokenDetails); err != nil {
        log.Printf("Erro decodificando token %s: %v", tokenAddress, err)
        return nil, err
    }
    return &tokenDetails, nil
}

// GetTransaction obtém detalhes de uma transação específica
func (c *DataAPI) GetTransaction(chainID, txHash string) (*TransactionDetail, error) {
    endpoint := fmt.Sprintf("/v1/chains/%s/transactions/%s", chainID, txHash)
    respBody, err := c.client.SendRequest("GET", endpoint, nil, nil)
    if err != nil {
        return nil, err
    }
    
    var response TransactionDetail
    if err := json.Unmarshal(respBody, &response); err != nil {
        return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
    }
    
    return &response, nil
}

// TransactionDetail inclui a transação e suas transferências associadas
type TransactionDetail struct {
    Transaction         Transaction          `json:"nativeTransaction"`
    ERC20Transfers      []ERC20Transfer      `json:"erc20Transfers,omitempty"`
    ERC721Transfers     []ERC721Transfer     `json:"erc721Transfers,omitempty"`
    ERC1155Transfers    []ERC1155Transfer    `json:"erc1155Transfers,omitempty"`
    InternalTransactions []InternalTransaction `json:"internalTransactions,omitempty"`
}

// Transaction representa uma transação em uma chain compatível com EVM
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

// Address representa um endereço com metadados adicionais
type Address struct {
    Address  string `json:"address"`
    Name     string `json:"name,omitempty"`
    Symbol   string `json:"symbol,omitempty"`
    LogoURI  string `json:"logoUri,omitempty"`
    Decimals int    `json:"decimals,omitempty"`
}

// Token representa um token com seus metadados
type Token struct {
    Address  string      `json:"address,omitempty"`
    Name     string      `json:"name"`
    Symbol   string      `json:"symbol"`
    Decimals int         `json:"decimals"`
    LogoURI  string      `json:"logoUri,omitempty"`
    Price    *TokenPrice `json:"price,omitempty"`
}

// TokenPrice representa informações de preço de um token
type TokenPrice struct {
    Value        float64 `json:"value"`
    CurrencyCode string  `json:"currencyCode"`
}

// ERC20Transfer representa uma transferência de token ERC-20
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

// ERC721Transfer representa uma transferência de token ERC-721 (NFT)
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

// ERC1155Transfer representa uma transferência de token ERC-1155
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

// InternalTransaction representa uma transação interna
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
