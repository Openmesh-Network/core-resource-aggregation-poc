package instance

import (
    "context"
    "log"
    "openmesh.network/aggregationpoc/internal/api"
    "openmesh.network/aggregationpoc/internal/gossip"
    "openmesh.network/aggregationpoc/internal/p2p"
)

// Instance is the top-level instance of the whole poc project
type Instance struct {
    Gossip *gossip.Instance  // Member management and data sharing
    HTTP   *api.HTTPInstance // HTTP RESTful APIs and WebSockets
    P2P    *p2p.Instance     // Libp2p instance
}

// NewInstance create all the low-level instances, then create the top-level instance
func NewInstance(instanceName, groupName string, gossipPort, httpPort, p2pPort int) *Instance {
    gi := gossip.NewInstance(instanceName, gossipPort)
    h := api.NewHTTPInstance(httpPort)
    pi := p2p.NewLibP2PInstance(p2pPort, groupName)
    return &Instance{
        Gossip: gi,
        HTTP:   h,
        P2P:    pi,
    }
}

// Start starts all the instances, then start the top-level instance
func (i *Instance) Start(ctx context.Context, knownPeers []string) {
    i.Gossip.Start(ctx, knownPeers)
    i.HTTP.Start()
    err := i.P2P.Start(ctx)
    if err != nil {
        log.Fatalf("Failed to start libp2p instance: %s", err.Error())
    }
}
