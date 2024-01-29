package main

import (
    "context"
    "log"
    "openmesh.network/aggregationpoc/internal/instance"
    "os"
    "os/signal"
    "strconv"
    "strings"
    "syscall"
    "time"
)

func main() {
    // Read instance name, gossip port, and known peers from the environment
    // XNODE_NAME: string
    peerName := os.Getenv("XNODE_NAME")
    if peerName == "" {
        peerName = "Xnode-1"
    }
    // XNODE_GOSSIP_PORT: number
    gossipPort, _ := strconv.Atoi(os.Getenv("XNODE_GOSSIP_PORT"))
    if gossipPort == 0 {
        gossipPort = 9090
    }
    // XNODE_KNOWN_PEERS: addresses split by comma (,)
    // e.g., 127.0.0.1:9090,127.0.0.1:9091
    knownPeersString := os.Getenv("XNODE_KNOWN_PEERS")
    var knownPeers []string
    if knownPeersString != "" {
        knownPeers = strings.Split(knownPeersString, ",")
    }

    // Initialise graceful shutdown
    cancelCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    // Initialise and start the instance
    pocInstance := instance.NewInstance(peerName, gossipPort)
    pocInstance.Start(cancelCtx, knownPeers)

    // Stop here!
    sig := <-sigChan
    log.Printf("Termination signal received: %v", sig)

    // Cleanup
    if err := pocInstance.Gossip.Leave(); err != nil {
        log.Printf("Failed to leave the cluster: %s", err.Error())
    }
}
