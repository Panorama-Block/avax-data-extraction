package extractor

import (
    "log"
    "time"

    "github.com/Panorama-Block/avax/internal/api"
    "github.com/Panorama-Block/avax/internal/kafka"
)

func StartCronJobs(client *api.Client, producer *kafka.Producer, network string) {
    go func() {
        for {
            err := fetchSubnetsAndBlockchains(client, producer, network)
            if err != nil {
                log.Printf("[CRON] Erro subnets: %v", err)
            }

            err = fetchValidators(client, producer, network)
            if err != nil {
                log.Printf("[CRON] Erro validators: %v", err)
            }

            err = fetchDelegators(client, producer, network)
            if err != nil {
                log.Printf("[CRON] Erro delegators: %v", err)
            }

            _ = fetchTeleporterTxs(client, producer)

            time.Sleep(10 * time.Minute)
        }
    }()
}

func fetchSubnetsAndBlockchains(client *api.Client, producer *kafka.Producer, network string) error {
    subnets, err := client.GetSubnets(network)
    if err != nil {
        return err
    }
    for _, s := range subnets {
        producer.PublishSubnet(s)
        det, err2 := client.GetSubnetByID(network, s.SubnetID)
        if err2 == nil && det != nil {
            producer.PublishSubnet(*det)
            for _, bc := range det.Blockchains {
                producer.PublishBlockchain(bc)
            }
        }
    }
    bcs, err := client.GetBlockchains(network)
    if err == nil {
        for _, bc := range bcs {
            producer.PublishBlockchain(bc)
        }
    }
    return nil
}

func fetchValidators(client *api.Client, producer *kafka.Producer, network string) error {
    vals, err := client.GetValidators(network)
    if err != nil {
        return err
    }
    for _, v := range vals {
        producer.PublishValidator(v)
    }
    return nil
}

func fetchDelegators(client *api.Client, producer *kafka.Producer, network string) error {
    delegs, err := client.GetDelegators(network)
    if err != nil {
        return err
    }
    for _, d := range delegs {
        producer.PublishDelegator(d)
    }
    return nil
}

func fetchTeleporterTxs(client *api.Client, producer *kafka.Producer) error {
    txs, err := client.GetTeleporterTxs("43114", "99999")
    if err != nil {
        log.Printf("[CRON] Erro teleporter TXs: %v", err)
        return err
    }
    for _, t := range txs {
        producer.PublishBridgeTx(t)
    }
    return nil
}