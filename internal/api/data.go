package api

import (
    "encoding/json"
    "fmt"
    "log"
    "net/url"
    "strconv"

    "github.com/Panorama-Block/avax/internal/types"
)

func (c *Client) GetChains() ([]types.Chain, error) {
    body, err := c.makeRequest("/chains")
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

func (c *Client) GetChainByID(chainID string) (*types.Chain, error) {
    body, err := c.makeRequest("/chains/" + chainID)
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

func (c *Client) GetBlocks(chainID string, pageSize int, pageToken string) ([]types.Block, string, error) {
    params := url.Values{}
    params.Set("chainId", chainID)
    if pageSize > 0 {
        params.Set("pageSize", strconv.Itoa(pageSize))
    }
    if pageToken != "" {
        params.Set("pageToken", pageToken)
    }
    respBody, err := c.makeRequest("/blocks?" + params.Encode())
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

func (c *Client) GetBlockByNumberOrHash(chainID, blockIdentifier string) (*types.Block, error) {
    endpoint := fmt.Sprintf("/chains/%s/blocks/%s", chainID, blockIdentifier)
    respBody, err := c.makeRequest(endpoint)
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

// func (c *Client) GetTransactions(chainID string, filter map[string]string) ([]types.Transaction, string, error) {
//     params := url.Values{}
//     params.Set("chainId", chainID)
//     for k, v := range filter {
//         params.Set(k, v)
//     }
//     endpoint := "/transactions?" + params.Encode()
//     respBody, err := c.makeRequest(endpoint)
//     if err != nil {
//         return nil, "", err
//     }

//     // Usa o wrapper
//     var resp types.TransactionsResponse
//     if err := json.Unmarshal(respBody, &resp); err != nil {
//         log.Printf("Erro decodificando transações chain %s: %v", chainID, err)
//         return nil, "", err
//     }
//     return resp.Transactions, resp.NextPageToken, nil
// }

// // GET /transactions/{chainId}/{txHash}
// func (c *Client) GetTransactionByHash(chainID, txHash string) (*types.Transaction, error) {
//     endpoint := fmt.Sprintf("/transactions/%s/%s", chainID, txHash)
//     body, err := c.makeRequest(endpoint)
//     if err != nil {
//         log.Printf("Erro ao buscar TX %s (chain %s): %v", txHash, chainID, err)
//         return nil, err
//     }
//     var transaction types.Transaction
//     if err := json.Unmarshal(body, &transaction); err != nil {
//         log.Printf("Erro decodificando TX %s: %v", txHash, err)
//         return nil, err
//     }
//     return &transaction, nil
// }

func (c *Client) GetValidators(network string) ([]types.Validator, error) {
    endpoint := fmt.Sprintf("/networks/%s/validators", network)
    body, err := c.makeRequest(endpoint)
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

func (c *Client) GetValidatorByNodeID(network, nodeID string) (*types.Validator, error) {
    endpoint := fmt.Sprintf("/networks/%s/validators/%s", network, nodeID)
    body, err := c.makeRequest(endpoint)
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

func (c *Client) GetDelegators(network string) ([]types.Delegator, error) {
    endpoint := fmt.Sprintf("/networks/%s/delegators", network)
    body, err := c.makeRequest(endpoint)
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

func (c *Client) GetSubnets(network string) ([]types.Subnet, error) {
    endpoint := fmt.Sprintf("/networks/%s/subnets", network)
    body, err := c.makeRequest(endpoint)
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

func (c *Client) GetSubnetByID(network, subnetID string) (*types.Subnet, error) {
    endpoint := fmt.Sprintf("/networks/%s/subnets/%s", network, subnetID)
    body, err := c.makeRequest(endpoint)
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

func (c *Client) GetBlockchains(network string) ([]types.Blockchain, error) {
    endpoint := fmt.Sprintf("/networks/%s/blockchains", network)
    body, err := c.makeRequest(endpoint)
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

func (c *Client) GetTeleporterTxs(sourceChain, destChain string) ([]types.TeleporterTx, error) {
    endpoint := fmt.Sprintf("/teleporter/transactions?sourceChain=%s&destChain=%s", sourceChain, destChain)
    body, err := c.makeRequest(endpoint)
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

func (c *Client) GetTokenDetails(tokenAddress string) (*types.TokenInfo, error) {
    endpoint := fmt.Sprintf("/assets/%s", tokenAddress)
    body, err := c.makeRequest(endpoint)
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
