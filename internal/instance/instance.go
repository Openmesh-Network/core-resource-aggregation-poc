package instance

import (
    "context"
    "openmesh.network/aggregationpoc/internal/api"
    "openmesh.network/aggregationpoc/internal/gossip"
)

// Instance is the top-level instance of the whole poc project
type Instance struct {
    Gossip *gossip.Instance  // Member management and data sharing
    HTTP   *api.HTTPInstance // HTTP RESTful APIs and WebSockets
}

// NewInstance create all the low-level instances, then create the top-level instance
func NewInstance(instanceName string, gossipPort int, httpPort int) *Instance {
    gi := gossip.NewInstance(instanceName, gossipPort)
    h := api.NewHTTPInstance(httpPort)
    return &Instance{
        Gossip: gi,
        HTTP:   h,
    }
}

// Start starts all the instances, then start the top-level instance
func (i *Instance) Start(ctx context.Context, knownPeers []string) {
    i.Gossip.Start(ctx, knownPeers)
    i.HTTP.Start()
}
