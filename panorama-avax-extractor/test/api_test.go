package test

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/Panorama-Block/avax/internal/api"
    "github.com/stretchr/testify/assert"
)

// Simulação do servidor HTTP para testes
func TestGetChains(t *testing.T) {
    mockResponse := `{
        "chains": [
            {
                "chainId": "1",
                "status": "OK",
                "chainName": "Avalanche",
                "description": "Avalanche Mainnet",
                "platformChainId": "P-Chain",
                "subnetId": "default",
                "vmId": "evm",
                "vmName": "EVM",
                "explorerUrl": "https://explorer.avax.network",
                "rpcUrl": "https://api.avax.network/ext/bc/C/rpc",
                "wsUrl": "wss://api.avax.network/ext/bc/C/ws",
                "isTestnet": false,
                "private": false,
                "chainLogoUri": "https://logo.png",
                "enabledFeatures": ["nftIndexing"],
                "networkToken": {
                    "name": "Wrapped AVAX",
                    "symbol": "WAVAX",
                    "decimals": 18,
                    "logoUri": "https://logo.png",
                    "description": "Wrapped AVAX Token"
                }
            }
        ]
    }`

    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(mockResponse))
    }))
    defer server.Close()

    client := api.NewClient(server.URL, "test-api-ac_wNQeDAeEFZJUcMsnOwHoH1xrJisi-0VgU47ag-Xriz-huJMkGyJd0pg3p7vgzsNCuXQ1WfYjyK6vDht5pCbtig")
    dataAPI := api.NewDataAPI(client)
    chains, err := dataAPI.GetChains()


    assert.NoError(t, err)
    assert.NotEmpty(t, chains)
    assert.Equal(t, "Avalanche", chains[0].ChainName)
    assert.Equal(t, "1", chains[0].ChainID)
    assert.Equal(t, "OK", chains[0].Status)
}
