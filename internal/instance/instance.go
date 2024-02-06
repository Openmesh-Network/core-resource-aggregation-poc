package instance

import (
	"context"
	"log"

	"openmesh.network/aggregationpoc/internal/api"
	"openmesh.network/aggregationpoc/internal/gossip"
	"openmesh.network/aggregationpoc/internal/ipfs"
	"openmesh.network/aggregationpoc/internal/p2p"
)

// Instance is the top-level instance of the whole poc project
type Instance struct {
	Gossip *gossip.Instance  // Member management and data sharing
	HTTP   *api.HTTPInstance // HTTP RESTful APIs and WebSockets
	Ipfs   *ipfs.Instance
	P2P    *p2p.Instance // Libp2p instance
}

// NewInstance create all the low-level instances, then create the top-level instance
func NewInstance(instanceName, groupName string, gossipPort int, httpPort int, p2pPort int) *Instance {
	gi := gossip.NewInstance(instanceName, gossipPort)
	ii := ipfs.NewInstance()

	h := api.NewHTTPInstance(httpPort, ii)

	// This is the default branch
	doP2pWithIPFS := true
	var pi *p2p.Instance
	if doP2pWithIPFS {
		pi = p2p.NewLibP2PInstance(p2pPort, groupName, &ii.Host)
	} else {
		pi = p2p.NewLibP2PInstance(p2pPort, groupName, nil)

	}

	return &Instance{
		Gossip: gi,
		HTTP:   h,
		Ipfs:   ii,
		P2P:    pi,
	}
}

// Start starts all the instances, then start the top-level instance
func (i *Instance) Start(ctx context.Context, gossipPeers []string, httpPeers []string) {

	log.Println("Running http!!")
	i.HTTP.Start()

	// NOTE(Tom): I've disabled gossip for now since I'm not using it and it crowds the logs
	// i.Gossip.Start(ctx, gossipPeers, i.Ipfs)

	log.Printf("Running ipfs!!\n")
	// XXX: Ipfs.Start is blocking for now. I'm very silly
	i.Ipfs.Start(ctx, httpPeers)

	err := i.P2P.Start(ctx)
	if err != nil {
		log.Fatalf("Failed to start libp2p instance: %s", err.Error())
	}
}
