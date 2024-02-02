package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"openmesh.network/aggregationpoc/internal/instance"
)

func main() {
	// Read instance name, gossip port, and known peers from the environment
	// XNODE_NAME: string

	log.Println("Calling gossip peers")

	peerName := os.Getenv("XNODE_NAME")
	if peerName == "" {
		peerName = "Xnode-1"
	}
	// XNODE_GOSSIP_PORT: number
	gossipPort, _ := strconv.Atoi(os.Getenv("XNODE_GOSSIP_PORT"))
	if gossipPort == 0 {
		gossipPort = 9090
	}
	// XNODE_XXXX_PEERS: addresses split by comma (,)
	// e.g., 127.0.0.1:9090,127.0.0.1:9091
	gossipPeers := make([]string, 0)
	httpPeers := make([]string, 0)

	// XNODE_HTTP_PORT: number
	httpPort, _ := strconv.Atoi(os.Getenv("XNODE_HTTP_PORT"))
	if httpPort == 0 {
		httpPort = 9080
	}

	{
		name := os.Getenv("XNODE_NAME")
		log.Println("My name is:", name)
		gossipPeersString := os.Getenv("XNODE_GOSSIP_PEERS")
		if gossipPeersString != "" {
			gossipPeers = strings.Split(gossipPeersString, ",")
		}
		log.Println("Got", len(gossipPeers), "peers", gossipPeers, "from:", gossipPeersString)

		httpPeers = make([]string, len(gossipPeers))

		for i, g := range gossipPeers {
			// NOTE(Tom): This assumes all peers have the SAME http port
			httpPeers[i] = strings.Split(g, ":")[0] + ":" + strconv.Itoa(httpPort)
		}
	}

	// Initialise graceful shutdown
	cancelCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer log.Println("Calling cancel")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	// Initialise and start the instance
	pocInstance := instance.NewInstance(peerName, gossipPort, httpPort)
	pocInstance.Start(cancelCtx, gossipPeers, httpPeers)

	// Stop here!
	sig := <-sigChan
	log.Printf("Termination signal received: %v", sig)

	// Cleanup
	if err := pocInstance.Gossip.Leave(); err != nil {
		log.Printf("Failed to leave the cluster: %s", err.Error())
	}
	pocInstance.HTTP.Stop()
}
