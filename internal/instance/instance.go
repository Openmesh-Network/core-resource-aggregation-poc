package instance

import (
	"context"
	"log"

	"openmesh.network/aggregationpoc/internal/api"
	"openmesh.network/aggregationpoc/internal/gossip"
	"openmesh.network/aggregationpoc/internal/ipfs"
)

// Instance is the top-level instance of the whole poc project
type Instance struct {
	Gossip *gossip.Instance  // Member management and data sharing
	HTTP   *api.HTTPInstance // HTTP RESTful APIs and WebSockets
	Ipfs   *ipfs.Instance
}

// NewInstance create all the low-level instances, then create the top-level instance
func NewInstance(instanceName string, gossipPort int, httpPort int) *Instance {
	gi := gossip.NewInstance(instanceName, gossipPort)
	ii := ipfs.NewInstance()

	h := api.NewHTTPInstance(httpPort, ii)

	return &Instance{
		Gossip: gi,
		HTTP:   h,
		Ipfs:   ii,
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

}
